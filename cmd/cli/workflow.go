package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
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
  - Local: Spawns client and worker as subprocesses (use --client-command, --worker-command)
  - Remote: Connects to pre-running client/worker via HTTP (use --client-url, --worker-url)
  - Hybrid: Mix local and remote (e.g., local client + remote worker)

Examples:
  # Local mode - spawns both client and worker
  omes workflow \
    --language python --sdk-version 1.21.0 \
    --client-command "python client.py" \
    --worker-command "python worker.py" \
    --iterations 100 --max-concurrent 10

  # Remote mode - connect to pre-running starters
  omes workflow \
    --client-url http://localhost:8080 \
    --worker-url http://localhost:8081 \
    --iterations 100

  # Hybrid mode - local client, remote worker
  omes workflow \
    --language python --sdk-version 1.21.0 \
    --client-command "python client.py" \
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
	// Mode flags
	language      string
	sdkVersion    string
	buildDir      string
	useExisting   bool
	clientCommand string
	workerCommand string
	clientURL     string
	workerURL     string

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
	// Local mode flags
	fs.StringVar(&r.language, "language", "", "SDK language (python, typescript)")
	fs.StringVar(&r.sdkVersion, "sdk-version", "", "SDK version or local path")
	fs.StringVar(&r.buildDir, "build-dir", "", "Directory for SDK build output")
	fs.BoolVar(&r.useExisting, "use-existing", false, "Use existing build, fail if not found")
	fs.StringVar(&r.clientCommand, "client-command", "", "Command to run client (e.g., 'python client.py')")
	fs.StringVar(&r.workerCommand, "worker-command", "", "Command to run worker (e.g., 'python worker.py')")

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
	// Cannot mix command and URL for same component
	if r.clientCommand != "" && r.clientURL != "" {
		return errors.New("cannot specify both --client-command and --client-url")
	}
	if r.workerCommand != "" && r.workerURL != "" {
		return errors.New("cannot specify both --worker-command and --worker-url")
	}

	// Must have at least one of each
	if r.clientCommand == "" && r.clientURL == "" {
		return errors.New("must specify --client-command or --client-url")
	}
	if r.workerCommand == "" && r.workerURL == "" {
		return errors.New("must specify --worker-command or --worker-url")
	}

	// Language required if spawning locally
	if (r.clientCommand != "" || r.workerCommand != "") && r.language == "" {
		return errors.New("--language required when using --client-command or --worker-command")
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

	// Build SDK if needed
	var buildDir string
	if r.needsBuild() {
		runner := &loadgen.SDKRunner{
			Language:    r.language,
			SDKVersion:  r.sdkVersion,
			BuildDir:    r.buildDir,
			UseExisting: r.useExisting,
			Logger:      r.logger,
		}
		var err error
		buildDir, err = runner.EnsureBuild(ctx)
		if err != nil {
			return fmt.Errorf("failed to build SDK: %w", err)
		}
	}

	// Track cleanup functions
	var cleanups []func()
	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	// Setup client
	clientStarter, clientCleanup, err := r.setupClient(ctx, buildDir, taskQueue)
	if err != nil {
		return fmt.Errorf("failed to setup client: %w", err)
	}
	cleanups = append(cleanups, clientCleanup)

	// Setup worker
	workerStarter, workerCleanup, err := r.setupWorker(ctx, buildDir, taskQueue)
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
	return r.clientCommand != "" || r.workerCommand != ""
}

func (r *workflowRunner) setupClient(ctx context.Context, buildDir, taskQueue string) (*loadgen.ClientStarter, func(), error) {
	if r.clientURL != "" {
		// Remote mode
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

	// Local mode
	port, err := findAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	args := parseCommand(r.clientCommand)
	args = append(args,
		"--port", strconv.Itoa(port),
		"--task-queue", taskQueue,
		"--server-address", r.clientOptions.Address,
		"--namespace", r.clientOptions.Namespace,
	)

	cmd, err := loadgen.BuildCommandWithOverride(r.language, buildDir, r.sdkVersion, args[0], args[1:])
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build command: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	r.logger.Infof("Starting local client on port %d: %s", port, r.clientCommand)
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

func (r *workflowRunner) setupWorker(ctx context.Context, buildDir, taskQueue string) (*loadgen.WorkerStarter, func(), error) {
	if r.workerURL != "" {
		// Remote mode
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

	// Local mode - use WorkerLifecycleServer
	port, err := findAvailablePort()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find available port: %w", err)
	}

	args := parseCommand(r.workerCommand)
	args = append(args,
		"--task-queue", taskQueue,
		"--server-address", r.clientOptions.Address,
		"--namespace", r.clientOptions.Namespace,
	)

	server := &loadgen.WorkerLifecycleServer{
		Language:   r.language,
		SDKVersion: r.sdkVersion,
		BuildDir:   buildDir,
		Command:    args,
		Port:       port,
		Logger:     r.logger,
	}

	r.logger.Infof("Starting local worker on port %d: %s", port, r.workerCommand)

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

func parseCommand(command string) []string {
	// Simple space-based splitting - doesn't handle quoted strings
	// For more complex cases, users should use shell wrapper scripts
	return strings.Fields(command)
}

