package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
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

Supports three modes:
  - Local: Build and run entry point (use --project-dir, --entry)
  - Remote: Connects to pre-running client/worker via HTTP (use --client-url, --worker-url)
  - Hybrid: Mix local and remote (e.g., local client + remote worker)

User code pattern:
  - User writes main.py/main.ts that calls: run(client=client_main, worker=worker_main)
  - The program is invoked with subcommand: python main.py client --port 8080 ...

Examples:
  # Local mode - builds and runs both client and worker
  omes workflow \
    --language python --sdk-version 1.21.0 \
    --project-dir ./my-test \
    --entry main.py \
    --iterations 100 --max-concurrent 10

  # Remote mode - connect to pre-running starters
  omes workflow \
    --client-url http://localhost:8080 \
    --worker-url http://localhost:8081 \
    --iterations 100

  # Hybrid mode - local client, remote worker
  omes workflow \
    --language python --sdk-version 1.21.0 \
    --project-dir ./my-test \
    --entry main.py \
    --client-only \
    --worker-url http://worker.ecs.internal:8081 \
    --iterations 100`,
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
	// Local build flags
	language   string
	sdkVersion string
	projectDir string
	entry      string // main.py or main.ts (calls run(client=..., worker=...))
	buildDir   string
	clientOnly bool // Only run client locally (use --worker-url for worker)
	workerOnly bool // Only run worker locally (use --client-url for client)

	// Remote mode flags
	clientURL string
	workerURL string

	// Load config
	iterations          int
	duration            time.Duration
	maxConcurrent       int
	maxIterationsPerSec float64
	timeout             time.Duration
	runID               string

	// Composed options
	clientOptions  clioptions.ClientOptions
	loggingOptions clioptions.LoggingOptions

	logger *zap.SugaredLogger
}

func (r *workflowRunner) addCLIFlags(fs *pflag.FlagSet) {
	// Local build flags
	fs.StringVar(&r.language, "language", "", "SDK language (python, typescript)")
	fs.StringVar(&r.sdkVersion, "sdk-version", "", "SDK version or local path")
	fs.StringVar(&r.projectDir, "project-dir", ".", "Path to user's test project")
	fs.StringVar(&r.entry, "entry", "", "Path to entry file (e.g., main.py). Defaults: main.py (python), main.ts (typescript)")
	fs.StringVar(&r.buildDir, "build-dir", "", "Directory for SDK build output (cached)")
	fs.BoolVar(&r.clientOnly, "client-only", false, "Only run client locally (requires --worker-url)")
	fs.BoolVar(&r.workerOnly, "worker-only", false, "Only run worker locally (requires --client-url)")

	// Remote mode flags
	fs.StringVar(&r.clientURL, "client-url", "", "URL of running client starter")
	fs.StringVar(&r.workerURL, "worker-url", "", "URL of running worker starter")

	// Load config flags
	fs.IntVar(&r.iterations, "iterations", 0, "Number of iterations (cannot be used with --duration)")
	fs.DurationVar(&r.duration, "duration", 0, "Test duration (cannot be used with --iterations)")
	fs.IntVar(&r.maxConcurrent, "max-concurrent", 10, "Max concurrent iterations")
	fs.Float64Var(&r.maxIterationsPerSec, "max-iterations-per-second", 0, "Max iterations per second rate limit")
	fs.DurationVar(&r.timeout, "timeout", 0, "Test timeout (0 = no timeout)")
	fs.StringVar(&r.runID, "run-id", "", "Run ID (auto-generated if not provided)")

	// Inherited flags
	fs.AddFlagSet(r.clientOptions.FlagSet())
	fs.AddFlagSet(r.loggingOptions.FlagSet())
}

func (r *workflowRunner) preRun() {
	r.logger = r.loggingOptions.MustCreateLogger()
}

func (r *workflowRunner) validate() error {
	// Determine if we're running locally
	runningLocally := r.language != "" || r.entry != ""

	// Cannot specify both --client-only and --worker-only
	if r.clientOnly && r.workerOnly {
		return errors.New("cannot specify both --client-only and --worker-only")
	}

	// If running locally, need language and sdk-version
	if runningLocally {
		if r.language == "" {
			return errors.New("--language required when building locally")
		}
		if r.sdkVersion == "" {
			return errors.New("--sdk-version required when building locally")
		}
	}

	// Hybrid mode validation
	if r.clientOnly && r.workerURL == "" {
		return errors.New("--client-only requires --worker-url")
	}
	if r.workerOnly && r.clientURL == "" {
		return errors.New("--worker-only requires --client-url")
	}

	// Full remote mode: need both URLs
	if !runningLocally {
		if r.clientURL == "" {
			return errors.New("must specify --client-url or build locally with --language and --entry")
		}
		if r.workerURL == "" {
			return errors.New("must specify --worker-url or build locally with --language and --entry")
		}
	}

	// Cannot specify both iterations and duration
	if r.iterations > 0 && r.duration > 0 {
		return errors.New("cannot specify both --iterations and --duration")
	}

	// Must specify at least one of iterations or duration
	if r.iterations == 0 && r.duration == 0 {
		return errors.New("must specify --iterations or --duration")
	}

	return nil
}

func (r *workflowRunner) run(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}

	// Generate run ID if not provided
	if r.runID == "" {
		id, err := generateExecutionID()
		if err != nil {
			return fmt.Errorf("failed to generate run ID: %w", err)
		}
		r.runID = id
	}
	taskQueue := fmt.Sprintf("omes-%s", r.runID)

	r.logger.Infof("Starting workflow load test (run-id: %s, task-queue: %s)", r.runID, taskQueue)

	// Set default entry file based on language
	entryFile := r.entry
	if entryFile == "" && r.language != "" {
		switch r.language {
		case "python":
			entryFile = "main.py"
		case "typescript":
			entryFile = "main.ts"
		}
	}

	// Build program if needed (single program handles both client and worker via subcommand)
	var prog sdkbuild.Program
	if r.needsBuild() {
		builder := &loadgen.ProgramBuilder{
			Language:   r.language,
			SDKVersion: r.sdkVersion,
			ProjectDir: r.projectDir,
			BuildDir:   r.buildDir,
			Logger:     r.logger,
		}

		var err error
		prog, err = builder.BuildProgram(ctx, entryFile)
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
	runClientLocally := prog != nil && !r.workerOnly
	runWorkerLocally := prog != nil && !r.clientOnly

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
		RunID:          r.runID,
		Logger:         r.logger,
		MetricsHandler: client.MetricsNopHandler,
		Configuration: loadgen.RunConfiguration{
			Iterations:             r.iterations,
			Duration:               r.duration,
			MaxConcurrent:          r.maxConcurrent,
			MaxIterationsPerSecond: r.maxIterationsPerSec,
			Timeout:                r.timeout,
		},
	}

	r.logger.Infof("Running load test with %d max concurrent", r.maxConcurrent)
	if err := executor.Run(ctx, scenarioInfo); err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}

	r.logger.Info("Load test completed successfully")
	return nil
}

func (r *workflowRunner) needsBuild() bool {
	return r.language != ""
}

func (r *workflowRunner) setupClient(ctx context.Context, prog sdkbuild.Program, runLocally bool, taskQueue string) (*loadgen.ClientStarter, func(), error) {
	if !runLocally || r.clientURL != "" {
		// Remote mode
		if r.clientURL == "" {
			return nil, nil, fmt.Errorf("no client URL specified for remote mode")
		}
		r.logger.Infof("Connecting to remote client at %s", r.clientURL)
		starter := loadgen.NewClientStarter(r.clientURL, r.logger)
		if err := r.waitForReady(ctx, r.clientURL+"/info", 30*time.Second); err != nil {
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
	port, err := findAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// First arg is subcommand "client", then runtime flags
	runtimeArgs := []string{
		"client", // subcommand
		"--port", strconv.Itoa(port),
		"--task-queue", taskQueue,
		"--server-address", r.clientOptions.Address,
		"--namespace", r.clientOptions.Namespace,
	}

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
	if err := r.waitForReady(ctx, clientURL+"/info", 30*time.Second); err != nil {
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
	if !runLocally || r.workerURL != "" {
		// Remote mode
		if r.workerURL == "" {
			return nil, nil, fmt.Errorf("no worker URL specified for remote mode")
		}
		r.logger.Infof("Connecting to remote worker at %s", r.workerURL)
		starter := loadgen.NewWorkerStarter(r.workerURL, r.logger)
		if err := r.waitForReady(ctx, r.workerURL+"/info", 30*time.Second); err != nil {
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
	port, err := findAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	// First arg is subcommand "worker", then runtime flags
	runtimeArgs := []string{
		"worker", // subcommand
		"--task-queue", taskQueue,
		"--server-address", r.clientOptions.Address,
		"--namespace", r.clientOptions.Namespace,
	}

	server := &loadgen.WorkerLifecycleServer{
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
	if err := r.waitForReadyWithErrCh(ctx, server.URL()+"/info", 30*time.Second, errCh); err != nil {
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

func (r *workflowRunner) waitForReady(ctx context.Context, url string, timeout time.Duration) error {
	return r.waitForReadyWithErrCh(ctx, url, timeout, nil)
}

func (r *workflowRunner) waitForReadyWithErrCh(ctx context.Context, url string, timeout time.Duration, errCh <-chan error) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		// Check for startup error
		if errCh != nil {
			select {
			case err := <-errCh:
				if err != nil {
					return fmt.Errorf("process failed to start: %w", err)
				}
			default:
			}
		}

		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for %s", url)
}

func findAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}
