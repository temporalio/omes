package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestValidateRequiresLanguage(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"project-name": "helloworld",
		},
	})
	require.EqualError(t, err, "--option language=<lang> is required")
}

func TestValidateRejectsConflictingProjectSources(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"language":             "python",
			"project-name":         "helloworld",
			"prebuilt-project-dir": "workers/python/projects/tests/project-build-helloworld",
		},
	})
	require.EqualError(t, err, "cannot specify both project-name and prebuilt-project-dir")
}

func TestPythonHelloWorld(t *testing.T) {
	runProjectScenario(t, "python", "helloworld", "", nil)
}

func runProjectScenario(t *testing.T, lang, projectName, version string, config []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	sugar := logger.Sugar()

	metrics := (&clioptions.MetricsOptions{}).MustCreateMetrics(ctx, sugar)
	runID := fmt.Sprintf("test-%s-%s-%d", lang, projectName, time.Now().UnixNano())
	defer func() {
		require.NoError(t, metrics.Shutdown(ctx, sugar, "project", runID, ""))
	}()

	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		LogLevel: "error",
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	})
	require.NoError(t, err)
	defer server.Stop()

	hostPort := server.FrontendHostPort()
	namespace := "default"
	taskQueue := loadgen.TaskQueueForRun(runID)
	executionID := fmt.Sprintf("exec-%d", time.Now().UnixNano())

	var language clioptions.Language
	require.NoError(t, language.Set(lang))

	clientOptions := clioptions.ClientOptions{
		Address:   hostPort,
		Namespace: namespace,
	}
	client, err := clientOptions.Dial(metrics, sugar)
	require.NoError(t, err)
	defer client.Close()

	prog, err := buildProject(ctx, repoRoot, projectScenarioOptions{
		sdkOpts: clioptions.SdkOptions{
			Language: language,
			Version:  version,
		},
		projectName: projectName,
	}, sugar)
	require.NoError(t, err)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	workerCmd, err := startWorkerProcess(workerCtx, prog, taskQueue, clientOptions)
	require.NoError(t, err)

	workerErrCh := make(chan error, 1)
	go func() {
		workerErrCh <- workerCmd.Wait()
	}()

	info := loadgen.ScenarioInfo{
		ScenarioName:   "project",
		RunID:          runID,
		ExecutionID:    executionID,
		MetricsHandler: metrics.NewHandler(),
		Logger:         sugar,
		Client:         client,
		ClientOptions:  clientOptions,
		Configuration: loadgen.RunConfiguration{
			Iterations:                    1,
			DoNotRegisterSearchAttributes: true,
		},
		ScenarioOptions: map[string]string{
			"language":     lang,
			"project-name": projectName,
			"version":      version,
		},
		Namespace: namespace,
		RootPath:  repoRoot,
	}
	if config != nil {
		configFile, err := os.CreateTemp(t.TempDir(), "project-config-*.json")
		require.NoError(t, err)
		_, err = configFile.Write(config)
		require.NoError(t, err)
		require.NoError(t, configFile.Close())
		info.ScenarioOptions["project-config-file"] = configFile.Name()
	}

	scenarioErrCh := make(chan error, 1)
	go func() {
		scenarioErrCh <- (&projectScenarioExecutor{}).Run(ctx, info)
	}()

	select {
	case err := <-workerErrCh:
		require.FailNowf(t, "project worker exited before scenario completed", "%v", err)
	case err := <-scenarioErrCh:
		require.NoError(t, err, "project scenario failed for %s/%s", lang, projectName)
	}

	stopWorkerProcess(t, workerCancel, workerErrCh, workerCmd)
}

func startWorkerProcess(ctx context.Context, prog sdkbuild.Program, taskQueue string, clientOptions clioptions.ClientOptions) (*exec.Cmd, error) {
	args := []string{
		"main",
		"worker",
		"--task-queue", taskQueue,
		"--server-address", clientOptions.Address,
		"--namespace", clientOptions.Namespace,
	}
	cmd, err := prog.NewCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
	}
	cmd.WaitDelay = 3 * time.Second
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd, cmd.Start()
}

func stopWorkerProcess(t *testing.T, cancel context.CancelFunc, workerErrCh <-chan error, cmd *exec.Cmd) {
	t.Helper()
	cancel()

	if waitForWorkerExit(workerErrCh, 3*time.Second) {
		return
	}

	require.NoError(t, syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL))

	if waitForWorkerExit(workerErrCh, 3*time.Second) {
		return
	}

	t.Fatal("timed out waiting for project worker to exit")
}

func waitForWorkerExit(workerErrCh <-chan error, timeout time.Duration) bool {
	select {
	case <-workerErrCh:
		return true
	case <-time.After(timeout):
		return false
	}
}
