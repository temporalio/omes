package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workerctl"
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
			exitOnError(r.Logger, r.run(ctx))
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

	// Reject bad input before building and starting a worker, which for some
	// languages takes minutes. Judging that needs only the scenario name and run
	// config, so this throwaway runner deliberately carries no client options.
	if _, _, err := (&scenarioRunner{
		scenario:          r.ScenarioID,
		scenarioRunConfig: r.scenarioRunConfig,
	}).validateInput(); err != nil {
		return err
	}

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
		workerErrCh <- r.Run(ctx, workerctl.BaseDir(repoDir, r.SdkOptions.Language))
	}()
	select {
	case err := <-workerErrCh:
		return fmt.Errorf("worker did not start: %w", err)
	case <-workerStartCh:
	}

	// Run scenario. The client options are read only now that the worker has
	// started, since starting an embedded server rewrites the server address.
	scenarioRunner := scenarioRunner{
		logger:            r.Logger,
		scenario:          r.ScenarioID,
		scenarioRunConfig: r.scenarioRunConfig,
		clientOptions:     r.ClientOptions,
		metricsOptions:    r.metricsOptions,
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
