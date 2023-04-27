package main

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/omes"
	"github.com/temporalio/omes/omes/runner"
	_ "github.com/temporalio/omes/scenarios" // Register scenarios (side-effect)
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

// options for bootstrapping a scenario
type appOptions struct {
	// Override for scnario duration
	duration time.Duration
	// Override for scnario iterations
	iterations int
	// Whether or not to start a local server
	startLocalServer bool
}

type App struct {
	logger     *zap.SugaredLogger
	metrics    *omes.Metrics
	runner     *runner.Runner
	appOptions appOptions
	// Options for configuring logging
	loggingOptions omes.LoggingOptions
	// Options for configuring the client connection
	clientOptions omes.ClientOptions
	// General runner options
	runnerOptions runner.Options
	// Options for runner.Cleanup
	cleanOptions runner.CleanupOptions
	// Options for runner.RunWorker
	workerOptions  runner.WorkerOptions
	metricsOptions omes.MetricsOptions
	devServer      *testsuite.DevServer
}

// applyOverrides from CLI flags to a loaded scenario
func (a *App) applyOverrides(scenario *omes.Scenario) error {
	opts, ok := scenario.Executor.(*omes.SharedIterationsExecutor)
	if !ok {
		// Don't know how to override options here
		return nil
	}
	iterations := a.appOptions.iterations
	duration := a.appOptions.duration

	if iterations > 0 && duration > 0 {
		return errors.New("invalid options: iterations and duration are mutually exclusive")
	}
	if iterations > 0 {
		opts.Iterations = iterations
		opts.Duration = 0
	} else if duration > 0 {
		opts.Duration = duration
		opts.Iterations = 0
	}
	return nil
}

func shortRand() (string, error) {
	b := make([]byte, 5)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return strings.ToLower(base32.StdEncoding.EncodeToString(b)), nil
}

// Setup the application and runner instance.
// Starts a local server if requested.
func (a *App) Setup(cmd *cobra.Command, args []string) {
	a.logger = a.loggingOptions.MustCreateLogger()
	a.metrics = a.metricsOptions.MustCreateMetrics(a.logger)

	if a.runnerOptions.RunID == "" {
		runID, err := shortRand()
		if err != nil {
			a.logger.Fatalf("Failed to generate a short random ID: %v", err)
		}
		a.runnerOptions.RunID = runID
	}

	scenario := omes.GetScenario(a.runnerOptions.ScenarioName)
	if scenario == nil {
		a.logger.Fatalf("failed to find a registered scenario named %q", a.runnerOptions.ScenarioName)
	}
	a.applyOverrides(scenario)
	if a.appOptions.startLocalServer {
		// TODO: cli version / log level
		server, err := testsuite.StartDevServer(context.Background(), testsuite.DevServerOptions{
			ClientOptions: &client.Options{
				Namespace: a.clientOptions.Namespace,
			},
			LogLevel: "error",
		})
		if err != nil {
			a.logger.Fatalf("Failed to start local dev server: %v", err)
		}
		a.logger.Infof("Started local dev server at: %s", server.FrontendHostPort())
		a.clientOptions.Address = server.FrontendHostPort()
		a.clientOptions.ClientCertPath = ""
		a.clientOptions.ClientKeyPath = ""
		a.devServer = server
	}
	a.runnerOptions.Scenario = scenario
	a.runnerOptions.ClientOptions = a.clientOptions
	a.workerOptions.ClientOptions = a.clientOptions
	a.runner = runner.NewRunner(a.runnerOptions, a.metrics, a.logger)
}

// Teardown stops the dev server in case one was started during Setup.
func (a *App) Teardown(cmd *cobra.Command, args []string) {
	if a.devServer != nil {
		a.logger.Info("Stopping local dev server")
		a.devServer.Stop()
		// TODO: uncomment this, the dev server should exit gracefully
		// if err := a.devServer.Stop(); err != nil {
		// 	a.logger.Fatalf("Failed to stop local dev server: %v", err)
		// }
	}
	if a.metrics != nil {
		if err := a.metrics.Shutdown(cmd.Context()); err != nil {
			a.logger.Fatalf("Failed to shutdown metrics: %v", err)
		}
	}
}

// Run starts a new scenario run.
func (a *App) Run(cmd *cobra.Command, args []string) {
	client := a.clientOptions.MustDial(a.metrics, a.logger)
	defer client.Close()

	if err := a.runner.Run(cmd.Context(), client); err != nil {
		a.logger.Fatalf("Run failed: %v", err)
	}
}

// Cleanup resources created by a previous run.
func (a *App) Cleanup(cmd *cobra.Command, args []string) {
	client := a.clientOptions.MustDial(a.metrics, a.logger)
	defer client.Close()

	if err := a.runner.Cleanup(cmd.Context(), client, a.cleanOptions); err != nil {
		a.logger.Fatalf("Cleanup failed: %v", err)
	}
}

