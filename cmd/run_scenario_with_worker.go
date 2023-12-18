package main

import (
	"context"
	"fmt"
	"time"

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
	iterations      int
	duration        time.Duration
	maxConcurrent   int
	scenarioOptions []string
	metricsOptions  cmdoptions.MetricsOptions
}

func (r *workerWithScenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.workerRunner.addCLIFlags(fs)
	fs.IntVar(&r.iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)")
	fs.IntVar(&r.maxConcurrent, "max-concurrent", 0, "Override max-concurrent for the scenario")
	fs.StringSliceVar(&r.scenarioOptions, "option", nil, "Additional options for the scenario, in key=value format")
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
	scenarioRunner := cmdoptions.ScenarioRunner{
		Logger:          r.logger,
		Scenario:        r.scenario,
		RunID:           r.runID,
		Iterations:      r.iterations,
		Duration:        r.duration,
		MaxConcurrent:   r.maxConcurrent,
		ScenarioOptions: r.scenarioOptions,
		ClientOptions:   r.clientOptions,
		MetricsOptions:  r.metricsOptions,
		LoggingOptions:  r.loggingOptions,
	}
	scenarioErr := scenarioRunner.Run(ctx)
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
