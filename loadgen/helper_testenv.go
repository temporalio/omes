package loadgen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/workers"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	testNamespace         = "default"
	testRunTimeout        = 30 * time.Second
	workerBuildTimeout    = 1 * time.Minute
	workerShutdownTimeout = 5 * time.Second
)

var (
	testStart       = time.Now().Unix()
	workerMutex     sync.RWMutex
	workerBuildOnce = map[cmdoptions.Language]*sync.Once{
		cmdoptions.LangGo:         new(sync.Once),
		cmdoptions.LangJava:       new(sync.Once),
		cmdoptions.LangPython:     new(sync.Once),
		cmdoptions.LangTypeScript: new(sync.Once),
		cmdoptions.LangDotNet:     new(sync.Once),
	}
	workerBuildErrs    = map[cmdoptions.Language]error{}
	workerCleanupFuncs []func()
)

// TestEnvironment encapsulates the test infrastructure needed to run scenarios
type TestEnvironment struct {
	DevServer      *testsuite.DevServer
	TemporalClient client.Client
	Logger         *zap.SugaredLogger
	BaseDir        string
	BuildDir       string
	RepoDir        string
}



// SetupTestEnvironment creates a test environment with dev server and client
func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	// Start dev server
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: testNamespace,
		},
		LogLevel: "error",
	})
	require.NoError(t, err, "Failed to start dev server")

	// Create Temporal client
	temporalClient, err := client.Dial(client.Options{
		HostPort:  server.FrontendHostPort(),
		Namespace: testNamespace,
	})
	require.NoError(t, err, "Failed to create Temporal client")

	// Setup directories
	testDir, _ := os.Getwd()
	repoDir := filepath.Dir(testDir)

	logger := zaptest.NewLogger(t).Sugar()

	env := &TestEnvironment{
		DevServer:      server,
		TemporalClient: temporalClient,
		Logger:         logger,
		RepoDir:        repoDir,
	}

	// Register cleanup
	t.Cleanup(func() {
		env.Cleanup()
	})

	return env
}

// Cleanup cleans up the test environment
func (env *TestEnvironment) Cleanup() {
	if env.TemporalClient != nil {
		env.TemporalClient.Close()
	}
	if env.DevServer != nil {
		env.DevServer.Stop()
	}
	
	// Clean up build directories
	workerMutex.Lock()
	defer workerMutex.Unlock()
	for _, cleanupFunc := range workerCleanupFuncs {
		cleanupFunc()
	}
}

// RunExecutorTest runs an executor with the given scenario info and validates it completes successfully
func (env *TestEnvironment) RunExecutorTest(t *testing.T, executor Executor, scenarioInfo ScenarioInfo) {
	// Use Go SDK by default
	sdk := cmdoptions.LangGo

	// Build worker
	env.ensureWorkerBuilt(t, sdk)

	// Setup scenario
	scenarioID := cmdoptions.ScenarioID{
		Scenario: scenarioInfo.ScenarioName,
		RunID:    scenarioInfo.RunID,
	}
	taskQueueName := TaskQueueForRun(scenarioID.Scenario, scenarioID.RunID)

	// Create test context with timeout
	ctx, cancel := context.WithTimeout(t.Context(), testRunTimeout)
	defer cancel()

	// Start worker
	workerDone := env.startWorker(t, sdk, taskQueueName, scenarioID)

	// Update scenario info with test environment details
	scenarioInfo.Logger = env.Logger.Named("executor")
	scenarioInfo.MetricsHandler = client.MetricsNopHandler
	scenarioInfo.Client = env.TemporalClient
	scenarioInfo.Namespace = testNamespace

	// Run the executor
	err := executor.Run(ctx, scenarioInfo)

	// Cancel context to trigger worker shutdown
	cancel()

	// Wait for worker shutdown with timeout
	select {
	case workerErr := <-workerDone:
		if workerErr != nil {
			t.Logf("Worker shutdown with error: %v", workerErr)
		}
	case <-time.After(5 * time.Second):
		t.Logf("Worker shutdown timed out, continuing...")
	}

	require.NoError(t, err, "Executor should complete successfully")
}

