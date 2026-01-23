package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/progbuild"
	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func workflowCmd() *cobra.Command {
	var r workflowRunner
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run workflow load tests",
		Long: `Run workflow load tests using user-defined client and worker code.

Modes:
  Local:  Build and run locally (--language, --project-dir)
  Remote: Connect to pre-running endpoints (--client-url, --worker-url)
  Hybrid: Mix local and remote (--client-only or --worker-only)

Specify load with --iterations or --duration.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			return r.run(ctx)
		},
	}
	r.addCLIFlags(cmd.Flags())
	return cmd
}

type workflowRunner struct {
	sdkOpts      clioptions.SdkOptions
	clientOpts   clioptions.ClientOptions
	loggingOpts  clioptions.LoggingOptions
	workflowOpts clioptions.WorkflowOptions

	logger *zap.SugaredLogger
}

func (r *workflowRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.sdkOpts.AddCLIFlags(fs)
	fs.AddFlagSet(r.clientOpts.FlagSet())
	fs.AddFlagSet(r.loggingOpts.FlagSet())
	fs.AddFlagSet(r.workflowOpts.FlagSet())
}

func (r *workflowRunner) preRun() {
	r.logger = r.loggingOpts.MustCreateLogger()
}

func (r *workflowRunner) validate() error {
	// Determine if we're running locally
	runningLocally := r.sdkOpts.Language != ""

	// Cannot specify both --client-only and --worker-only
	if r.workflowOpts.ClientOnly && r.workflowOpts.WorkerOnly {
		return errors.New("cannot specify both --client-only and --worker-only")
	}

	// If running locally, need language and version
	if runningLocally {
		if r.sdkOpts.Version == "" {
			return errors.New("--sdk-version required when building locally")
		}
	}

	// Hybrid mode validation
	if r.workflowOpts.ClientOnly && r.workflowOpts.WorkerURL == "" {
		return errors.New("--client-only requires --worker-url")
	}
	if r.workflowOpts.WorkerOnly && r.workflowOpts.ClientURL == "" {
		return errors.New("--worker-only requires --client-url")
	}

	// Full remote mode: need both URLs
	if !runningLocally {
		if r.workflowOpts.ClientURL == "" {
			return errors.New("must specify --client-url or build locally with --language")
		}
		if r.workflowOpts.WorkerURL == "" {
			return errors.New("must specify --worker-url or build locally with --language")
		}
	}

	// Cannot specify both iterations and duration
	if r.workflowOpts.Iterations > 0 && r.workflowOpts.Duration > 0 {
		return errors.New("cannot specify both --iterations and --duration")
	}

	// Must specify at least one of iterations or duration
	if r.workflowOpts.Iterations == 0 && r.workflowOpts.Duration == 0 {
		return errors.New("must specify --iterations or --duration")
	}

	return nil
}

func (r *workflowRunner) run(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}

	// Generate run ID if not provided
	runID := r.workflowOpts.RunID
	if runID == "" {
		id, err := generateExecutionID()
		if err != nil {
			return fmt.Errorf("failed to generate run ID: %w", err)
		}
		runID = id
	}
	taskQueue := fmt.Sprintf("omes-%s", runID)

	r.logger.Infof("Starting workflow load test (run-id: %s, task-queue: %s)", runID, taskQueue)

	// Build program if needed (single program handles both client and worker via subcommand)
	var prog sdkbuild.Program
	if r.needsBuild() {
		builder := &progbuild.ProgramBuilder{
			Language:   r.sdkOpts.Language.String(),
			SDKVersion: r.sdkOpts.Version,
			ProjectDir: r.workflowOpts.ProjectDir,
			BuildDir:   r.workflowOpts.BuildDir,
			Logger:     r.logger,
		}

		var err error
		prog, err = builder.BuildProgram(ctx)
		if err != nil {
			return fmt.Errorf("failed to build program: %w", err)
		}
	}

	// Track cleanup functions
	var cleanups []func()
	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	// Determine what to run locally vs remotely
	runClientLocally := prog != nil && !r.workflowOpts.WorkerOnly
	runWorkerLocally := prog != nil && !r.workflowOpts.ClientOnly

	// Setup client
	clientStarter, clientCleanup, err := r.setupClient(ctx, prog, runClientLocally, taskQueue)
	if err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}
	cleanups = append(cleanups, clientCleanup)

	// Setup worker
	workerStarter, workerCleanup, err := r.setupWorker(ctx, prog, runWorkerLocally, taskQueue)
	if err != nil {
		return fmt.Errorf("failed to setup worker: %w", err)
	}
	cleanups = append(cleanups, workerCleanup)

	// Run load test
	executor := &loadgen.HTTPExecutor{
		Client: clientStarter,
		Worker: workerStarter,
		Logger: r.logger,
	}

	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   "workflow-test",
		RunID:          runID,
		Logger:         r.logger,
		MetricsHandler: client.MetricsNopHandler,
		Configuration: loadgen.RunConfiguration{
			Iterations:             r.workflowOpts.Iterations,
			Duration:               r.workflowOpts.Duration,
			MaxConcurrent:          r.workflowOpts.MaxConcurrent,
			MaxIterationsPerSecond: r.workflowOpts.MaxIterationsPerSec,
			Timeout:                r.workflowOpts.Timeout,
		},
	}

	r.logger.Infof("Running load test with %d max concurrent", r.workflowOpts.MaxConcurrent)
	if err := executor.Run(ctx, scenarioInfo); err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}

	r.logger.Info("Load test completed successfully")
	return nil
}

func (r *workflowRunner) needsBuild() bool {
	return r.sdkOpts.Language != ""
}

func (r *workflowRunner) setupClient(ctx context.Context, prog sdkbuild.Program, runLocally bool, taskQueue string) (*loadgen.ClientStarter, func(), error) {
	if !runLocally || r.workflowOpts.ClientURL != "" {
		// Remote mode
		if r.workflowOpts.ClientURL == "" {
			return nil, nil, fmt.Errorf("no client URL specified for remote mode")
		}
		r.logger.Infof("Connecting to remote client at %s", r.workflowOpts.ClientURL)
		starter := loadgen.NewClientStarter(r.workflowOpts.ClientURL, r.logger)
		if err := utils.WaitForReady(ctx, r.workflowOpts.ClientURL+"/info", 30*time.Second); err != nil {
			return nil, nil, fmt.Errorf("client not ready: %w", err)
		}
		cleanup := func() {
			if err := starter.Shutdown(ctx); err != nil {
				r.logger.Warnf("Failed to shutdown remote client: %v", err)
			}
		}
		return starter, cleanup, nil
	}

	// Local mode - spawn client process with "client" subcommand
	port, err := utils.FindAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Build runtime args based on language
	var runtimeArgs []string
	projectName := filepath.Base(r.workflowOpts.ProjectDir)
	switch r.sdkOpts.Language {
	case clioptions.LangPython:
		// Python: first arg is module name
		runtimeArgs = []string{projectName}
	case clioptions.LangTypeScript:
		// TypeScript: first arg is compiled entry point (preserves dir structure)
		runtimeArgs = []string{fmt.Sprintf("tslib/tests/%s/main.js", projectName)}
	}
	runtimeArgs = append(runtimeArgs,
		"client", // subcommand
		"--port", strconv.Itoa(port),
		"--task-queue", taskQueue,
		"--server-address", r.clientOpts.Address,
		"--namespace", r.clientOpts.Namespace,
	)

	cmd, err := prog.NewCommand(ctx, runtimeArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create command: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.logger.Infof("Starting local client on port %d", port)
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("failed to start client: %w", err)
	}

	clientURL := fmt.Sprintf("http://localhost:%d", port)
	if err := utils.WaitForReady(ctx, clientURL+"/info", 30*time.Second); err != nil {
		cmd.Process.Kill()
		return nil, nil, fmt.Errorf("client not ready: %w", err)
	}

	starter := loadgen.NewClientStarter(clientURL, r.logger)
	cleanup := func() {
		if err := starter.Shutdown(ctx); err != nil {
			r.logger.Warnf("Graceful shutdown failed, killing process: %v", err)
			cmd.Process.Kill()
		}
	}
	return starter, cleanup, nil
}

func (r *workflowRunner) setupWorker(ctx context.Context, prog sdkbuild.Program, runLocally bool, taskQueue string) (*loadgen.WorkerStarter, func(), error) {
	if !runLocally || r.workflowOpts.WorkerURL != "" {
		// Remote mode
		if r.workflowOpts.WorkerURL == "" {
			return nil, nil, fmt.Errorf("no worker URL specified for remote mode")
		}
		r.logger.Infof("Connecting to remote worker at %s", r.workflowOpts.WorkerURL)
		starter := loadgen.NewWorkerStarter(r.workflowOpts.WorkerURL, r.logger)
		if err := utils.WaitForReady(ctx, r.workflowOpts.WorkerURL+"/info", 30*time.Second); err != nil {
			return nil, nil, fmt.Errorf("worker not ready: %w", err)
		}
		cleanup := func() {
			if err := starter.Shutdown(ctx); err != nil {
				r.logger.Warnf("Failed to shutdown remote worker: %v", err)
			}
		}
		return starter, cleanup, nil
	}

	// Local mode - use WorkerLifecycleServer with Program
	port, err := utils.FindAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// Build runtime args based on language
	var runtimeArgs []string
	projectName := filepath.Base(r.workflowOpts.ProjectDir)
	switch r.sdkOpts.Language {
	case clioptions.LangPython:
		// Python: first arg is module name (derive: "simple-test" → "simple_test")
		moduleName := strings.ReplaceAll(projectName, "-", "_")
		runtimeArgs = []string{moduleName}
	case clioptions.LangTypeScript:
		// TypeScript: first arg is compiled entry point (preserves dir structure)
		runtimeArgs = []string{fmt.Sprintf("tslib/tests/%s/main.js", projectName)}
	}
	runtimeArgs = append(runtimeArgs,
		"worker", // subcommand
		"--task-queue", taskQueue,
		"--server-address", r.clientOpts.Address,
		"--namespace", r.clientOpts.Namespace,
	)

	server := &progbuild.WorkerLifecycleServer{
		Program: prog,
		Args:    runtimeArgs,
		Port:    port,
		Logger:  r.logger,
	}

	r.logger.Infof("Starting local worker on port %d", port)

	// Start server in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(ctx)
	}()

	// Wait for ready or error
	if err := utils.WaitForReadyWithErrCh(ctx, server.URL()+"/info", 30*time.Second, errCh); err != nil {
		server.Kill()
		return nil, nil, fmt.Errorf("worker not ready: %w", err)
	}

	starter := loadgen.NewWorkerStarter(server.URL(), r.logger)
	cleanup := func() {
		if err := starter.Shutdown(ctx); err != nil {
			r.logger.Warnf("Graceful shutdown failed, killing process: %v", err)
			server.Kill()
		}
	}
	return starter, cleanup, nil
}
