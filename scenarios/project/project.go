package project

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workerctl"
	"github.com/temporalio/omes/loadgen"
	api "github.com/temporalio/omes/workers/go/harness/api"
	"go.uber.org/zap"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: `Run a self-contained project test. Builds (or loads) the project program, spawns a project-server, and drives iterations via gRPC.
  Required: --option language=<lang>
  Required: --option project-name=<name>  (app entrypoint; optional --option version=<version> override)
  Optional: --option prebuilt-project-dir=<path>  (use pre-built root worker package)
  Optional: --option project-config-file=<path>   (project-specific JSON config)
  Optional: --option project-server-ready-timeout=<duration> (timeout to connect to project-server, default 15s)
  See README.md ("Project" section) for local usage examples and current limitations.`,
		ExecutorFn: func() loadgen.Executor {
			return &projectScenarioExecutor{}
		},
	})
}

type projectScenarioOptions struct {
	sdkOpts                   clioptions.SdkOptions
	projectName               string
	prebuiltDir               string
	configJSON                []byte
	projectServerReadyTimeout time.Duration
}

type projectScenarioExecutor struct{}

func (e *projectScenarioExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	opts, err := e.validate(info)
	if err != nil {
		return err
	}

	var prog sdkbuild.Program
	if opts.prebuiltDir != "" {
		info.Logger.Infof("Loading prebuilt project from %s", opts.prebuiltDir)
		baseDir := workerctl.BaseDir(info.RootPath, opts.sdkOpts.Language)
		prog, err = workerctl.LoadProgramFromDir(opts.prebuiltDir, baseDir, opts.sdkOpts.Language)
	} else {
		info.Logger.Infof("Building project %s", filepath.Base(opts.projectName))
		prog, err = (&workerctl.Builder{
			DirName:    fmt.Sprintf("project-build-runner-%s", opts.projectName),
			SdkOptions: opts.sdkOpts,
			Logger:     info.Logger,
		}).Build(ctx, workerctl.BaseDir(info.RootPath, opts.sdkOpts.Language))
	}
	if err != nil {
		return fmt.Errorf("failed to prepare project: %w", err)
	}

	port, err := findAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	taskQueue := loadgen.TaskQueueForRun(info.RunID)

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverCmd, err := startProjectProcess(serverCtx, prog, info.Logger, opts.sdkOpts.Language, opts.projectName, port)
	if err != nil {
		return fmt.Errorf("failed to spawn project server: %w", err)
	}
	defer stopProjectProcess("project-server", serverCancel, serverCmd, info.Logger)

	co := info.ClientOptions
	if len(co.ServerAddresses()) != 1 {
		return fmt.Errorf("project scenario supports exactly one server address")
	}
	handle, err := newProjectHandle(ctx, port, &api.InitRequest{
		ExecutionId: info.ExecutionID,
		RunId:       info.RunID,
		TaskQueue:   taskQueue,
		ConnectOptions: &api.ConnectOptions{
			Namespace:               co.Namespace,
			ServerAddress:           co.ServerAddress(),
			AuthHeader:              co.AuthHeader,
			EnableTls:               co.EnableTLS,
			TlsCertPath:             co.ClientCertPath,
			TlsKeyPath:              co.ClientKeyPath,
			TlsServerName:           co.TLSServerName,
			DisableHostVerification: co.DisableHostVerification,
		},
		ConfigJson: opts.configJSON,
	}, opts.projectServerReadyTimeout)
	if err != nil {
		return fmt.Errorf("failed to init project: %w", err)
	}
	defer handle.close()

	executor := newSteadyRateExecutor(&handle)
	return executor.Run(ctx, info)
}

func (e *projectScenarioExecutor) validate(info loadgen.ScenarioInfo) (projectScenarioOptions, error) {
	var opts projectScenarioOptions

	lang := info.ScenarioOptions["language"]
	if lang == "" {
		return opts, fmt.Errorf("--option language=<lang> is required")
	}
	if err := opts.sdkOpts.Language.Set(lang); err != nil {
		return opts, fmt.Errorf("unrecognized language: %s", lang)
	}

	projectName := info.ScenarioOptions["project-name"]
	prebuiltDir := info.ScenarioOptions["prebuilt-project-dir"]
	if projectName == "" {
		return opts, fmt.Errorf("--option project-name=<name> is required")
	}
	opts.projectName = projectName

	if prebuiltDir != "" {
		abs, err := filepath.Abs(prebuiltDir)
		if err != nil {
			return opts, fmt.Errorf("failed to resolve prebuilt-project-dir: %w", err)
		}
		opts.prebuiltDir = abs
	}

	version := info.ScenarioOptions["version"]
	if version != "" && opts.prebuiltDir == "" {
		opts.sdkOpts.Version = version
	}

	if configPath := info.ScenarioOptions["project-config-file"]; configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return opts, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		opts.configJSON = data
	}

	opts.projectServerReadyTimeout = defaultClientReadyTimeout
	if timeout := info.ScenarioOptions["project-server-ready-timeout"]; timeout != "" {
		parsed, err := time.ParseDuration(timeout)
		if err != nil {
			return opts, fmt.Errorf("invalid project-server-ready-timeout %q: %w", timeout, err)
		}
		opts.projectServerReadyTimeout = parsed
	}

	return opts, nil
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

func startProjectProcess(ctx context.Context, prog sdkbuild.Program, logger *zap.SugaredLogger, lang clioptions.Language, appName string, port int) (*exec.Cmd, error) {
	var args []string
	switch lang {
	case clioptions.LangPython:
		args = append(args, "apps.registry")
	case clioptions.LangTypeScript:
		args = append(args, "./tslib/apps/registry.js")
	}
	args = append(args, "--app", appName, "project-server", "--port", strconv.Itoa(port))

	cmd, err := prog.NewCommand(ctx, args...)
	if err != nil {
		return nil, err
	}

	// Set pgid to kill spawned child processes
	// (needed for Python, which spawns a child process through uv)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}
	cmd.WaitDelay = 3 * time.Second
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	logger.Infof("Started process (PID %d): %v", cmd.Process.Pid, args)
	return cmd, nil
}

func stopProjectProcess(name string, cancel context.CancelFunc, cmd *exec.Cmd, logger *zap.SugaredLogger) {
	cancel()
	if cmd != nil {
		if err := cmd.Wait(); err != nil {
			logger.Debugf("Process %s exited: %v", name, err)
		}
	}
}
