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
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	api "github.com/temporalio/omes/workers/go/projects/harness/api"
	"go.uber.org/zap"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: `Run a self-contained project test. Builds (or loads) the project program, spawns a project-server, and drives iterations via gRPC.
  Required: --option language=<lang>
  One of:   --option project-name=<name>  (build from source; optional --option version=<version> override)
            --option prebuilt-project-dir=<path>  (use pre-built project dir)
  Optional: --option project-config-file=<path>   (project-specific JSON config)
  See README.md ("Project (Python only)") for local usage examples and current limitations.`,
		ExecutorFn: func() loadgen.Executor {
			return &projectScenarioExecutor{}
		},
	})
}

type projectScenarioOptions struct {
	sdkOpts     clioptions.SdkOptions
	projectName string
	prebuiltDir string
	configJSON  []byte
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
		prog, err = loadPrebuilt(opts.prebuiltDir, opts.sdkOpts.Language)
	} else {
		info.Logger.Infof("Building project %s", filepath.Base(opts.projectName))
		prog, err = buildProject(ctx, info.RootPath, opts, info.Logger)
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

	serverCmd, err := startProjectProcess(serverCtx, prog, info.Logger, opts.sdkOpts.Language, port)
	if err != nil {
		return fmt.Errorf("failed to spawn project server: %w", err)
	}
	defer stopProjectProcess("project-server", serverCancel, serverCmd, info.Logger)

	co := info.ClientOptions
	handle, err := NewProjectHandle(ctx, port, &api.InitRequest{
		ExecutionId: info.ExecutionID,
		RunId:       info.RunID,
		TaskQueue:   taskQueue,
		ConnectOptions: &api.ConnectOptions{
			Namespace:               co.Namespace,
			ServerAddress:           co.Address,
			AuthHeader:              co.AuthHeader,
			EnableTls:               co.EnableTLS,
			TlsCertPath:             co.ClientCertPath,
			TlsKeyPath:              co.ClientKeyPath,
			TlsServerName:           co.TLSServerName,
			DisableHostVerification: co.DisableHostVerification,
		},
		ConfigJson: opts.configJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to init project: %w", err)
	}
	defer handle.Close()

	executor := NewSteadyRateExecutor(&handle)
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
	if opts.sdkOpts.Language != clioptions.LangPython {
		return opts, fmt.Errorf("project scenario is currently limited to Python, got %s", lang)
	}

	projectName := info.ScenarioOptions["project-name"]
	prebuiltDir := info.ScenarioOptions["prebuilt-project-dir"]
	if projectName == "" && prebuiltDir == "" {
		return opts, fmt.Errorf("either --option project-name or --option prebuilt-project-dir is required")
	}
	if projectName != "" && prebuiltDir != "" {
		return opts, fmt.Errorf("cannot specify both project-name and prebuilt-project-dir")
	}

	if prebuiltDir != "" {
		abs, err := filepath.Abs(prebuiltDir)
		if err != nil {
			return opts, fmt.Errorf("failed to resolve prebuilt-project-dir: %w", err)
		}
		opts.prebuiltDir = abs
	} else {
		opts.projectName = projectName
		version := info.ScenarioOptions["version"]
		if version != "" {
			opts.sdkOpts.Version = version
		}
	}

	if configPath := info.ScenarioOptions["project-config-file"]; configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return opts, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
		opts.configJSON = data
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

func startProjectProcess(ctx context.Context, prog sdkbuild.Program, logger *zap.SugaredLogger, lang clioptions.Language, port int) (*exec.Cmd, error) {
	var args []string
	if lang == clioptions.LangPython {
		args = append(args, "main")
	}
	args = append(args, "project-server", "--port", strconv.Itoa(port))
	cmd, err := prog.NewCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	// Set pgid to kill spawned child processes
	// (needed largely for Python, which spawns a child process through uv)
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
