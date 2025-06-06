package main

import (
	"context"
	"fmt"
	"os"
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
	scenarioID
	scenarioRunConfig
	logger         *zap.SugaredLogger
	connectTimeout time.Duration
	clientOptions  cmdoptions.ClientOptions
	metricsOptions cmdoptions.MetricsOptions
	loggingOptions cmdoptions.LoggingOptions
}

type scenarioID struct {
	scenario string
	runID    string
}

type scenarioRunConfig struct {
	iterations                    int
	duration                      time.Duration
	maxConcurrent                 int
	maxIterationsPerSecond        float64
	scenarioOptions               []string
	timeout                       time.Duration
	doNotRegisterSearchAttributes bool
}

func (r *scenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.scenarioID.addCLIFlags(fs)
	r.scenarioRunConfig.addCLIFlags(fs)
	fs.DurationVar(&r.connectTimeout, "connect-timeout", 0, "Duration to try to connect to server before failing")
	r.clientOptions.AddCLIFlags(fs)
	r.metricsOptions.AddCLIFlags(fs, "")
	r.loggingOptions.AddCLIFlags(fs)
}

func (r *scenarioID) addCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.runID, "run-id", "", "Run ID for this run")
}

func (r *scenarioRunConfig) addCLIFlags(fs *pflag.FlagSet) {
	fs.IntVar(&r.iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)."+
		" This is the amount of time for which we will start new iterations of the scenario.")
	fs.Float64Var(&r.maxIterationsPerSecond, "max-iterations-per-second", 0, "Override iterations per second rate limit for the scenario."+
		" This is the maximum rate at which we will start new iterations of the scenario.")
	fs.DurationVar(&r.timeout, "timeout", 0, "If set, the scenario will stop after this amount of"+
		" time has elapsed. Any still-running iterations will be cancelled, and omes will exit nonzero.")
	fs.IntVar(&r.maxConcurrent, "max-concurrent", 0, "Override max-concurrent for the scenario")
	fs.StringSliceVar(&r.scenarioOptions, "option", nil, "Additional options for the scenario, in key=value format")
	fs.BoolVar(&r.doNotRegisterSearchAttributes, "do-not-register-search-attributes", false,
		"Do not register the default search attributes used by scenarios. "+
			"If the search attributes are not registed by the scenario they must be registered through some other method")
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
		key, value := pieces[0], pieces[1]

		// If the value starts with '@', read the file and use its contents as the value.
		if strings.HasPrefix(value, "@") {
			filePath := strings.TrimPrefix(value, "@")
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", filePath, err)
			}
			value = string(data)
		}
		scenarioOptions[key] = value

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
			Iterations:                    r.iterations,
			Duration:                      r.duration,
			MaxConcurrent:                 r.maxConcurrent,
			MaxIterationsPerSecond:        r.maxIterationsPerSecond,
			Timeout:                       r.timeout,
			DoNotRegisterSearchAttributes: r.doNotRegisterSearchAttributes,
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
