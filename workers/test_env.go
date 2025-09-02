package workers

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
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

const (
	testNamespace         = "default"
	defaultTestRunTimeout = 30 * time.Second
	workerBuildTimeout    = 1 * time.Minute
	workerShutdownTimeout = 5 * time.Second
)

var (
	testStart          = time.Now().Unix()
	workerMutex        sync.RWMutex
	workerBuildOnce    = map[cmdoptions.Language]*sync.Once{}
	workerBuildErrs    = map[cmdoptions.Language]error{}
	workerCleanupFuncs []func()
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

type TestEnvironment struct {
	testEnvConfig
	devServer         *testsuite.DevServer
	temporalClient    client.Client
	logger            *zap.SugaredLogger
	baseDir           string
	buildDir          string
	repoDir           string
	nexusEndpointName string
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
	logger := zaptest.NewLogger(t).Sugar()

	cfg := testEnvConfig{
		executorTimeout: defaultTestRunTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	serverLogger := logger.Named("devserver")
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
		logger:         logger,
		repoDir:        repoDir,
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

	workerMutex.Lock()
	defer workerMutex.Unlock()
	for _, cleanupFunc := range workerCleanupFuncs {
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
) error {
	env.ensureWorkerBuilt(t, sdk)

	scenarioID := cmdoptions.ScenarioID{
		Scenario: scenarioInfo.ScenarioName,
		RunID:    scenarioInfo.RunID,
	}
	taskQueueName := loadgen.TaskQueueForRun(scenarioID.Scenario, scenarioID.RunID)

	testCtx, cancelTestCtx := context.WithTimeout(t.Context(), env.executorTimeout)
	defer cancelTestCtx()

	workerDone := env.startWorker(t, sdk, taskQueueName, scenarioID)

	// Update scenario info with test environment details
	scenarioInfo.Logger = env.logger.Named("executor")
	scenarioInfo.MetricsHandler = client.MetricsNopHandler
	scenarioInfo.Client = env.temporalClient
	scenarioInfo.Namespace = testNamespace

	err := executor.Run(testCtx, scenarioInfo)

	// Trigger worker shutdown.
	cancelTestCtx()

	// Wait for worker shutdown.
	select {
	case workerErr := <-workerDone:
		if workerErr != nil {
			t.Logf("Worker shutdown with error: %v", workerErr)
		}
	case <-time.After(2 * workerShutdownTimeout):
		t.Logf("Worker shutdown timed out, continuing...")
	}

	return err
}

func (env *TestEnvironment) ensureWorkerBuilt(t *testing.T, sdk cmdoptions.Language) {
	workerMutex.Lock()
	once, exists := workerBuildOnce[sdk]
	if !exists {
		once = new(sync.Once)
		workerBuildOnce[sdk] = once
	}
	workerMutex.Unlock()

	once.Do(func() {
		baseDir := BaseDir(env.repoDir, sdk)
		dirName := buildDirName()
		buildDir := filepath.Join(baseDir, dirName)

		workerMutex.Lock()
		workerCleanupFuncs = append(workerCleanupFuncs, func() {
			if err := os.RemoveAll(buildDir); err != nil {
				fmt.Printf("Failed to clean up build dir for %s at %s: %v\n", sdk, buildDir, err)
			}
		})
		workerMutex.Unlock()

		builder := Builder{
			DirName:    dirName,
			SdkOptions: cmdoptions.SdkOptions{Language: sdk},
			Logger:     env.logger.Named(fmt.Sprintf("%s-builder", sdk)),
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

func (env *TestEnvironment) startWorker(
	t *testing.T,
	sdk cmdoptions.Language,
	taskQueueName string,
	scenarioID cmdoptions.ScenarioID,
) <-chan error {
	workerDone := make(chan error, 1)

	go func() {
		defer close(workerDone)
		baseDir := BaseDir(env.repoDir, sdk)
		runner := &Runner{
			Builder: Builder{
				DirName:    buildDirName(),
				SdkOptions: cmdoptions.SdkOptions{Language: sdk},
				Logger:     env.logger.Named(fmt.Sprintf("%s-worker", sdk)),
			},
			TaskQueueName:            taskQueueName,
			GracefulShutdownDuration: workerShutdownTimeout,
			ScenarioID:               scenarioID,
			ClientOptions: cmdoptions.ClientOptions{
				Address:   env.DevServerAddress(),
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