// runExecutorTestWithSDK runs an executor with a specific SDK and server address
func (env *TestEnvironment) runExecutorTestWithSDK(t *testing.T, executor Executor, scenarioInfo ScenarioInfo, sdk cmdoptions.Language, devServerAddr string) error {
	// Build worker
	env.ensureWorkerBuilt(t, sdk)

	// Setup scenario
	scenarioID := cmdoptions.ScenarioID{
		Scenario: scenarioInfo.ScenarioName,
		RunID:    scenarioInfo.RunID,
	}
	taskQueueName := TaskQueueForRun(scenarioID.Scenario, scenarioID.RunID)

	// Create test context with timeout
	ctx, cancel := context.WithTimeout(t.Context(), testRunTimeout)
	defer cancel()

	// Start worker with specific address
	workerDone := env.startWorkerWithAddress(t, sdk, taskQueueName, scenarioID, devServerAddr)

	// Update scenario info with test environment details
	scenarioInfo.Logger = env.Logger.Named("executor")
	scenarioInfo.MetricsHandler = client.MetricsNopHandler
	scenarioInfo.Client = env.TemporalClient
	scenarioInfo.Namespace = testNamespace

	// Run the executor
	err := executor.Run(ctx, scenarioInfo)

	// Cancel context to trigger worker shutdown
	cancel()

	// Wait for worker shutdown with timeout
	select {
	case workerErr := <-workerDone:
		if workerErr != nil {
			t.Logf("Worker shutdown with error: %v", workerErr)
		}
	case <-time.After(5 * time.Second):
		t.Logf("Worker shutdown timed out, continuing...")
	}

	return err
}

// ensureWorkerBuilt ensures the worker is built for the given SDK
func (env *TestEnvironment) ensureWorkerBuilt(t *testing.T, sdk cmdoptions.Language) {
	once := workerBuildOnce[sdk]
	once.Do(func() {
		baseDir := workers.BaseDir(env.RepoDir, sdk)
		dirName := buildDirName()
		buildDir := filepath.Join(baseDir, dirName)

		workerMutex.Lock()
		workerCleanupFuncs = append(workerCleanupFuncs, func() {
			if err := os.RemoveAll(buildDir); err != nil {
				fmt.Printf("Failed to clean up build dir for %s at %s: %v\n", sdk, buildDir, err)
			}
		})
		workerMutex.Unlock()

		builder := workers.Builder{
			DirName:    dirName,
			SdkOptions: cmdoptions.SdkOptions{Language: sdk},
			Logger:     env.Logger.Named(fmt.Sprintf("%s-builder", sdk)),
		}

		buildCtx, buildCancel := context.WithTimeout(t.Context(), workerBuildTimeout)
		defer buildCancel()

		_, err := builder.Build(buildCtx, baseDir)

		workerMutex.Lock()
		workerBuildErrs[sdk] = err
		workerMutex.Unlock()
	})

	workerMutex.RLock()
	err := workerBuildErrs[sdk]
	workerMutex.RUnlock()

	require.NoError(t, err, "Failed to build worker for SDK %s", sdk)
}

// startWorker starts a worker for the given SDK and returns a channel that will receive any error
func (env *TestEnvironment) startWorker(t *testing.T, sdk cmdoptions.Language, taskQueueName string, scenarioID cmdoptions.ScenarioID) <-chan error {
	workerDone := make(chan error, 1)
	
	go func() {
		defer close(workerDone)
		baseDir := workers.BaseDir(env.RepoDir, sdk)
		runner := &workers.Runner{
			Builder: workers.Builder{
				DirName:    buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     env.Logger.Named(fmt.Sprintf("%s-worker", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID:               scenarioID,
			ClientOptions: cmdoptions.ClientOptions{
				Address:   env.DevServer.FrontendHostPort(),
				Namespace: testNamespace,
			},
		}
		workerDone <- runner.Run(t.Context(), baseDir)
	}()
	
	return workerDone
}

// startWorkerWithAddress starts a worker with a specific server address
func (env *TestEnvironment) startWorkerWithAddress(t *testing.T, sdk cmdoptions.Language, taskQueueName string, scenarioID cmdoptions.ScenarioID, serverAddr string) <-chan error {
	workerDone := make(chan error, 1)
	
	go func() {
		defer close(workerDone)
		baseDir := workers.BaseDir(env.RepoDir, sdk)
		runner := &workers.Runner{
			Builder: workers.Builder{
				DirName:    buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     env.Logger.Named(fmt.Sprintf("%s-worker", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID:               scenarioID,
			ClientOptions: cmdoptions.ClientOptions{
				Address:   serverAddr,
				Namespace: testNamespace,
			},
		}
		workerDone <- runner.Run(t.Context(), baseDir)
	}()
	
	return workerDone
}

func buildDirName() string {
	return fmt.Sprintf("omes-temp-%d", testStart)
}