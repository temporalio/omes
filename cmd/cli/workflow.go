package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/programbuild"
	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func workflowCmd() *cobra.Command {
	var r workflowRunner
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run a workflow load test",
		Long: `Build a test project, spawn a client process, run load, and exit.

The client is always spawned. Use --spawn-worker to also spawn a
local worker process; otherwise a worker should already be running.

Examples:
  omes workflow --language python --project-dir ./my-test --spawn-worker --iterations 100
  omes workflow --language python --project-dir ./my-test --iterations 100`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := r.preRun(); err != nil {
				return err
			}
			return r.validate(cmd)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			return r.run(ctx)
		},
	}
	r.addCLIFlags(cmd)
	return cmd
}

type workflowRunner struct {
	sdkOpts           clioptions.SdkOptions
	clientOpts        clioptions.ClientOptions
	loggingOpts       clioptions.LoggingOptions
	loadOpts          clioptions.LoadOptions
	programOpts       clioptions.ProgramOptions
	workflowOpts      clioptions.WorkflowOptions
	workerMetricsOpts clioptions.MetricsOptions
	runID             string
	taskQueue         string
	clientPort        int

	logger *zap.SugaredLogger
}

func (r *workflowRunner) addCLIFlags(cmd *cobra.Command) {
	fs := cmd.Flags()
	r.sdkOpts.AddCLIFlags(fs)
	fs.AddFlagSet(r.clientOpts.FlagSet())
	fs.AddFlagSet(r.loggingOpts.FlagSet())
	r.loadOpts.AddCLIFlags(cmd)
	r.programOpts.AddFlags(fs)
	r.workflowOpts.AddCLIFlags(fs)
	fs.StringVar(&r.runID, "run-id", "", "Run ID (auto-generated if not provided)")
	fs.StringVar(&r.taskQueue, "task-queue", "", "Task queue name (default: omes-<run-id>)")
	fs.IntVar(&r.clientPort, "client-port", 0, "Port for local client HTTP server (0 = auto)")
	fs.AddFlagSet(r.workerMetricsOpts.FlagSet("worker-"))
}

func (r *workflowRunner) preRun() error {
	r.logger = r.loggingOpts.MustCreateLogger()

	if r.runID == "" {
		r.runID = generateExecutionID()
	}
	if r.taskQueue == "" {
		r.taskQueue = fmt.Sprintf("omes-%s", r.runID)
	}
	if err := utils.DefaultPort(&r.clientPort); err != nil {
		return err
	}
	return nil
}

func (r *workflowRunner) validate(cmd *cobra.Command) error {
	if r.sdkOpts.Language == "" {
		return fmt.Errorf("--language is required")
	}
	if r.programOpts.ProgramDir == "" {
		return fmt.Errorf("--project-dir is required")
	}
	if !r.workflowOpts.SpawnWorker && !cmd.Flags().Changed("task-queue") {
		return fmt.Errorf("--task-queue is required when --spawn-worker is false (runner does not own worker lifecycle)")
	}
	return nil
}

