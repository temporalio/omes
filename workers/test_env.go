package workers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

const (
	testNamespace         = "default"
	defaultTestRunTimeout = 30 * time.Second
	workerBuildTimeout    = 1 * time.Minute
	workerShutdownTimeout = 5 * time.Second
)

// Functional options configuration
type testEnvConfig struct {
	executorTimeout time.Duration
	nexusTaskQueue  string
}

type TestEnvOption func(*testEnvConfig)

// WithExecutorTimeout sets a custom timeout for the executor run context.
func WithExecutorTimeout(d time.Duration) TestEnvOption {
	return func(c *testEnvConfig) {
		c.executorTimeout = d
	}
}

// WithNexusEndpoint instructs the test environment to create a Nexus endpoint.
func WithNexusEndpoint(taskQueue string) TestEnvOption {
	return func(c *testEnvConfig) {
		c.nexusTaskQueue = taskQueue
	}
}

type TestResult struct {
	ObservedLogs *observer.ObservedLogs
}

type TestEnvironment struct {
	testEnvConfig
	createdAt          time.Time
	devServer          *testsuite.DevServer
	temporalClient     client.Client
	baseDir            string
	buildDir           string
	repoDir            string
	nexusEndpointName  string
	workerMutex        sync.RWMutex
	workerBuildOnce    map[cmdoptions.Language]*sync.Once
	workerBuildErrs    map[cmdoptions.Language]error
	workerCleanupFuncs []func()
}

func (env *TestEnvironment) TemporalClient() client.Client {
	return env.temporalClient
}

func (env *TestEnvironment) DevServerAddress() string {
	return env.devServer.FrontendHostPort()
}

func (env *TestEnvironment) NexusEndpointName() string {
	return env.nexusEndpointName
}

