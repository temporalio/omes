package main

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func runScenarioCmd() *cobra.Command {
	var r scenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario",
		Short: "Run scenario",
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
	cmd.MarkFlagRequired("run-id")
	return cmd
}

type scenarioRunner struct {
	logger         *zap.SugaredLogger
	scenario       string
	runID          string
	iterations     int
	duration       time.Duration
	waitForServer  time.Duration
	clientOptions  cmdoptions.ClientOptions
	metricsOptions cmdoptions.MetricsOptions
	loggingOptions cmdoptions.LoggingOptions
}

func (r *scenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.runID, "run-id", "", "Run ID for this run")
	fs.IntVar(&r.iterations, "iterations", 0, "Iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Duration for the scenario (cannot be provided with iteration)")
	fs.DurationVar(&r.waitForServer, "wait-for-server", 0, "Duration to try to connect to server before failing")
	r.clientOptions.AddCLIFlags(fs)
	r.metricsOptions.AddCLIFlags(fs, "")
	r.loggingOptions.AddCLIFlags(fs)
}

func (r *scenarioRunner) run(ctx context.Context) error {
	if r.logger == nil {
		r.logger = r.loggingOptions.MustCreateLogger()
	}
	scenario := loadgen.GetScenario(r.scenario)
	if scenario == nil {
		return fmt.Errorf("scenario not found")
	} else if r.runID == "" {
		return fmt.Errorf("run ID not found")
	} else if r.iterations > 0 && r.duration > 0 {
		return fmt.Errorf("cannot provide both iterations and duration")
	}
	// Configure executor if iterations or duration is set
	if r.iterations > 0 || r.duration > 0 {
		iterExec, _ := scenario.Executor.(*loadgen.SharedIterationsExecutor)
		if iterExec == nil {
			return fmt.Errorf("this scenario does not support iterations or duration customization")
		}
		iterExec.Iterations = r.iterations
		iterExec.Duration = r.duration
	}

	metrics := r.metricsOptions.MustCreateMetrics(r.logger)
	defer metrics.Shutdown(ctx)
	start := time.Now()
	var client client.Client
	var err error
	for {
		client, err = r.clientOptions.Dial(metrics, r.logger)
		if err == nil {
			break
		}
		// Only fail if past wait period
		if time.Since(start) > r.waitForServer {
			return fmt.Errorf("failed dialing: %w", err)
		}
		// Wait 300ms and try again
		time.Sleep(300 * time.Millisecond)
	}
	defer client.Close()
	return scenario.Executor.Run(ctx, &loadgen.RunOptions{
		ScenarioName:   r.scenario,
		RunID:          r.runID,
		Logger:         r.logger,
		MetricsHandler: metrics.NewHandler(),
		Client:         client,
	})
}