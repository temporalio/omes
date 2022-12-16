package main

import (
	"errors"
	"time"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/runner"
	devserver "github.com/temporalio/omes/runner/dev_server"
	"github.com/temporalio/omes/scenarios"
	"github.com/temporalio/omes/shared"
)

// options to pass from the command line to the runner
type appOptions struct {
	// Name of the scenario to run
	scenario    string
	logLevel    string
	logEncoding string
	// Override for scnario duration
	duration time.Duration
	// Override for scnario iterations
	iterations  uint32
	tlsCertPath string
	tlsKeyPath  string
	// Whether or not to start a local server
	startLocalServer bool
}

type App struct {
	logger     *zap.SugaredLogger
	runner     *runner.Runner
	appOptions appOptions
	// Options for configuring the client connection
	clientOptions client.Options
	// General runner options
	runnerOptions runner.Options
	// Options for runner.Run
	runOptions runner.RunOptions
	// Options for runner.Cleanup
	cleanOptions runner.CleanupOptions
	// Options for runner.StartWorker
	workerOptions runner.WorkerOptions
	// Dev server handle (see startLocalServer)
	devServer *devserver.DevServer
}

// applyOverrides from CLI flags to a loaded scenario
func (a *App) applyOverrides(scenario *scenarios.Scenario) error {
	iterations := a.appOptions.iterations
	duration := a.appOptions.duration

	if iterations > 0 && duration > 0 {
		return errors.New("invalid options: iterations and duration are mutually exclusive")
	}
	if iterations > 0 {
		scenario.Iterations = iterations
		scenario.Duration = 0
	} else if duration > 0 {
		scenario.Duration = duration
		scenario.Iterations = 0
	}
	return nil
}

// Setup the application and runner instance.
// If a local server should be started, that will be done in this method.
func (a *App) Setup(cmd *cobra.Command, args []string) {
	a.logger = shared.SetupLogging(a.appOptions.logLevel, a.appOptions.logEncoding)

	if a.runnerOptions.RunID == "" {
		// TODO: make a nicer, shorter ID for this
		a.runnerOptions.RunID = uuid.NewString()
	}

	scenario, err := scenarios.GetScenarioByName(a.appOptions.scenario)
	if err != nil {
		a.logger.Fatalf("failed to find scenario: %v", err)
	}
	a.applyOverrides(&scenario)
	if a.appOptions.startLocalServer {
		// TODO: cli version / log level
		server, err := devserver.Start(devserver.Options{Namespace: a.clientOptions.Namespace, Log: a.logger, LogLevel: "error"})
		if err != nil {
			a.logger.Fatalf("Failed to start local dev server: %v", err)
		}
		a.logger.Infof("Started local dev server at: %s", server.FrontendHostPort)
		a.clientOptions.HostPort = server.FrontendHostPort
		a.clientOptions.ConnectionOptions.TLS = nil
		a.devServer = server
	} else {
		shared.LoadCertsIntoOptions(&a.clientOptions, a.appOptions.tlsCertPath, a.appOptions.tlsKeyPath)
		a.workerOptions.TLSCertPath = a.appOptions.tlsCertPath
		a.workerOptions.TLSKeyPath = a.appOptions.tlsKeyPath
	}
	a.runnerOptions.Scenario = scenario
	a.runnerOptions.ClientOptions = a.clientOptions
	r, err := runner.NewRunner(a.runnerOptions, a.logger)
	if err != nil {
		a.logger.Fatalf("failed to instantiate runner: %v", err)
	}
	a.runner = r
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
}

func (a *App) Run(cmd *cobra.Command, args []string) {
	if err := a.runner.Run(cmd.Context(), a.runOptions); err != nil {
		a.logger.Fatalf("Run failed: %v", err)
	}
}

func (a *App) Cleanup(cmd *cobra.Command, args []string) {
	if err := a.runner.Cleanup(cmd.Context(), a.cleanOptions); err != nil {
		a.logger.Fatalf("Cleanup failed: %v", err)
	}
}

func (a *App) StartWorker(cmd *cobra.Command, args []string) {
	if err := a.runner.StartWorker(cmd.Context(), a.workerOptions); err != nil {
		a.logger.Fatalf("Worker failed: %v", err)
	}
}

func (a *App) RunAllInOne(cmd *cobra.Command, args []string) {
	options := runner.AllInOneOptions{StartWorkerOptions: a.workerOptions, RunOptions: a.runOptions}
	if err := a.runner.AllInOne(cmd.Context(), options); err != nil {
		a.logger.Fatalf("Run failed: %v", err)
	}
}

func addRunFlags(cmd *cobra.Command, app *App) {
	cmd.Flags().DurationVarP(&app.appOptions.duration, "duration", "d", 0, "Scenario duration - mutually exclusive with iterations")
	cmd.Flags().Uint32VarP(&app.appOptions.iterations, "iterations", "i", 0, "Scenario iterations - mutually exclusive with duration")
	cmd.Flags().BoolVarP(&app.runOptions.FailFast, "fail-fast", "f", false, "Fail run after first encountered error")
}

func addWorkerFlags(cmd *cobra.Command, app *App) {
	cmd.Flags().StringVar(&app.workerOptions.Language, "language", "", "Language of worker that will be spawned (go typescript python java)")
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
		Use:   "start-worker",
		Short: "Start a local worker",
		Run:   app.StartWorker,
	}

	var allInOneCmd = &cobra.Command{
		Use:   "all-in-one",
		Short: "Run a complete scenario with a local worker",
		Run:   app.RunAllInOne,
	}

	rootCmd.PersistentFlags().StringVar(&app.appOptions.logLevel, "log-level", "info", "(debug info warn error dpanic panic fatal)")
	rootCmd.PersistentFlags().StringVar(&app.appOptions.logEncoding, "log-encoding", "console", "(console json)")
	rootCmd.PersistentFlags().StringVarP(&app.appOptions.scenario, "scenario", "s", "", "Scenario to run (see scenarios/)")
	rootCmd.MarkFlagRequired("scenario")
	rootCmd.PersistentFlags().StringVar(&app.runnerOptions.RunID, "run-id", "", "Optional unique ID for a scenario run")
	rootCmd.PersistentFlags().StringVarP(&app.clientOptions.HostPort, "server-address", "a", "localhost:7233", "address of Temporal server")
	rootCmd.PersistentFlags().StringVar(&app.appOptions.tlsCertPath, "tls-cert-path", "", "Path to TLS certificate")
	rootCmd.PersistentFlags().StringVar(&app.appOptions.tlsKeyPath, "tls-key-path", "", "Path to private key")
	rootCmd.PersistentFlags().StringVarP(&app.clientOptions.Namespace, "namespace", "n", "default", "namespace to connect to")

	// TODO: this should only apply to all-in-one
	rootCmd.PersistentFlags().BoolVar(&app.appOptions.startLocalServer, "start-local-server", false, "Start a local server (server-address and TLS options will be ignored)")

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

	defer func() {
		if app.logger != nil {
			app.logger.Sync()
		}
	}()
	if err := rootCmd.Execute(); err != nil {
		if app.logger != nil {
			app.logger.Fatal(err)
		} else {
			shared.BackupLogger.Fatal(err)
		}
	}
}
