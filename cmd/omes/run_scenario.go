package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	iterationFailurePolicyContinue = "continue"
	iterationFailurePolicyFailFast = "fail-fast"
)

func runScenarioCmd() *cobra.Command {
	var r scenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario",
		Short: "Run scenario",
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			exitOnError(r.logger, r.run(ctx))
		},
	}
	r.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("scenario")
	cmd.MarkFlagRequired("run-id")
	return cmd
}

type scenarioRunner struct {
	scenarioRunConfig
	scenario       clioptions.ScenarioID
	logger         *zap.SugaredLogger
	connectTimeout time.Duration
	clientOptions  clioptions.ClientOptions
	metricsOptions clioptions.MetricsOptions
	loggingOptions clioptions.LoggingOptions
}

type scenarioRunConfig struct {
	iterations                    int
	duration                      time.Duration
	maxConcurrent                 int
	maxIterationsPerSecond        float64
	maxIterationAttempts          int
	iterationFailurePolicy        string
	scenarioOptions               []string
	timeout                       time.Duration
	doNotRegisterSearchAttributes bool
	ignoreAlreadyStarted          bool
	exportHistoriesDir            string
	exportHistoriesFilter         string
}

func (r *scenarioRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.scenario.AddCLIFlags(fs)
	r.scenarioRunConfig.addCLIFlags(fs)
	fs.DurationVar(&r.connectTimeout, "connect-timeout", 0, "Duration to try to connect to server before failing")
	fs.AddFlagSet(r.clientOptions.FlagSet())
	fs.AddFlagSet(r.metricsOptions.FlagSet(""))
	fs.AddFlagSet(r.loggingOptions.FlagSet())
}

func (r *scenarioRunConfig) addCLIFlags(fs *pflag.FlagSet) {
	fs.IntVar(&r.iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)."+
		" This is the amount of time for which we will start new iterations of the scenario.")
	fs.Float64Var(&r.maxIterationsPerSecond, "max-iterations-per-second", 0, "Override iterations per second rate limit for the scenario."+
		" This is the maximum rate at which we will start new iterations of the scenario.")
	fs.IntVar(&r.maxIterationAttempts, "max-iteration-attempts", 1, "Maximum attempts per iteration")
	fs.StringVar(&r.iterationFailurePolicy, "iteration-failure-policy", iterationFailurePolicyContinue,
		"How to handle terminal iteration failures: continue or fail-fast")
	fs.DurationVar(&r.timeout, "timeout", 0, "If set, the scenario will stop after this amount of"+
		" time has elapsed. Any still-running iterations will be cancelled, and omes will exit nonzero.")
	fs.IntVar(&r.maxConcurrent, "max-concurrent", 0, "Override max-concurrent for the scenario")
	fs.StringArrayVar(&r.scenarioOptions, "option", nil, "Option specific to the chosen scenario, in key=value format."+
		" Repeatable. Run 'omes list-scenarios' to see which options a scenario accepts, with their types and defaults")
	fs.BoolVar(&r.doNotRegisterSearchAttributes, "do-not-register-search-attributes", false,
		"Do not register the default search attributes used by scenarios. "+
			"If the search attributes are not registed by the scenario they must be registered through some other method")
	fs.BoolVar(&r.ignoreAlreadyStarted, "ignore-already-started", false,
		"Ignore if a workflow with the same ID already exists. A Scenario may choose to override this behavior.")
	fs.StringVar(&r.exportHistoriesDir, "export-histories-dir", "", "Export workflow histories to this directory")
	fs.StringVar(&r.exportHistoriesFilter, "export-histories-filter", "all", "Filter which workflows are exported by execution status (options: 'failed', 'terminated', 'failed,terminated', 'all'). Default is 'all'")
}

func (r *scenarioRunner) preRun() {
	r.logger = r.loggingOptions.MustCreateLogger()
}

// validateInput checks everything that can be judged from the command line alone
// and resolves the scenario's options.
//
// It touches nothing external — no server, no worker, no files beyond the @-file
// option values — so callers can run it before doing expensive setup and report a
// typo immediately rather than after a build or a connection attempt.
func (r *scenarioRunner) validateInput() (*loadgen.Scenario, *loadgen.OptionSet, error) {
	scenario := loadgen.GetScenario(r.scenario.Scenario)
	if scenario == nil {
		return nil, nil, loadgen.NewScenarioNotFoundError(r.scenario.Scenario)
	} else if r.scenario.RunID == "" {
		return nil, nil, loadgen.NewUsageError("--run-id must not be empty")
	} else if r.iterations > 0 && r.duration > 0 {
		return nil, nil, loadgen.NewUsageError("--iterations and --duration cannot be combined; " +
			"use --iterations to run a fixed number of times, or --duration to keep starting " +
			"iterations for a period")
	} else if policy := r.resolvedIterationFailurePolicy(); policy != iterationFailurePolicyContinue && policy != iterationFailurePolicyFailFast {
		return nil, nil, loadgen.NewUsageError(
			"--iteration-failure-policy must be %q or %q, got %q",
			iterationFailurePolicyContinue,
			iterationFailurePolicyFailFast,
			r.iterationFailurePolicy,
		)
	}

	// Parse options
	scenarioOptions := make(map[string]string, len(r.scenarioOptions))
	for _, v := range r.scenarioOptions {
		pieces := strings.SplitN(v, "=", 2)
		if len(pieces) != 2 {
			return nil, nil, loadgen.NewUsageError("--option %q is not in key=value format", v)
		}
		key, value := pieces[0], pieces[1]

		// If the value starts with '@', read the file and use its contents as the value.
		if after, ok := strings.CutPrefix(value, "@"); ok {
			filePath := after
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, nil, loadgen.NewUsageError("--option %s reads from file %q, which could not be read: %v",
					key, filePath, err)
			}
			value = string(data)
		}
		scenarioOptions[key] = value
	}

	resolvedOptions, resolveErr := scenario.ResolveOptions(scenarioOptions)
	if resolveErr != nil {
		return nil, nil, &loadgen.InvalidOptionsError{ScenarioName: r.scenario.Scenario, Err: resolveErr}
	}
	return scenario, resolvedOptions, nil
}

