package cli

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/programbuild"
	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/metrics"
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
	sdkOpts      clioptions.SdkOptions
	clientOpts   clioptions.ClientOptions
	loggingOpts  clioptions.LoggingOptions
	loadOpts     clioptions.LoadOptions
	programOpts  clioptions.ProgramOptions
	workflowOpts clioptions.WorkflowOptions
	metricsOpts  workflowMetricsFlags
	runID        string
	runFamily    string
	taskQueue    string
	clientPort   int

	logger *zap.SugaredLogger
}

type workflowMetricsFlags struct {
	PrometheusAddress           string
	ExportParquetPath           string
	MetricsVersionTag           string
	WorkerPromListenAddress     string
	WorkerProcessMetricsAddress string
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
	fs.StringVar(&r.runFamily, "run-family", "", "Human-readable identifier for grouping related runs")
	fs.StringVar(&r.taskQueue, "task-queue", "", "Task queue name (default: omes-<run-id>)")
	fs.IntVar(&r.clientPort, "client-port", 0, "Port for local client HTTP server (0 = auto)")
	fs.StringVar(&r.metricsOpts.PrometheusAddress, "prometheus-address", "http://localhost:9090", "Prometheus API address")
	fs.StringVar(&r.metricsOpts.ExportParquetPath, "export-parquet-path", "", "Export metrics to parquet at this path")
	fs.StringVar(&r.metricsOpts.MetricsVersionTag, "metrics-version-tag", "", "SDK version/ref label for exported metrics")
	fs.StringVar(&r.metricsOpts.WorkerPromListenAddress, "worker-prom-listen-address", "", "Worker Prometheus listen address")
	fs.StringVar(&r.metricsOpts.WorkerProcessMetricsAddress, "worker-process-metrics-address", "", "Worker process metrics sidecar address")
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
		return fmt.Errorf("--task-queue is required when --spawn-worker is false")
	}
	if r.metricsOpts.ExportParquetPath != "" && r.metricsOpts.PrometheusAddress == "" {
		return fmt.Errorf("--prometheus-address is required when --export-parquet-path is set")
	}
	if r.metricsOpts.WorkerProcessMetricsAddress != "" && !r.workflowOpts.SpawnWorker {
		return fmt.Errorf("--spawn-worker is required when --worker-process-metrics-address is set")
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
	var workerCmd *exec.Cmd
	if r.workflowOpts.SpawnWorker {
		workerCtx, workerCancel := context.WithCancel(ctx)
		workerCmd, err = r.spawnLocalWorker(workerCtx, prog)
		if err != nil {
			workerCancel()
			return fmt.Errorf("failed to spawn worker: %w", err)
		}
		defer r.stopCommand("worker", workerCancel, workerCmd)
	}

	// Spawn worker process monitoring server (if requested)
	spawnSidecar := workerCmd != nil && r.metricsOpts.WorkerProcessMetricsAddress != ""
	if spawnSidecar {
		sidecar := clioptions.StartProcessMetricsSidecar(
			r.logger,
			r.metricsOpts.WorkerProcessMetricsAddress,
			workerCmd.Process.Pid,
			r.resolvedMetricsVersionTag(),
			"",
			r.sdkOpts.Language.String(),
		)
		defer func() {
			if err := sidecar.Shutdown(context.Background()); err != nil {
				r.logger.Warnf("Failed to stop worker process metrics sidecar: %v", err)
			}
		}()
	}

	// Spawn client
	clientCtx, clientCancel := context.WithCancel(ctx)
	clientCmd, err := r.spawnLocalClient(clientCtx, prog)
	if err != nil {
		clientCancel()
		return fmt.Errorf("failed to spawn client: %w", err)
	}
	defer r.stopCommand("client", clientCancel, clientCmd)

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

	loadStart := time.Now()
	r.logger.Infof("Running load test with %d max concurrent", r.loadOpts.MaxConcurrent)
	if err := executor.Run(ctx, scenarioInfo); err != nil {
		return fmt.Errorf("load test failed: %w", err)
	}
	loadEnd := time.Now()

	// Export metrics (if requested)
	if r.metricsOpts.ExportParquetPath != "" {
		err := metrics.ExportMetricLineParquetFromPrometheus(
			ctx,
			metrics.PromQueryConfig{
				Address: r.metricsOpts.PrometheusAddress,
				Start:   loadStart,
				End:     loadEnd,
				Queries: metrics.DefaultDerivedQueries(),
			},
			r.metricsOpts.ExportParquetPath,
			metrics.MetricLineMetadata{
				Scenario:   scenarioInfo.ScenarioName,
				RunID:      scenarioInfo.RunID,
				RunFamily:  r.runFamily,
				SDKVersion: r.resolvedMetricsVersionTag(),
				BuildID:    "",
				Language:   r.sdkOpts.Language.String(),
			},
		)
		if err != nil {
			return err
		}
		r.logger.Infof("Exported parquet metrics to %s", r.metricsOpts.ExportParquetPath)
	}

	r.logger.Info("Load test completed successfully")
	return nil
}

func (r *workflowRunner) spawnLocalClient(ctx context.Context, program sdkbuild.Program) (*exec.Cmd, error) {
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

func (r *workflowRunner) spawnLocalWorker(ctx context.Context, program sdkbuild.Program) (*exec.Cmd, error) {
	workerArgs := []string{
		"worker",
		"--task-queue", r.taskQueue,
	}
	workerArgs = append(workerArgs, r.connectionRuntimeArgs()...)
	if addr := r.metricsOpts.WorkerPromListenAddress; addr != "" {
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

func (r *workflowRunner) stopCommand(name string, cancel context.CancelFunc, cmd *exec.Cmd) {
	if cancel != nil {
		cancel()
	}
	if cmd == nil {
		return
	}
	if err := cmd.Wait(); err != nil && cmd.ProcessState == nil {
		r.logger.Warnf("Failed waiting for %s process shutdown: %v", name, err)
	}
}

func (r *workflowRunner) resolvedMetricsVersionTag() string {
	if r.metricsOpts.MetricsVersionTag != "" {
		return r.metricsOpts.MetricsVersionTag
	}
	return r.sdkOpts.Version
}
