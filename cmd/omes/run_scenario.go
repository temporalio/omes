package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/load"
)

func runScenarioCmd() *cobra.Command {
	var r scenarioRunner
	cmd := &cobra.Command{
		Use:   "run-scenario",
		Short: "Run scenario",
		Args:  validateLoadArgs,
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
	load                          string
	loadPattern                   *load.Pattern
	flags                         *pflag.FlagSet
	maxConcurrent                 int
	maxIterationsPerSecond        float64
	maxIterationAttempts          int
	scenarioOptions               []string
	timeout                       time.Duration
	doNotRegisterSearchAttributes bool
	ignoreAlreadyStarted          bool
	exportHistoriesDir            string
	exportHistoriesFilter         string
	continueOnIterationFailure    bool
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
	r.flags = fs
	fs.StringVar(&r.load, "load", "", `Load pattern (requires --duration); parameters must directly follow --load <pattern>:
  steady: peak required
  ramp-hold-ramp: peak required, up=0.1, down=0.1
  sweep: peak required, from=0.25, to=1, steps=4
  spike-train: peak required, base=0.4, every=10m, for=1m
Example: --load spike-train peak=120/s every=5m`)
	fs.IntVar(&r.iterations, "iterations", 0, "Override default iterations for the scenario (cannot be provided with duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Override duration for the scenario (cannot be provided with iteration)."+
		" This is the amount of time for which we will start new iterations of the scenario.")
	fs.Float64Var(&r.maxIterationsPerSecond, "max-iterations-per-second", 0, "Override iterations per second rate limit for the scenario."+
		" This is the maximum rate at which we will start new iterations of the scenario.")
	fs.IntVar(&r.maxIterationAttempts, "max-iteration-attempts", 1, "Maximum attempts per iteration")
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
	fs.BoolVar(&r.continueOnIterationFailure, "continue-on-iteration-failure", false, "Continue running iterations even if an iteration fails")
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
	}
	if err := r.validateLoad(scenario); err != nil {
		return nil, nil, err
	}
	if r.iterations > 0 && r.duration > 0 {
		return nil, nil, loadgen.NewUsageError("--iterations and --duration cannot be combined; " +
			"use --iterations to run a fixed number of times, or --duration to keep starting " +
			"iterations for a period")
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

func (r *scenarioRunConfig) validateLoad(scenario *loadgen.Scenario) error {
	usageError := func(message string) error {
		return loadgen.NewUsageError("%s; e.g. %s --duration 20m", message, load.Example("spike-train"))
	}
	r.loadPattern = nil
	changed := func(name string) bool { return r.flags != nil && r.flags.Changed(name) }
	if r.load == "" && !changed("load") {
		return nil
	}
	if r.iterations != 0 || changed("iterations") {
		return usageError("--iterations cannot be combined with --load; use --duration")
	}
	if r.maxIterationsPerSecond != 0 || changed("max-iterations-per-second") {
		return usageError("--max-iterations-per-second cannot be combined with --load; set peak= in --load")
	}
	if r.duration <= 0 {
		return usageError("--load requires a positive --duration")
	}
	if r.maxConcurrent < 0 {
		return usageError("--max-concurrent must be positive (or 0 for the default)")
	}
	pattern, err := load.Parse(r.load, r.duration)
	if err != nil {
		return loadgen.NewUsageError("%v", err)
	}
	if executor, ok := scenario.ExecutorFn().(loadgen.LoadPatternExecutor); !ok || !executor.SupportsLoadPattern() {
		return usageError("this scenario does not support --load; its executor must use the GenericExecutor pacer")
	}
	r.loadPattern = pattern
	return nil
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
		Configuration: loadgen.RunConfiguration{
			Iterations:                    r.iterations,
			Duration:                      r.duration,
			LoadPattern:                   r.loadPattern,
			MaxConcurrent:                 r.maxConcurrent,
			MaxIterationsPerSecond:        r.maxIterationsPerSecond,
			MaxIterationAttempts:          r.maxIterationAttempts,
			Timeout:                       r.timeout,
			DoNotRegisterSearchAttributes: r.doNotRegisterSearchAttributes,
			IgnoreAlreadyStarted:          r.ignoreAlreadyStarted,
			ContinueOnIterationFailure:    r.continueOnIterationFailure,
		},
		Options:   resolvedOptions,
		Namespace: r.clientOptions.Namespace,
		RootPath:  repoDir,
		ExportOptions: loadgen.ExportOptions{
			ExportHistoriesDir:    r.exportHistoriesDir,
			ExportHistoriesFilter: r.exportHistoriesFilter,
		},
	}
	executor := scenario.ExecutorFn()
	err = executor.Run(ctx, scenarioInfo)
	if err != nil {
		return fmt.Errorf("failed scenario: %w", err)
	}
	err = loadgen.ExportWorkflowHistories(ctx, scenarioInfo)
	if err != nil {
		scenarioInfo.Logger.Errorf("Error exporting workflow histories:\n %v", err)
	}
	return nil
}
