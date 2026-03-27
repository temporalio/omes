package project

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers/go/projects/api"
	"go.uber.org/zap"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a self-contained project test. Options: language=<lang> (required), project-dir=<path> (build from source) or prebuilt-project-dir=<path> (use pre-built binary), project-config-file=<path> (optional).",
		ExecutorFn: func() loadgen.Executor {
			return &projectScenarioExecutor{}
		},
	})
}

type projectScenarioOptions struct {
	language    clioptions.Language
	projectDir  string
	prebuiltDir string
	configJSON  []byte
}

// projectScenarioExecutor wraps the full project test lifecycle:
// build (or load) project binary, spawn processes, gRPC init, execute iterations, cleanup.
type projectScenarioExecutor struct{}

func (e *projectScenarioExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	opts, err := e.validate(info)
	if err != nil {
		return err
	}

	var prog sdkbuild.Program
	if opts.prebuiltDir != "" {
		info.Logger.Infof("Loading prebuilt project from %s", opts.prebuiltDir)
		prog, err = LoadPrebuilt(opts.prebuiltDir, opts.language)
	} else {
		info.Logger.Infof("Building project %s", filepath.Base(opts.projectDir))
		prog, err = Build(ctx, BuildOptions{
			Language:   opts.language,
			ProjectDir: opts.projectDir,
			BaseDir:    filepath.Dir(opts.projectDir),
			Logger:     info.Logger,
		})
	}
	if err != nil {
		return fmt.Errorf("failed to prepare project: %w", err)
	}

	port, err := findAvailablePort()
	if err != nil {
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	taskQueue := loadgen.TaskQueueForRun(info.RunID)

	// Spawn project server
	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverCmd, err := startProjectProcess(serverCtx, prog, info.Logger, []string{
		"project-server",
		"--port", strconv.Itoa(port),
	})
	if err != nil {
		return fmt.Errorf("failed to spawn project server: %w", err)
	}
	defer stopProjectProcess("project-server", serverCancel, serverCmd, info.Logger)

	// Connect and Init
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
		ConfigJson:               opts.configJSON,
		RegisterSearchAttributes: !info.Configuration.DoNotRegisterSearchAttributes,
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
	if err := opts.language.Set(lang); err != nil {
		return opts, fmt.Errorf("unrecognized language: %s", lang)
	}

	projectDir := info.ScenarioOptions["project-dir"]
	prebuiltDir := info.ScenarioOptions["prebuilt-project-dir"]
	if projectDir == "" && prebuiltDir == "" {
		return opts, fmt.Errorf("either --option project-dir or --option prebuilt-project-dir is required")
	}
	if projectDir != "" && prebuiltDir != "" {
		return opts, fmt.Errorf("cannot specify both project-dir and prebuilt-project-dir")
	}

	if projectDir != "" {
		abs, err := filepath.Abs(projectDir)
		if err != nil {
			return opts, fmt.Errorf("failed to resolve project-dir: %w", err)
		}
		opts.projectDir = abs
	} else {
		abs, err := filepath.Abs(prebuiltDir)
		if err != nil {
			return opts, fmt.Errorf("failed to resolve prebuilt-project-dir: %w", err)
		}
		opts.prebuiltDir = abs
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

func startProjectProcess(ctx context.Context, prog interface {
	NewCommand(context.Context, ...string) (*exec.Cmd, error)
}, logger *zap.SugaredLogger, args []string) (*exec.Cmd, error) {
	cmd, err := prog.NewCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
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
