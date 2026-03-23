package project

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers/go/projects/api"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestGoHelloworld(t *testing.T) {
	runProjectTest(t, "go", "helloworld", nil)
}

func runProjectTest(t *testing.T, lang, testName string, config []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		LogLevel: "error",
		Stdout:   os.Stderr,
		Stderr:   os.Stderr,
	})
	require.NoError(t, err)
	defer server.Stop()

	hostPort := server.FrontendHostPort()
	namespace := "default"
	taskQueue := fmt.Sprintf("test-%s-%s-%d", lang, testName, time.Now().UnixNano())

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	projectDir := filepath.Join(repoRoot, "workers", lang, "projects/tests", testName)
	baseDir := filepath.Join(repoRoot, "workers", lang, "projects/tests")

	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	var language clioptions.Language
	language.Set(lang)
	prog, err := Build(ctx, BuildOptions{
		Language:   language,
		ProjectDir: projectDir,
		BaseDir:    baseDir,
		Logger:     sugar,
	})
	require.NoError(t, err, "failed to build project %s/%s", lang, testName)

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	workerCmd, err := startTestProcess(workerCtx, prog, []string{
		"worker",
		"--task-queue", taskQueue,
		"--server-address", hostPort,
		"--namespace", namespace,
	})
	require.NoError(t, err, "failed to start worker")
	defer func() {
		workerCancel()
		workerCmd.Wait()
	}()

	serverPort, err := findAvailablePort()
	require.NoError(t, err)

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverCmd, err := startTestProcess(serverCtx, prog, []string{
		"project-server",
		"--port", strconv.Itoa(serverPort),
	})
	require.NoError(t, err, "failed to start project server")
	defer func() {
		serverCancel()
		serverCmd.Wait()
	}()

	handle, err := NewProjectHandle(ctx, serverPort, &api.InitRequest{
		ExecutionId:              fmt.Sprintf("test-%d", time.Now().UnixNano()),
		RunId:                    "test-run",
		TaskQueue:                taskQueue,
		ConnectOptions:           &api.ConnectOptions{Namespace: namespace, ServerAddress: hostPort},
		RegisterSearchAttributes: true,
		ConfigJson:               config,
	})
	require.NoError(t, err, "failed to init project handle")
	defer handle.Close()

	_, err = handle.Execute(ctx, &api.ExecuteRequest{Iteration: 1})
	require.NoError(t, err, "execute failed for project %s/%s", lang, testName)
}

func startTestProcess(ctx context.Context, prog interface{ NewCommand(context.Context, ...string) (*exec.Cmd, error) }, args []string) (*exec.Cmd, error) {
	cmd, err := prog.NewCommand(ctx, args...)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
