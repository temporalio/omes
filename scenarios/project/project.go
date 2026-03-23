package project

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers/go/projects/api"
	"go.uber.org/zap"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a project test via gRPC harness (--option language=<lang> --option test=<name>)",
		ExecutorFn: func() loadgen.Executor {
			return &projectScenarioExecutor{}
		},
	})
}

type projectScenarioOptions struct {
	language   clioptions.Language
	testName   string
	projectDir string
	configJSON []byte
}

// projectScenarioExecutor wraps the full project test lifecycle:
// build project binary, spawn processes, gRPC init, execute iterations, cleanup.
type projectScenarioExecutor struct{}

func (e *projectScenarioExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	projectOpts, err := e.validate(info)
	if err != nil {
		return err
	}

	info.Logger.Infof("Building project %s/%s", projectOpts.language, projectOpts.testName)

	baseDir := filepath.Join(info.RootPath, "workers", projectOpts.language.String(), "projects/tests")
	prog, err := Build(ctx, BuildOptions{
		Language:   projectOpts.language,
		ProjectDir: projectOpts.projectDir,
		BaseDir:    baseDir,
		Logger:     info.Logger,
	})
	if err != nil {
		return fmt.Errorf("failed to build project: %w", err)
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
		ConfigJson:               projectOpts.configJSON,
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
	opts := projectScenarioOptions{}
	testName := info.ScenarioOptions["test"]
	if testName == "" {
		return opts, fmt.Errorf("--option test=<name> is required for the project scenario")
	}
	opts.testName = testName

	lang := info.ScenarioOptions["language"]
	if lang == "" {
		return opts, fmt.Errorf("--option language=<lang> is required for the project scenario")
	}
	err := opts.language.Set(lang)
	if err != nil {
		return opts, fmt.Errorf("unrecognized language provided: %s", lang)
	}

	projectDir := filepath.Join(info.RootPath, "workers", lang, "projects/tests", testName)
	if _, statErr := os.Stat(projectDir); statErr != nil {
		return opts, fmt.Errorf("project not found at %s: %w", projectDir, statErr)
	}
	opts.projectDir = projectDir

	// Read config file if provided
	var configJSON []byte
	if configPath := info.ScenarioOptions["config-file"]; configPath != "" {
		_, err := os.ReadFile(configPath)
		if err != nil {
			return opts, fmt.Errorf("failed to read config file %s: %w", configPath, err)
		}
	}
	opts.configJSON = configJSON
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
