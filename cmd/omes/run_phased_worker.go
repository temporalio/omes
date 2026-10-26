package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/internal/workerctl"
	"github.com/temporalio/omes/loadgen/phaseplan"
)

func runPhasedWorkerCmd() *cobra.Command {
	var r phasedWorkerRunner
	cmd := &cobra.Command{
		Use:   "run-phased-worker",
		Short: "Run OMES workers according to a synchronized phase plan",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := applyEnvFallbacks(cmd.Flags(), "OMES_"); err != nil {
				return err
			}
			if err := requireFlagsOrEnv(cmd.Flags(), "OMES_", "language", "run-id", "dir-name", "phase-plan-file", "start-at"); err != nil {
				return err
			}
			if cmd.Flags().Changed("worker-profile") {
				return fmt.Errorf("worker-profile is defined by each phase; do not pass --worker-profile")
			}
			data, err := os.ReadFile(r.phasePlanFile)
			if err != nil {
				return fmt.Errorf("read phase plan: %w", err)
			}
			r.plan, err = phaseplan.Parse(string(data))
			if err != nil {
				return fmt.Errorf("parse phase plan: %w", err)
			}
			r.startAt, err = time.Parse(time.RFC3339Nano, r.startAtValue)
			if err != nil {
				return fmt.Errorf("invalid --start-at %q: %w", r.startAtValue, err)
			}
			r.preRun()
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.runPhases(ctx); err != nil {
				r.Logger.Fatal(err)
			}
		},
	}
	r.addCLIFlags(cmd.Flags())
	return cmd
}

type phasedWorkerRunner struct {
	workerRunner
	phasePlanFile string
	startAtValue  string
	startAt       time.Time
	plan          *phaseplan.Plan
}

type runningPhasedWorker struct {
	profile string
	cancel  context.CancelFunc
	errCh   <-chan error
}

func (r *phasedWorkerRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.workerRunner.addCLIFlags(fs)
	fs.Lookup("worker-profile").Hidden = true
	fs.StringVar(&r.phasePlanFile, "phase-plan-file", "", "Path to the shared phased activity load plan")
	fs.StringVar(&r.startAtValue, "start-at", "", "RFC3339 time at which phase zero begins")
}

func (r *phasedWorkerRunner) runPhases(ctx context.Context) error {
	if err := r.validatePreparedWorker(); err != nil {
		return err
	}
	r.Logger.Infof("Waiting for synchronized phase-plan start: start_at=%s",
		r.startAt.UTC().Format(time.RFC3339Nano))
	if err := phaseplan.WaitForStart(ctx, r.startAt); err != nil {
		return err
	}

	var active *runningPhasedWorker
	for index, phase := range r.plan.Phases {
		scheduledStart := r.plan.PhaseStart(r.startAt, index)
		if err := phaseplan.WaitUntil(ctx, scheduledStart); err != nil {
			return err
		}
		actualStart := time.Now()
		r.Logger.Infof(
			"Worker phase started: index=%d name=%q worker_profile=%q scheduled_boundary=%s "+
				"actual_boundary=%s lateness=%v",
			index, phase.Name, workerProfileName(phase.WorkerProfile),
			scheduledStart.UTC().Format(time.RFC3339Nano), actualStart.UTC().Format(time.RFC3339Nano),
			max(actualStart.Sub(scheduledStart), 0),
		)

		if !sameWorkerProfile(active, phase.WorkerProfile) {
			if err := r.stopActiveWorker(active); err != nil {
				return fmt.Errorf("stop worker for phase %q: %w", phase.Name, err)
			}
			active = nil
			if phase.WorkerProfile != nil {
				var err error
				active, err = r.startWorker(ctx, *phase.WorkerProfile)
				if err != nil {
					return fmt.Errorf("start worker for phase %q: %w", phase.Name, err)
				}
			}
		}

		if err := waitForWorkerPhase(ctx, r.plan.PhaseEnd(r.startAt, index), active); err != nil {
			return fmt.Errorf("worker phase %q failed: %w", phase.Name, err)
		}
		phaseCompleted := time.Now()
		scheduledEnd := r.plan.PhaseEnd(r.startAt, index)
		r.Logger.Infof(
			"Worker phase completed: index=%d name=%q scheduled_boundary=%s actual_boundary=%s lateness=%v",
			index, phase.Name, scheduledEnd.UTC().Format(time.RFC3339Nano),
			phaseCompleted.UTC().Format(time.RFC3339Nano), max(phaseCompleted.Sub(scheduledEnd), 0),
		)
	}
	if err := r.stopActiveWorker(active); err != nil {
		return fmt.Errorf("stop final worker: %w", err)
	}
	return nil
}

func (r *phasedWorkerRunner) startWorker(ctx context.Context, profile string) (*runningPhasedWorker, error) {
	workerCtx, cancel := context.WithCancel(ctx)
	startedCh := make(chan struct{})
	errCh := make(chan error, 1)
	r.WorkerProfile = profile
	r.OnWorkerStarted = func() { close(startedCh) }
	go func() {
		errCh <- r.workerRunner.run(workerCtx)
	}()
	select {
	case <-ctx.Done():
		cancel()
		return nil, ctx.Err()
	case err := <-errCh:
		cancel()
		return nil, fmt.Errorf("worker exited before reporting ready: %w", err)
	case <-startedCh:
		r.Logger.Infof("Worker process started: profile=%q process_started_at=%s",
			profile, time.Now().UTC().Format(time.RFC3339Nano))
		return &runningPhasedWorker{profile: profile, cancel: cancel, errCh: errCh}, nil
	}
}

func (r *phasedWorkerRunner) validatePreparedWorker() error {
	repoDir, err := getRepoDir()
	if err != nil {
		return fmt.Errorf("get repository root: %w", err)
	}
	baseDir := workerctl.BaseDir(repoDir, r.SdkOptions.Language)
	if r.ProjectName != "" {
		baseDir = workerctl.ProjectDir(baseDir, r.ProjectName)
	}
	loadDir := filepath.Join(baseDir, r.DirName)
	if _, err := workerctl.LoadProgramFromDir(loadDir, baseDir, r.SdkOptions.Language); err != nil {
		return fmt.Errorf("load prepared worker %q: %w", r.DirName, err)
	}
	return nil
}

func (r *phasedWorkerRunner) stopActiveWorker(active *runningPhasedWorker) error {
	if active == nil {
		return nil
	}
	active.cancel()
	if err := <-active.errCh; err != nil {
		return err
	}
	r.Logger.Infof("Worker stopped: profile=%q stopped_at=%s",
		active.profile, time.Now().UTC().Format(time.RFC3339Nano))
	return nil
}

func waitForWorkerPhase(ctx context.Context, endsAt time.Time, active *runningPhasedWorker) error {
	delay := max(time.Until(endsAt), 0)
	timer := time.NewTimer(delay)
	defer timer.Stop()
	if active == nil {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			return nil
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-active.errCh:
		return fmt.Errorf("worker exited unexpectedly: %w", err)
	case <-timer.C:
		return nil
	}
}

func sameWorkerProfile(active *runningPhasedWorker, desired *string) bool {
	if active == nil || desired == nil {
		return active == nil && desired == nil
	}
	return active.profile == *desired
}

func workerProfileName(profile *string) string {
	if profile == nil {
		return "Pz"
	}
	return *profile
}