// RunWorker builds and runs a worker for the language specified in options.
func (a *App) RunWorker(cmd *cobra.Command, args []string) {
	a.workerOptions.LoggingOptions = a.loggingOptions
	if err := a.runner.RunWorker(cmd.Context(), a.workerOptions); err != nil {
		a.logger.Fatalf("Worker failed: %v", err)
	}
}

// RunAllInOne runs a worker, an optional local server, and a scenario.
func (a *App) RunAllInOne(cmd *cobra.Command, args []string) {
	client := a.clientOptions.MustDial(a.metrics, a.logger)
	defer client.Close()

	options := runner.AllInOneOptions{WorkerOptions: a.workerOptions}
	if err := a.runner.RunAllInOne(cmd.Context(), client, options); err != nil {
		a.logger.Fatalf("Run failed: %v", err)
	}
}

func addRunFlags(cmd *cobra.Command, app *App) {
	cmd.Flags().DurationVarP(&app.appOptions.duration, "duration", "d", 0, "Scenario duration - mutually exclusive with iterations")
	cmd.Flags().IntVarP(&app.appOptions.iterations, "iterations", "i", 0, "Scenario iterations - mutually exclusive with duration")
}

func addWorkerFlags(cmd *cobra.Command, app *App) {
	cmd.Flags().StringVar(&app.workerOptions.Language, "language", "", "Language of worker that will be spawned (go typescript python java)")
	cmd.Flags().BoolVar(&app.workerOptions.RetainBuildDir, "retain-build-dir", false, "If set, retain the directory used to build the worker")
	cmd.Flags().DurationVar(&app.workerOptions.GracefulShutdownDuration, "graceful-shutdown-duration", 30*time.Second, "Time to wait for worker to respond to SIGTERM before SIGKILLing it")
	cmd.MarkFlagRequired("language")
}

func main() {
	var app App

	var rootCmd = &cobra.Command{
		Use:               "omes",
		Short:             "A load generator for Temporal",
		PersistentPreRun:  app.Setup,
		PersistentPostRun: app.Teardown,
	}

	var runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run a test scenario",
		Run:   app.Run,
	}

	var cleanupCmd = &cobra.Command{
		Use:   "cleanup",
		Short: "Cleanup after scenario run",
		Run:   app.Cleanup,
	}

	var startWorkerCmd = &cobra.Command{
		Use:   "run-worker",
		Short: "Build and run a local worker",
		Run:   app.RunWorker,
	}

	var allInOneCmd = &cobra.Command{
		Use:   "all-in-one",
		Short: "Run a complete scenario with a local worker",
		Run:   app.RunAllInOne,
	}

	app.loggingOptions.AddCLIFlags(rootCmd.PersistentFlags(), "")
	app.metricsOptions.AddCLIFlags(rootCmd.PersistentFlags(), "")
	app.clientOptions.AddCLIFlags(rootCmd.PersistentFlags())
	rootCmd.PersistentFlags().StringVarP(&app.runnerOptions.ScenarioName, "scenario", "s", "", "Scenario to run (see scenarios/)")
	rootCmd.MarkFlagRequired("scenario")
	rootCmd.PersistentFlags().StringVar(&app.runnerOptions.RunID, "run-id", "", "Optional unique ID for a scenario run")

	// run command
	rootCmd.AddCommand(runCmd)
	addRunFlags(runCmd, &app)

	// cleanup command
	rootCmd.AddCommand(cleanupCmd)
	cleanupCmd.Flags().DurationVar(&app.cleanOptions.PollInterval, "poll-interval", time.Second, "Interval for polling on cleanup batch job completion")

	// start-worker command
	rootCmd.AddCommand(startWorkerCmd)
	addWorkerFlags(startWorkerCmd, &app)

	// all-in-one command
	rootCmd.AddCommand(allInOneCmd)
	addWorkerFlags(allInOneCmd, &app)
	addRunFlags(allInOneCmd, &app)
	app.workerOptions.LoggingOptions.AddCLIFlags(allInOneCmd.Flags(), "worker-")
	app.workerOptions.MetricsOptions.AddCLIFlags(allInOneCmd.Flags(), "worker-")
	allInOneCmd.Flags().BoolVar(&app.appOptions.startLocalServer, "start-local-server", false, "Start a local server (server-address and TLS options will be ignored)")

	defer func() {
		if app.logger != nil {
			app.logger.Sync()
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		if app.logger != nil {
			app.logger.Fatal(err)
		} else {
			omes.BackupLogger.Fatal(err)
		}
	}
}