func (r *workflowRunner) run(ctx context.Context) error {
	r.logger.Infof("Starting workflow load test (run-id: %s, task-queue: %s)", r.runID, r.taskQueue)

	// Build program
	builder := programbuild.ProgramBuilder{
		Language:   r.sdkOpts.Language.String(),
		ProjectDir: r.programOpts.ProgramDir,
		BuildDir:   r.programOpts.BuildDir,
		Logger:     r.logger,
	}
	prog, err := builder.BuildProgram(ctx, r.sdkOpts.Version)
	if err != nil {
		return fmt.Errorf("failed to build program: %w", err)
	}

	// Spawn worker (if requested)
	// TODO: add worker readiness gate (e.g. DescribeTaskQueue poller check) to avoid
	// race where load starts before worker is polling. Currently relies on worker starting fast enough.
	if r.workflowOpts.SpawnWorker {
		workerProcess, err := r.spawnLocalWorker(ctx, prog)
		if err != nil {
			return fmt.Errorf("failed to spawn worker: %w", err)
		}
		defer r.killProcess("worker", workerProcess)
	}

	// Spawn client
	clientProcess, err := r.spawnLocalClient(ctx, prog)
	if err != nil {
		return fmt.Errorf("failed to spawn client: %w", err)
	}
	defer r.killProcess("client", clientProcess)

	clientURL := fmt.Sprintf("http://127.0.0.1:%d", r.clientPort)
	clientHandle := loadgen.NewClientHandle(clientURL)
	if err := clientHandle.WaitForReady(ctx); err != nil {
		return fmt.Errorf("client not ready: %w", err)
	}

	// Run load test
	executor := &loadgen.HTTPExecutor{
		Client: clientHandle,
	}
	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:   "workflow-test",
		RunID:          r.runID,
		Logger:         r.logger,
		MetricsHandler: client.MetricsNopHandler,
		Configuration: loadgen.RunConfiguration{
			Iterations:             r.loadOpts.Iterations,
			Duration:               r.loadOpts.Duration,
			MaxConcurrent:          r.loadOpts.MaxConcurrent,
			MaxIterationsPerSecond: r.loadOpts.MaxIterationsPerSecond,
			Timeout:                r.loadOpts.Timeout,
		},
	}

	r.logger.Infof("Running load test with %d max concurrent", r.loadOpts.MaxConcurrent)
	if err := executor.Run(ctx, scenarioInfo); err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}

	r.logger.Info("Load test completed successfully")
	return nil
}

func (r *workflowRunner) spawnLocalClient(ctx context.Context, program sdkbuild.Program) (*os.Process, error) {
	clientArgs := []string{
		"client",
		"--task-queue", r.taskQueue,
	}
	clientArgs = append(clientArgs, r.connectionRuntimeArgs()...)
	clientArgs = append(clientArgs, "--port", strconv.Itoa(r.clientPort))

	runtimeArgs, err := programbuild.BuildRuntimeArgs(
		r.sdkOpts.Language,
		r.programOpts.ProgramDir,
		clientArgs...,
	)
	if err != nil {
		return nil, err
	}
	return programbuild.StartProgramProcess(ctx, program, runtimeArgs)
}

func (r *workflowRunner) spawnLocalWorker(ctx context.Context, program sdkbuild.Program) (*os.Process, error) {
	workerArgs := []string{
		"worker",
		"--task-queue", r.taskQueue,
	}
	workerArgs = append(workerArgs, r.connectionRuntimeArgs()...)
	if addr := r.workerMetricsOpts.PrometheusListenAddress; addr != "" {
		workerArgs = append(workerArgs, "--prom-listen-address", addr)
	}
	runtimeArgs, err := programbuild.BuildRuntimeArgs(
		r.sdkOpts.Language,
		r.programOpts.ProgramDir,
		workerArgs...,
	)
	if err != nil {
		return nil, err
	}
	return programbuild.StartProgramProcess(ctx, program, runtimeArgs)
}

func (r *workflowRunner) connectionRuntimeArgs() []string {
	args := []string{
		"--server-address", r.clientOpts.Address,
		"--namespace", r.clientOpts.Namespace,
	}
	if r.clientOpts.AuthHeader != "" {
		args = append(args, "--auth-header", r.clientOpts.AuthHeader)
	}
	if r.clientOpts.EnableTLS {
		args = append(args, "--tls")
	}
	if r.clientOpts.TLSServerName != "" {
		args = append(args, "--tls-server-name", r.clientOpts.TLSServerName)
	}
	return args
}

func (r *workflowRunner) killProcess(name string, process *os.Process) {
	r.logger.Infof("Sending SIGTERM to %s (PID %d)", name, process.Pid)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		r.logger.Warnf("SIGTERM failed for %s: %v, killing", name, err)
		process.Kill()
		process.Wait()
		return
	}
	done := make(chan struct{})
	go func() { process.Wait(); close(done) }()
	select {
	case <-done:
		r.logger.Infof("%s process exited", name)
	case <-time.After(15 * time.Second):
		r.logger.Warnf("%s did not exit in 15s, killing", name)
		process.Kill()
	}
}