func (r scenarioRunConfig) resolvedIterationFailurePolicy() string {
	if r.iterationFailurePolicy == "" {
		return iterationFailurePolicyContinue
	}
	return r.iterationFailurePolicy
}

func (r scenarioRunConfig) loadgenConfiguration() loadgen.RunConfiguration {
	return loadgen.RunConfiguration{
		Iterations:                    r.iterations,
		Duration:                      r.duration,
		MaxConcurrent:                 r.maxConcurrent,
		MaxIterationsPerSecond:        r.maxIterationsPerSecond,
		MaxIterationAttempts:          r.maxIterationAttempts,
		Timeout:                       r.timeout,
		DoNotRegisterSearchAttributes: r.doNotRegisterSearchAttributes,
		IgnoreAlreadyStarted:          r.ignoreAlreadyStarted,
		ContinueOnIterationFailure:    r.resolvedIterationFailurePolicy() == iterationFailurePolicyContinue,
	}
}

// iterationFailuresOnly recognizes an IterationFailuresError through ordinary
// single-cause wrapping. It deliberately rejects multi-errors so a degraded
// completion cannot hide another run-level failure joined to it.
func iterationFailuresOnly(err error) (*loadgen.IterationFailuresError, bool) {
	for err != nil {
		if failures, ok := err.(*loadgen.IterationFailuresError); ok {
			return failures, true
		}
		if _, ok := err.(interface{ Unwrap() []error }); ok {
			return nil, false
		}
		err = errors.Unwrap(err)
	}
	return nil, false
}

func (r *scenarioRunner) run(ctx context.Context) error {
	scenario, resolvedOptions, err := r.validateInput()
	if err != nil {
		return err
	}

	metrics := r.metricsOptions.MustCreateMetrics(ctx, r.logger)
	defer metrics.Shutdown(ctx, r.logger, r.scenario.Scenario, r.scenario.RunID, r.scenario.RunFamily)
	start := time.Now()
	var client client.Client
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

	// Finalize any capability-gated feature options against the namespace under
	// test. This needs a dialed client, so it cannot happen in validateInput with
	// the rest of the option resolution. Wrapped as an InvalidOptionsError so an
	// explicitly-enabled-but-unsupported feature prints as the usage error it is,
	// rather than a fatal stack trace.
	if err := loadgen.ResolveFeatureOptions(ctx, client, r.clientOptions.Namespace, resolvedOptions, r.logger); err != nil {
		return &loadgen.InvalidOptionsError{ScenarioName: r.scenario.Scenario, Err: err}
	}

	repoDir, err := getRepoDir()
	if err != nil {
		return fmt.Errorf("failed to get root directory: %w", err)
	}

	// Generate a random execution ID to ensure no two executions with the same RunID collide
	executionID, err := generateExecutionID()
	if err != nil {
		return fmt.Errorf("failed to generate execution ID: %w", err)
	}

	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   r.scenario.Scenario,
		RunID:          r.scenario.RunID,
		ExecutionID:    executionID,
		Logger:         r.logger,
		MetricsHandler: metrics.NewHandler(),
		Client:         client,
		ClientOptions:  r.clientOptions,
		Configuration:  r.loadgenConfiguration(),
		Options:        resolvedOptions,
		Namespace:      r.clientOptions.Namespace,
		RootPath:       repoDir,
		ExportOptions: loadgen.ExportOptions{
			ExportHistoriesDir:    r.exportHistoriesDir,
			ExportHistoriesFilter: r.exportHistoriesFilter,
		},
	}
	executor := scenario.ExecutorFn()
	err = executor.Run(ctx, scenarioInfo)
	if err != nil {
		if r.resolvedIterationFailurePolicy() == iterationFailurePolicyContinue {
			if _, ok := iterationFailuresOnly(err); ok {
				err = nil
			}
		}
		if err != nil {
			return fmt.Errorf("failed scenario: %w", err)
		}
	}
	err = loadgen.ExportWorkflowHistories(ctx, scenarioInfo)
	if err != nil {
		scenarioInfo.Logger.Errorf("Error exporting workflow histories:\n %v", err)
	}
	return nil
}
