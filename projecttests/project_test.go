package projecttests

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/internal/programbuild"
	"github.com/temporalio/omes/internal/utils"
	"github.com/temporalio/omes/loadgen/projectexecutor"
	"github.com/temporalio/omes/projecttests/go/harness/api"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
)

func TestHelloworld(t *testing.T) {
	runProjectTest(t, "helloworld", nil)
}

func TestThroughputstress(t *testing.T) {
	config := []byte(`{
		"internal_iterations": 2,
		"continue_as_new_after_iter": 0,
		"sleep_duration": "100ms",
		"payload_size_bytes": 64
	}`)
	runProjectTest(t, "throughputstress", config)
}

func runProjectTest(t *testing.T, projectName string, config []byte) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Start dev server
	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		LogLevel: "error",
		Stdout:   os.Stderr,
		Stderr:   os.Stderr,
	})
	require.NoError(t, err)
	defer server.Stop()

	hostPort := server.FrontendHostPort()
	namespace := "default"
	taskQueue := fmt.Sprintf("test-%s-%d", projectName, time.Now().UnixNano())

	// Build the project binary
	projectDir := filepath.Join("go", "tests", projectName)
	logger, _ := zap.NewDevelopment()
	sugar := logger.Sugar()

	builder := programbuild.ProgramBuilder{
		Language:   "go",
		ProjectDir: projectDir,
		Logger:     sugar,
	}
	prog, err := builder.BuildProgram(ctx, "")
	require.NoError(t, err, "failed to build project %s", projectName)

	// Start worker process
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	workerCmd, err := programbuild.StartProgramProcess(workerCtx, prog, []string{
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

	// Start project-server process
	var serverPort int
	require.NoError(t, utils.DefaultPort(&serverPort))

	serverCtx, serverCancel := context.WithCancel(ctx)
	defer serverCancel()

	serverCmd, err := programbuild.StartProgramProcess(serverCtx, prog, []string{
		"project-server",
		"--port", strconv.Itoa(serverPort),
	})
	require.NoError(t, err, "failed to start project server")
	defer func() {
		serverCancel()
		serverCmd.Wait()
	}()

	// Connect to project server and call Init
	handle, err := projectexecutor.NewProjectHandle(ctx, serverPort, &api.InitRequest{
		ExecutionId: fmt.Sprintf("test-%d", time.Now().UnixNano()),
		RunId:       "test-run",
		TaskQueue:   taskQueue,
		ConnectOptions: &api.ConnectOptions{
			Namespace:     namespace,
			ServerAddress: hostPort,
		},
		RegisterSearchAttributes: true,
		Config:                   config,
	})
	require.NoError(t, err, "failed to init project handle")
	defer handle.Close()

	// Execute one iteration
	_, err = handle.Execute(ctx, &api.ExecuteRequest{
		Iteration: 1,
	})
	require.NoError(t, err, "execute failed for project %s", projectName)
}
