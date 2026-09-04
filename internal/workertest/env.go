package workertest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/devserver"
	"github.com/temporalio/omes/internal/versions"
	"github.com/temporalio/omes/internal/workerctl"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

const (
	defaultTestNamespace  = "default"
	defaultTestRunTimeout = 30 * time.Second
	workerBuildTimeout    = 1 * time.Minute
	workerShutdownTimeout = 5 * time.Second
)

// Functional options configuration
type testEnvConfig struct {
	executorTimeout time.Duration
	runID           string
	dynamicConfig   map[string]any
	namespace       string
	devServerGroup  string
}

type TestEnvOption func(*testEnvConfig)

// WithExecutorTimeout sets a custom timeout for the executor run context.
func WithExecutorTimeout(d time.Duration) TestEnvOption {
	return func(c *testEnvConfig) {
		c.executorTimeout = d
	}
}

// WithNexusEndpoint instructs the test environment to create a Nexus endpoint.
func WithNexusEndpoint(runID string) TestEnvOption {
	return func(c *testEnvConfig) {
		c.runID = runID
	}
}

// WithDynamicConfig passes dynamic-config overrides to the dev server, on top
// of the test-friendly defaults that devserver applies. Use this for opt-in
// feature gates that a specific test needs.
func WithDynamicConfig(values map[string]any) TestEnvOption {
	return func(c *testEnvConfig) {
		c.dynamicConfig = values
	}
}

// WithNamespace configures the namespace used by the test environment.
func WithNamespace(namespace string) TestEnvOption {
	return func(c *testEnvConfig) {
		c.namespace = namespace
	}
}

// WithDevServerGroup configures the test environment to share a dev server
// with other environments created by the same test using the same group.
func WithDevServerGroup(group string) TestEnvOption {
	return func(c *testEnvConfig) {
		c.devServerGroup = group
	}
}

type TestResult struct {
	ObservedLogs *observer.ObservedLogs
}

// runExecutorConfig holds per-run options for RunExecutorTest.
type runExecutorConfig struct {
	errOnUnimplemented bool
}

type RunExecutorOption func(*runExecutorConfig)

// WithErrOnUnimplemented starts the worker with --err-on-unimplemented, so that
// actions a worker does not support fail loudly instead of being skipped. Use
// it for tests that assert a feature is rejected by a given SDK.
func WithErrOnUnimplemented() RunExecutorOption {
	return func(c *runExecutorConfig) {
		c.errOnUnimplemented = true
	}
}

type TestEnvironment struct {
	testEnvConfig
	devServer         *devserver.Server
	temporalClient    client.Client
	buildID           string
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

func (env *TestEnvironment) Namespace() string {
	return env.namespace
}

func SetupTestEnvironment(t *testing.T, opts ...TestEnvOption) *TestEnvironment {
	cfg := testEnvConfig{
		executorTimeout: defaultTestRunTimeout,
		namespace:       defaultTestNamespace,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	serverRef, err := versions.Get("SERVER_VERSION")
	require.NoError(t, err)
	require.NotEmpty(t, serverRef, "SERVER_VERSION must be set in mise.toml")

	testLogger := zaptest.NewLogger(t)
	serverLogger := testLogger.Named("devserver").Sugar()
	serverOpts := devserver.Options{
		Ref:                 serverRef,
		Namespace:           cfg.namespace,
		DynamicConfigValues: cfg.dynamicConfig,
		Output:              workerctl.NewLogWriter(serverLogger),
		Logger:              serverLogger,
	}

	var server *devserver.Server
	if cfg.devServerGroup == "" {
		server, err = devserver.Start(t.Context(), serverOpts)
		if err == nil {
			t.Cleanup(func() { _ = server.Stop() })
		}
	} else {
		server, err = groupedTestDevServers.acquire(
			t,
			cfg.devServerGroup,
			cfg.namespace,
			serverOpts,
		)
	}
	require.NoError(t, err, "Failed to acquire dev server")

	temporalClient, err := client.Dial(client.Options{
		HostPort:  server.FrontendHostPort(),
		Namespace: cfg.namespace,
	})
	require.NoError(t, err, "Failed to create Temporal client")

	testDir, _ := os.Getwd()
	env := &TestEnvironment{
		testEnvConfig:  cfg,
		devServer:      server,
		temporalClient: temporalClient,
		repoDir:        filepath.Dir(testDir),
		buildID:        uuid.NewString(),
	}
	env.workerPool = NewWorkerPool(env)

	// Optionally create a Nexus endpoint
	if cfg.runID != "" {
		endpointName, err := env.CreateNexusEndpoint(t.Context(), cfg.runID)
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
	env.workerPool.cleanup()
}

func (env *TestEnvironment) CreateNexusEndpoint(ctx context.Context, runID string) (string, error) {
	endpointName := loadgen.NexusEndpointForRun(runID)
	taskQueueName := loadgen.TaskQueueForRun(runID)
	_, err := env.temporalClient.OperatorService().CreateNexusEndpoint(ctx,
		&operatorservice.CreateNexusEndpointRequest{
			Spec: &nexus.EndpointSpec{
				Name: endpointName,
				Target: &nexus.EndpointTarget{
					Variant: &nexus.EndpointTarget_Worker_{
						Worker: &nexus.EndpointTarget_Worker{
							Namespace: env.namespace,
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
	sdk clioptions.Language,
	opts ...RunExecutorOption,
) (TestResult, error) {
	var cfg runExecutorConfig
	for _, opt := range opts {
		opt(&cfg)
	}
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
	scenarioInfo.Namespace = env.namespace

	// Resolve any capability-gated feature options against the dev server, so
	// integration tests exercise the same code path as the CLI.
	if err := loadgen.ResolveFeatureOptions(testCtx, env.temporalClient, env.namespace, scenarioInfo.Options, logger); err != nil {
		return TestResult{ObservedLogs: observedLogs}, err
	}

	taskQueueName := loadgen.TaskQueueForRun(scenarioInfo.RunID)
	workerShutdownCh := env.workerPool.startWorker(testCtx, logger, sdk, taskQueueName, scenarioInfo, cfg.errOnUnimplemented)

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
	return fmt.Sprintf("omes-temp-%s", env.buildID)
}
