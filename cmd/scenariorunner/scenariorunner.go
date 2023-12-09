package scenariorunner

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type ScenarioRunner struct {
	Logger          *zap.SugaredLogger
	Scenario        string
	RunID           string
	Iterations      int
	Duration        time.Duration
	MaxConcurrent   int
	ScenarioOptions []string
	ConnectTimeout  time.Duration
	ClientOptions   cmdoptions.ClientOptions
	MetricsOptions  cmdoptions.MetricsOptions
	LoggingOptions  cmdoptions.LoggingOptions
}

func (r *ScenarioRunner) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&r.Scenario, "scenario", "", "Scenario name to run")
	fs.StringVar(&r.RunID, "run-id", "", "Run ID for this run")
	fs.IntVar(&r.Iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.Duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)")
	fs.IntVar(&r.MaxConcurrent, "max-concurrent", 0, "Override max-concurrent for the scenario")
	fs.StringSliceVar(&r.ScenarioOptions, "option", nil, "Additional options for the scenario, in key=value format")
	fs.DurationVar(&r.ConnectTimeout, "connect-timeout", 0, "Duration to try to connect to server before failing")
	r.ClientOptions.AddCLIFlags(fs)
	r.MetricsOptions.AddCLIFlags(fs, "")
	r.LoggingOptions.AddCLIFlags(fs)
}

func (r *ScenarioRunner) Run(ctx context.Context) error {
	if r.Logger == nil {
		r.Logger = r.LoggingOptions.MustCreateLogger()
	}
	scenario := loadgen.GetScenario(r.Scenario)
	if scenario == nil {
		return fmt.Errorf("scenario not found")
	} else if r.RunID == "" {
		return fmt.Errorf("run ID not found")
	} else if r.Iterations > 0 && r.Duration > 0 {
		return fmt.Errorf("cannot provide both iterations and duration")
	}

	// Parse options
	scenarioOptions := make(map[string]string, len(r.ScenarioOptions))
	for _, v := range r.ScenarioOptions {
		pieces := strings.SplitN(v, "=", 2)
		if len(pieces) != 2 {
			return fmt.Errorf("option does not have '='")
		}
		scenarioOptions[pieces[0]] = pieces[1]
	}

	metrics := r.MetricsOptions.MustCreateMetrics(r.Logger)
	defer metrics.Shutdown(ctx)
	start := time.Now()
	var client client.Client
	var err error
	for {
		client, err = r.ClientOptions.Dial(metrics, r.Logger)
		if err == nil {
			break
		}
		// Only fail if past wait period
		if time.Since(start) > r.ConnectTimeout {
			return fmt.Errorf("failed dialing: %w", err)
		}
		// Wait 300ms and try again
		time.Sleep(300 * time.Millisecond)
	}
	defer client.Close()
	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   r.Scenario,
		RunID:          r.RunID,
		Logger:         r.Logger,
		MetricsHandler: metrics.NewHandler(),
		Client:         client,
		Configuration: loadgen.RunConfiguration{
			Iterations:    r.Iterations,
			Duration:      r.Duration,
			MaxConcurrent: r.MaxConcurrent,
		},
		ScenarioOptions: scenarioOptions,
		Namespace:       r.ClientOptions.Namespace,
		RootPath:        rootDir(),
	}
	err = scenario.Executor.Run(ctx, scenarioInfo)
	if err != nil {
		return fmt.Errorf("failed scenario: %w", err)
	}
	return nil
}

func rootDir() string {
	_, currFile, _, _ := runtime.Caller(0)
	return filepath.Dir(filepath.Dir(currFile))
}