func SetupTestEnvironment(t *testing.T, opts ...TestEnvOption) *TestEnvironment {
	cfg := testEnvConfig{
		executorTimeout: defaultTestRunTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	testLogger := zaptest.NewLogger(t)
	serverLogger := testLogger.Named("devserver").Sugar()
	server, err := testsuite.StartDevServer(t.Context(), testsuite.DevServerOptions{
		ClientOptions: &client.Options{
			Namespace: testNamespace,
		},
		LogLevel: "error",
		Stdout:   &logWriter{logger: serverLogger},
		Stderr:   &logWriter{logger: serverLogger},
	})
	require.NoError(t, err, "Failed to start dev server")

	temporalClient, err := client.Dial(client.Options{
		HostPort:  server.FrontendHostPort(),
		Namespace: testNamespace,
	})
	require.NoError(t, err, "Failed to create Temporal client")

	testDir, _ := os.Getwd()
	repoDir := filepath.Dir(testDir)

	env := &TestEnvironment{
		testEnvConfig:   cfg,
		devServer:       server,
		temporalClient:  temporalClient,
		repoDir:         repoDir,
		createdAt:       time.Now(),
		workerBuildOnce: map[cmdoptions.Language]*sync.Once{},
		workerBuildErrs: map[cmdoptions.Language]error{},
	}

	// Optionally create a Nexus endpoint
	if cfg.nexusTaskQueue != "" {
		endpointName, err := env.createNexusEndpoint(t.Context(), cfg.nexusTaskQueue)
		require.NoError(t, err, "Failed to create Nexus endpoint")
		env.nexusEndpointName = endpointName
	}

	t.Cleanup(func() {
		env.cleanup()
	})

	return env
}

func (env *TestEnvironment) cleanup() {
	if env.temporalClient != nil {
		env.temporalClient.Close()
	}
	if env.devServer != nil {
		env.devServer.Stop()
	}

	env.workerMutex.Lock()
	defer env.workerMutex.Unlock()
	for _, cleanupFunc := range env.workerCleanupFuncs {
		cleanupFunc()
	}
}

func (env *TestEnvironment) createNexusEndpoint(ctx context.Context, taskQueueName string) (string, error) {
	endpointName := fmt.Sprintf("test-nexus-endpoint-%d", time.Now().Unix())
	_, err := env.temporalClient.OperatorService().CreateNexusEndpoint(ctx,
		&operatorservice.CreateNexusEndpointRequest{
			Spec: &nexus.EndpointSpec{
				Name: endpointName,
				Target: &nexus.EndpointTarget{
					Variant: &nexus.EndpointTarget_Worker_{
						Worker: &nexus.EndpointTarget_Worker{
							Namespace: testNamespace,
							TaskQueue: taskQueueName,
						},
					},
				},
			},
		})
	if err != nil {
		return "", err
	}
	return endpointName, nil
}

// RunExecutorTest runs an executor with a specific SDK and server address
func (env *TestEnvironment) RunExecutorTest(
	t *testing.T,
	executor loadgen.Executor,
	scenarioInfo loadgen.ScenarioInfo,
	sdk cmdoptions.Language,
) (TestResult, error) {
	testLogger := zaptest.NewLogger(t).Core()
	observeLogger, observedLogs := observer.New(zap.DebugLevel)
	logger := zap.New(zapcore.NewTee(testLogger, observeLogger)).Sugar()

	env.ensureWorkerBuilt(t, logger, sdk)

	testCtx, cancelTestCtx := context.WithTimeout(t.Context(), env.executorTimeout)
	defer cancelTestCtx()

	// Update scenario info with test environment details
	scenarioInfo.Logger = logger.Named("executor")
	scenarioInfo.MetricsHandler = client.MetricsNopHandler
	scenarioInfo.Client = env.temporalClient
	scenarioInfo.Namespace = testNamespace

	taskQueueName := loadgen.TaskQueueForRun(scenarioInfo.RunID)
	workerDone := env.startWorker(testCtx, logger, sdk, taskQueueName, scenarioInfo)

	execErr := executor.Run(testCtx, scenarioInfo)

	// Trigger worker shutdown.
	cancelTestCtx()

	// Wait for worker shutdown.
	var workerErr error
	select {
	case workerErr = <-workerDone:
		if workerErr != nil {
			t.Logf("Worker shutdown with error: %v", workerErr)
		}
	case <-time.After(2 * workerShutdownTimeout):
		workerErr = fmt.Errorf("timed out waiting for worker shutdown")
	}

	return TestResult{ObservedLogs: observedLogs}, errors.Join(execErr, workerErr)
}

func (env *TestEnvironment) ensureWorkerBuilt(
	t *testing.T,
	logger *zap.SugaredLogger,
	sdk cmdoptions.Language,
) {
	env.workerMutex.Lock()
	once, exists := env.workerBuildOnce[sdk]
	if !exists {
		once = new(sync.Once)
		env.workerBuildOnce[sdk] = once
	}
	env.workerMutex.Unlock()

	once.Do(func() {
		baseDir := BaseDir(env.repoDir, sdk)
		dirName := env.buildDirName()
		buildDir := filepath.Join(baseDir, dirName)

		env.workerMutex.Lock()
		env.workerCleanupFuncs = append(env.workerCleanupFuncs, func() {
			if err := os.RemoveAll(buildDir); err != nil {
				fmt.Printf("Failed to clean up build dir for %s at %s: %v\n", sdk, buildDir, err)
			}
		})
		env.workerMutex.Unlock()

		builder := Builder{
			DirName:    dirName,
			SdkOptions: cmdoptions.SdkOptions{Language: sdk},
			Logger:     logger.Named(fmt.Sprintf("%s-builder", sdk)),
		}

		buildCtx, buildCancel := context.WithTimeout(t.Context(), workerBuildTimeout)
		defer buildCancel()

		_, err := builder.Build(buildCtx, baseDir)

		env.workerMutex.Lock()
		env.workerBuildErrs[sdk] = err
		env.workerMutex.Unlock()
	})

	env.workerMutex.RLock()
	err := env.workerBuildErrs[sdk]
	env.workerMutex.RUnlock()

	require.NoError(t, err, "Failed to build worker for SDK %s", sdk)
}

func (env *TestEnvironment) startWorker(
	ctx context.Context,
	logger *zap.SugaredLogger,
	sdk cmdoptions.Language,
	taskQueueName string,
	scenarioInfo loadgen.ScenarioInfo,
) <-chan error {
	workerDone := make(chan error, 1)

	go func() {
		defer close(workerDone)
		baseDir := BaseDir(env.repoDir, sdk)
		runner := &Runner{
			Builder: Builder{
				DirName:    env.buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     logger.Named(fmt.Sprintf("%s-worker-builder", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID: cmdoptions.ScenarioID{
				Scenario: scenarioInfo.ScenarioName,
				RunID:    scenarioInfo.RunID,
			},
			ClientOptions: cmdoptions.ClientOptions{
				Address:   env.DevServerAddress(),
				Namespace: testNamespace,
			},
			LoggingOptions: cmdoptions.LoggingOptions{
				PreparedLogger: logger.Named(fmt.Sprintf("%s-worker", sdk)),
			},
		}
		workerDone <- runner.Run(ctx, baseDir)
	}()

	return workerDone
}

func (env *TestEnvironment) buildDirName() string {
	return fmt.Sprintf("omes-temp-%d", env.createdAt.UnixMilli())
}
