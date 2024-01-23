package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
)

func runScenarioWithWorkerCmd() *cobra.Command {
	var r workerWithScenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario-with-worker",
		Short: "Run a worker and a scenario",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := r.run(ctx); err != nil {
				r.logger.Fatal(err)
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
	metricsOptions cmdoptions.MetricsOptions
}

func (r *workerWithScenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.workerRunner.addCLIFlags(fs)
	r.scenarioRunConfig.addCLIFlags(fs)
	r.metricsOptions.AddCLIFlags(fs, "")
}

func (r *workerWithScenarioRunner) run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Start worker and wait on error or started
	workerErrCh := make(chan error, 1)
	workerStartCh := make(chan struct{})
	r.onWorkerStarted = func() { close(workerStartCh) }
	go func() { workerErrCh <- r.workerRunner.run(ctx) }()
	select {
	case err := <-workerErrCh:
		return fmt.Errorf("worker did not start: %w", err)
	case <-workerStartCh:
	}

	// Run scenario
	scenarioRunner := scenarioRunner{
		logger: r.logger,
		scenarioID: scenarioID{
			scenario: r.scenario,
			runID:    r.runID,
		},
		scenarioRunConfig: scenarioRunConfig{
			iterations:      r.iterations,
			duration:        r.duration,
			maxConcurrent:   r.maxConcurrent,
			scenarioOptions: r.scenarioOptions,
		},
		clientOptions:  r.clientOptions,
		metricsOptions: r.metricsOptions,
		loggingOptions: r.loggingOptions,
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
