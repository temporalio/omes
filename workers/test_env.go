package workers

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	createdAt         time.Time
	devServer         *testsuite.DevServer
	temporalClient    client.Client
	baseDir           string
	buildDir          string
	repoDir           string
	nexusEndpointName string
	workerPool        *workerPool
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
		testEnvConfig:  cfg,
		devServer:      server,
		temporalClient: temporalClient,
		repoDir:        repoDir,
		createdAt:      time.Now(),
	}
	env.workerPool = NewWorkerPool(env)

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
	env.workerPool.cleanup()
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

	err := env.workerPool.ensureWorkerBuilt(t, logger, sdk)
	if err != nil {
		return TestResult{ObservedLogs: observedLogs},
			fmt.Errorf("Failed to build worker for SDK %s: %w", sdk, err)
	}

	testCtx, cancelTestCtx := context.WithTimeout(t.Context(), env.executorTimeout)
	defer cancelTestCtx()

	// Update scenario info with test environment details
	scenarioInfo.Logger = logger.Named("executor")
	scenarioInfo.MetricsHandler = client.MetricsNopHandler
	scenarioInfo.Client = env.temporalClient
	scenarioInfo.Namespace = testNamespace

	taskQueueName := loadgen.TaskQueueForRun(scenarioInfo.RunID)
	workerShutdownCh := env.workerPool.startWorker(testCtx, logger, sdk, taskQueueName, scenarioInfo)

	execErr := executor.Run(testCtx, scenarioInfo)

	// Trigger worker shutdown.
	cancelTestCtx()

	// Wait for worker shutdown.
	var workerErr error
	select {
	case workerErr = <-workerShutdownCh:
		if workerErr != nil {
			t.Logf("Worker shutdown with error: %v", workerErr)
		}
	case <-time.After(2 * workerShutdownTimeout):
		workerErr = fmt.Errorf("timed out waiting for worker shutdown")
	}

	return TestResult{ObservedLogs: observedLogs},
		errors.Join(execErr, workerErr)
}

func (env *TestEnvironment) buildDirName() string {
	return fmt.Sprintf("omes-temp-%d", env.createdAt.UnixMilli())
}
