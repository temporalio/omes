package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers"
)

func runScenarioWithWorkerCmd() *cobra.Command {
	var r workerWithScenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario-with-worker",
		Short: "Run a worker and a scenario",
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.run(ctx); err != nil {
				r.Logger.Fatal(err)
			}
		},
	}
	r.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("scenario")
	cmd.MarkFlagRequired("language")
	return cmd
}

type workerWithScenarioRunner struct {
	workerRunner
	scenarioRunConfig
	metricsOptions clioptions.MetricsOptions
}

func (r *workerWithScenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.workerRunner.addCLIFlags(fs)
	r.scenarioRunConfig.addCLIFlags(fs)
	fs.AddFlagSet(r.metricsOptions.FlagSet(""))
}

func (r *workerWithScenarioRunner) preRun() {
	r.workerRunner.preRun()
}

func (r *workerWithScenarioRunner) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Start worker and wait on error or started
	workerErrCh := make(chan error, 1)
	workerStartCh := make(chan struct{})
	r.OnWorkerStarted = func() { close(workerStartCh) }
	go func() {
		repoDir, err := getRepoDir()
		if err != nil {
			workerErrCh <- fmt.Errorf("failed to get root directory: %w", err)
			return
		}
		workerErrCh <- r.Run(ctx, workers.BaseDir(repoDir, r.SdkOptions.Language))
	}()
	select {
	case err := <-workerErrCh:
		return fmt.Errorf("worker did not start: %w", err)
	case <-workerStartCh:
	}

	// Run scenario
	scenarioRunner := scenarioRunner{
		logger:   r.Logger,
		scenario: r.ScenarioID,
		scenarioRunConfig: scenarioRunConfig{
			iterations:                    r.iterations,
			duration:                      r.duration,
			maxConcurrent:                 r.maxConcurrent,
			maxIterationsPerSecond:        r.maxIterationsPerSecond,
			scenarioOptions:               r.scenarioOptions,
			timeout:                       r.timeout,
			verificationTimeout:           r.verificationTimeout,
			doNotRegisterSearchAttributes: r.doNotRegisterSearchAttributes,
		},
		clientOptions:  r.ClientOptions,
		metricsOptions: r.metricsOptions,
	}
	scenarioErr := scenarioRunner.run(ctx)
	cancel()

	// Wait for worker complete
	workerErr := <-workerErrCh
	if scenarioErr != nil {
		if workerErr != nil {
			return fmt.Errorf("worker failed with: %v, scenario failed with: %w", workerErr, scenarioErr)
		}
		return fmt.Errorf("scenario failed: %w", scenarioErr)
	} else if workerErr != nil {
		return fmt.Errorf("worker failed: %w", workerErr)
	}
	return nil
}
