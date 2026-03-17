package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/programbuild"
	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/internal/verification"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/projectexecutor"
	"github.com/temporalio/omes/metrics"
	"github.com/temporalio/omes/projecttests/go/harness/api"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func workflowCmd() *cobra.Command {
	var r projectRunner
	cmd := &cobra.Command{
		Use:   "workflow",
		Short: "Run a workflow load test",
		Long: `Build a test project, spawn a client process, run load, and exit.

The client is always spawned. Use --spawn-worker to also spawn a
local worker process; otherwise a worker should already be running.

Examples:
  omes workflow --language go --project-dir ./projecttests/go/tests/helloworld --spawn-worker --iterations 100
  omes workflow --language go --project-dir ./projecttests/go/tests/helloworld --iterations 100`,
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

type projectRunner struct {
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
	executor     string
	configPath   string
	options      []string
	verify       bool

	logger *zap.SugaredLogger
}

type workflowMetricsFlags struct {
	PrometheusAddress           string
	ExportParquetPath           string
	MetricsVersionTag           string
	WorkerPromListenAddress     string
	WorkerProcessMetricsAddress string
}

func (r *projectRunner) addCLIFlags(cmd *cobra.Command) {
	fs := cmd.Flags()
	r.sdkOpts.AddCLIFlags(fs)
	fs.Lookup("language").Usage = "Language to use for workflow tests (go only)"
	fs.AddFlagSet(r.clientOpts.FlagSet())
	fs.AddFlagSet(r.loggingOpts.FlagSet())
	r.loadOpts.AddCLIFlags(cmd)
	r.programOpts.AddFlags(fs)
	r.workflowOpts.AddCLIFlags(fs)
	fs.StringVar(&r.runID, "run-id", "", "Run ID (auto-generated if not provided)")
	fs.StringVar(&r.runFamily, "run-family", "", "Human-readable identifier for grouping related runs")
	fs.StringVar(&r.taskQueue, "task-queue", "", "Task queue name (default: omes-<run-id>)")
	fs.IntVar(&r.clientPort, "client-port", 0, "Port for local client HTTP server (0 = auto)")
	fs.StringVar(&r.executor, "executor", "", "projecttests executor to run")
	fs.StringVar(&r.configPath, "config", "", "Path to JSON config file for project test")
	fs.StringSliceVar(&r.options, "option", nil, "Additional options for the executor, in key=value format")
	fs.BoolVar(&r.verify, "verify", true, "Run post-load verification (visibility count, no failures)")
	fs.StringVar(&r.metricsOpts.PrometheusAddress, "prometheus-address", "", "Prometheus API address")
	fs.StringVar(&r.metricsOpts.ExportParquetPath, "export-parquet-path", "", "Export metrics to parquet at this path")
	fs.StringVar(&r.metricsOpts.MetricsVersionTag, "metrics-version-tag", "", "SDK version/ref label for exported metrics")
	fs.StringVar(&r.metricsOpts.WorkerPromListenAddress, "worker-prom-listen-address", "", "Worker Prometheus listen address")
	fs.StringVar(&r.metricsOpts.WorkerProcessMetricsAddress, "worker-process-metrics-address", "", "Worker process metrics sidecar address")
}

func (r *projectRunner) preRun() error {
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

func (r *projectRunner) validate(cmd *cobra.Command) error {
	if r.sdkOpts.Language == "" {
		return fmt.Errorf("--language is required")
	}
	if r.sdkOpts.Language != clioptions.LangGo {
		return fmt.Errorf("--language must be go for workflow tests (got %q)", r.sdkOpts.Language)
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
	switch r.executor {
	case projectexecutor.SteadyRateProjectExecutor,
		projectexecutor.EbbAndFlowProjectExecutor,
		projectexecutor.SaturationProjectExecutor:
	default:
		return fmt.Errorf("--executor must be one of: %s, %s, %s",
			projectexecutor.SteadyRateProjectExecutor,
			projectexecutor.EbbAndFlowProjectExecutor,
			projectexecutor.SaturationProjectExecutor)
	}
	if r.executor == projectexecutor.SaturationProjectExecutor {
		if r.metricsOpts.PrometheusAddress == "" {
			return fmt.Errorf("--prometheus-address is required for the saturation executor")
		}
		if r.metricsOpts.WorkerProcessMetricsAddress == "" {
			return fmt.Errorf("--worker-process-metrics-address is required for the saturation executor (process sidecar must be running)")
		}
	}
	return nil
}

func (r *projectRunner) run(ctx context.Context) error {
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

	// Spawn project server
	projectServerCtx, projectServerCancel := context.WithCancel(ctx)
	projectServerCmd, err := r.spawnLocalProjectServer(projectServerCtx, prog)
	if err != nil {
		projectServerCancel()
		return fmt.Errorf("failed to spawn project server: %w", err)
	}
	defer r.stopCommand("project-server", projectServerCancel, projectServerCmd)
	executionID := generateExecutionID()

	// Read project config file if provided
	var configBytes []byte
	if r.configPath != "" {
		configBytes, err = os.ReadFile(r.configPath)
		if err != nil {
			return fmt.Errorf("failed to read config file %s: %w", r.configPath, err)
		}
	}

	projectHandle, err := projectexecutor.NewProjectHandle(ctx, r.clientPort, &api.InitRequest{
		ExecutionId: executionID,
		RunId:       r.runID,
		TaskQueue:   r.taskQueue,
		ConnectOptions: &api.ConnectOptions{
			Namespace:                r.clientOpts.Namespace,
			ServerAddress:            r.clientOpts.Address,
			AuthHeader:               r.clientOpts.AuthHeader,
			EnableTls:                r.clientOpts.EnableTLS,
			TlsCertPath:              r.clientOpts.ClientCertPath,
			TlsKeyPath:               r.clientOpts.ClientKeyPath,
			TlsServerName:            r.clientOpts.TLSServerName,
			DisableHostVerification:  r.clientOpts.DisableHostVerification,
		},
		Config:                   configBytes,
		RegisterSearchAttributes: true,
	})
	if err != nil {
		return fmt.Errorf("Error getting new client handle: %v", err)
	}
	defer projectHandle.Close()
	executor, err := projectexecutor.NewProjectTestExecutor(r.executor, &projectHandle, r.metricsOpts.PrometheusAddress)
	if err != nil {
		return err
	}

	scenarioOptions, err := parseOptionValues(r.options)
	if err != nil {
		return err
	}

	// Run load test
	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName:    "project-test",
		RunID:           r.runID,
		ExecutionID:     executionID,
		Logger:          r.logger,
		MetricsHandler:  client.MetricsNopHandler,
		ScenarioOptions: scenarioOptions,
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

	// Post-run verification
	if r.verify {
		if err := verification.RunAll(ctx, verification.Config{
			Dial:                 func() (client.Client, error) { return r.clientOpts.DialLite(r.logger) },
			Namespace:            r.clientOpts.Namespace,
			Logger:               r.logger,
			ExecutionID:          executionID,
			MinWorkflowCount:     r.loadOpts.Iterations,
			MinThroughputPerHour: scenarioInfo.ScenarioOptionFloat("min-throughput-per-hour", 0),
			LoadDuration:         loadEnd.Sub(loadStart),
		}); err != nil {
			return fmt.Errorf("post-run verification failed: %w", err)
		}
	}

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

func (r *projectRunner) spawnLocalProjectServer(ctx context.Context, program sdkbuild.Program) (*exec.Cmd, error) {
	clientArgs := []string{
		"project-server",
		"--port", strconv.Itoa(r.clientPort),
	}

	return programbuild.StartProgramProcess(ctx, program, clientArgs)
}

func (r *projectRunner) spawnLocalWorker(ctx context.Context, program sdkbuild.Program) (*exec.Cmd, error) {
	workerArgs := []string{
		"worker",
		"--task-queue", r.taskQueue,
	}
	workerArgs = append(workerArgs, r.connectionRuntimeArgs()...)
	if addr := r.metricsOpts.WorkerPromListenAddress; addr != "" {
		workerArgs = append(workerArgs, "--prom-listen-address", addr)
	}
	return programbuild.StartProgramProcess(ctx, program, workerArgs)
}

func (r *projectRunner) connectionRuntimeArgs() []string {
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
	if r.clientOpts.ClientCertPath != "" {
		args = append(args, "--tls-cert-path", r.clientOpts.ClientCertPath)
	}
	if r.clientOpts.ClientKeyPath != "" {
		args = append(args, "--tls-key-path", r.clientOpts.ClientKeyPath)
	}
	if r.clientOpts.TLSServerName != "" {
		args = append(args, "--tls-server-name", r.clientOpts.TLSServerName)
	}
	if r.clientOpts.DisableHostVerification {
		args = append(args, "--disable-tls-host-verification")
	}
	return args
}

func (r *projectRunner) stopCommand(name string, cancel context.CancelFunc, cmd *exec.Cmd) {
	if cancel != nil {
		cancel()
	}
	if cmd == nil {
		return
	}
	if err := cmd.Wait(); err != nil {
		r.logger.Warnf("Failed waiting for %s process shutdown: %v", name, err)
	}
}

func (r *projectRunner) resolvedMetricsVersionTag() string {
	if r.metricsOpts.MetricsVersionTag != "" {
		return r.metricsOpts.MetricsVersionTag
	}
	return r.sdkOpts.Version
}

