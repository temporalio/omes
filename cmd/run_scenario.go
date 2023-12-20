package main

import (
	"context"
	"fmt"
	"strings"
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
	logger          *zap.SugaredLogger
	scenario        string
	runID           string
	iterations      int
	duration        time.Duration
	maxConcurrent   int
	scenarioOptions []string
	connectTimeout  time.Duration
	clientOptions   cmdoptions.ClientOptions
	metricsOptions  cmdoptions.MetricsOptions
	loggingOptions  cmdoptions.LoggingOptions
}

func (r *scenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.runID, "run-id", "", "Run ID for this run")
	fs.IntVar(&r.iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)")
	fs.IntVar(&r.maxConcurrent, "max-concurrent", 0, "Override max-concurrent for the scenario")
	fs.StringSliceVar(&r.scenarioOptions, "option", nil, "Additional options for the scenario, in key=value format")
	fs.DurationVar(&r.connectTimeout, "connect-timeout", 0, "Duration to try to connect to server before failing")
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

	// Parse options
	scenarioOptions := make(map[string]string, len(r.scenarioOptions))
	for _, v := range r.scenarioOptions {
		pieces := strings.SplitN(v, "=", 2)
		if len(pieces) != 2 {
			return fmt.Errorf("option does not have '='")
		}
		scenarioOptions[pieces[0]] = pieces[1]
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
		if time.Since(start) > r.connectTimeout {
			return fmt.Errorf("failed dialing: %w", err)
		}
		// Wait 300ms and try again
		time.Sleep(300 * time.Millisecond)
	}
	defer client.Close()
	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   r.scenario,
		RunID:          r.runID,
		Logger:         r.logger,
		MetricsHandler: metrics.NewHandler(),
		Client:         client,
		Configuration: loadgen.RunConfiguration{
			Iterations:    r.iterations,
			Duration:      r.duration,
			MaxConcurrent: r.maxConcurrent,
		},
		ScenarioOptions: scenarioOptions,
		Namespace:       r.clientOptions.Namespace,
		RootPath:        rootDir(),
	}
	err = scenario.Executor.Run(ctx, scenarioInfo)
	if err != nil {
		return fmt.Errorf("failed scenario: %w", err)
	}
	return nil
}
