package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	"go.uber.org/zap/zaptest"
)

func TestValidateRequiresLanguage(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"project-name": "helloworld",
		},
	})
	require.EqualError(t, err, "--option language=<lang> is required")
}

func TestValidateLimitedProjectLanguageSupport(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"language": "go",
		},
	})
	require.EqualError(t, err, "project scenario is currently limited to Python and TypeScript, got go")
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

func TestPythonHelloWorldSourceBuild(t *testing.T) {
	runProjectScenario(t, "python", "helloworld", "", nil, false)
}

func TestPythonHelloWorldPrebuilt(t *testing.T) {
	runProjectScenario(t, "python", "helloworld", "", nil, true)
}

func TestTypeScriptHelloWorldSourceBuild(t *testing.T) {
	runProjectScenario(t, "typescript", "helloworld", "", nil, false)
}

func TestTypeScriptHelloWorldPrebuilt(t *testing.T) {
	runProjectScenario(t, "typescript", "helloworld", "", nil, true)
}

func runProjectScenario(
	t *testing.T,
	lang, projectName, version string,
	config []byte,
	usePrebuilt bool,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Minute)
	defer cancel()

	client, clientOptions := startServerAndClient(t, ctx)
	info := buildScenarioInfo(t, client, clientOptions, lang, projectName, version, config)
	opts, err := (&projectScenarioExecutor{}).validate(info)
	require.NoError(t, err)

	var prog sdkbuild.Program
	if usePrebuilt {
		prog, err = buildProject(ctx, info.RootPath, opts, info.Logger)
		require.NoError(t, err)
		info.ScenarioOptions["project-name"] = ""
		info.ScenarioOptions["prebuilt-project-dir"] = prog.Dir()
	}

	workerErrCh := startProjectWorker(
		t,
		ctx,
		info,
		opts,
		prog,
	)

	scenarioErrCh := make(chan error, 1)
	go func() {
		scenarioErrCh <- (&projectScenarioExecutor{
			clientReadyTimeout: 30 * time.Second,
		}).Run(ctx, info)
	}()

	select {
	case err := <-workerErrCh:
		require.FailNowf(t, "project worker exited before scenario completed", "%v", err)
	case err := <-scenarioErrCh:
		require.NoError(t, err, "project scenario failed for %s/%s", lang, projectName)
	}

	cancel()

	select {
	case err := <-workerErrCh:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for project worker to exit")
	}
}

func startServerAndClient(
	t *testing.T,
	ctx context.Context,
) (sdkclient.Client, clioptions.ClientOptions) {
	t.Helper()

	server, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		ClientOptions: &sdkclient.Options{
			Namespace: "default",
		},
	})
	require.NoError(t, err)

	clientOptions := clioptions.ClientOptions{
		Address:   server.FrontendHostPort(),
		Namespace: "default",
	}
	temporalClient, err := sdkclient.Dial(sdkclient.Options{
		HostPort:  clientOptions.Address,
		Namespace: clientOptions.Namespace,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		temporalClient.Close()
		server.Stop()
	})
	return temporalClient, clientOptions
}

func buildScenarioInfo(
	t *testing.T,
	client sdkclient.Client,
	clientOptions clioptions.ClientOptions,
	lang, projectName, version string,
	config []byte,
) loadgen.ScenarioInfo {
	t.Helper()

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	info := loadgen.ScenarioInfo{
		ScenarioName:   "project",
		RunID:          fmt.Sprintf("test-%s-%s-%d", lang, projectName, time.Now().UnixNano()),
		ExecutionID:    fmt.Sprintf("exec-%d", time.Now().UnixNano()),
		MetricsHandler: sdkclient.MetricsNopHandler,
		Logger:         zaptest.NewLogger(t).Sugar(),
		Client:         client,
		ClientOptions:  clientOptions,
		Configuration: loadgen.RunConfiguration{
			Iterations:                    1,
			DoNotRegisterSearchAttributes: true,
		},
		ScenarioOptions: map[string]string{
			"language":     lang,
			"project-name": projectName,
		},
		Namespace: clientOptions.Namespace,
		RootPath:  repoRoot,
	}
	if version != "" {
		info.ScenarioOptions["version"] = version
	}
	if config != nil {
		configPath := filepath.Join(t.TempDir(), "project-config.json")
		require.NoError(t, os.WriteFile(configPath, config, 0644))
		info.ScenarioOptions["project-config-file"] = configPath
	}
	return info
}

func startProjectWorker(
	t *testing.T,
	ctx context.Context,
	info loadgen.ScenarioInfo,
	opts projectScenarioOptions,
	prog sdkbuild.Program,
) <-chan error {
	t.Helper()
	require.NotEmpty(t, opts.projectName)

	builder := workers.Builder{
		ProjectName: opts.projectName,
		SdkOptions:  opts.sdkOpts,
		Logger:      info.Logger.Named(fmt.Sprintf("%s-worker-builder", opts.sdkOpts.Language)),
	}

	// If we have a prebuilt program, use it
	if prog != nil {
		require.NotNil(t, prog)
		builder.DirName = filepath.Base(prog.Dir())
	}

	runner := &workers.Runner{
		Builder:                  builder,
		TaskQueueName:            loadgen.TaskQueueForRun(info.RunID),
		GracefulShutdownDuration: 5 * time.Second,
		ScenarioID: clioptions.ScenarioID{
			Scenario: "project",
			RunID:    info.RunID,
		},
		LoggingOptions: clioptions.LoggingOptions{
			PreparedLogger: info.Logger.Named(fmt.Sprintf("%s-worker", opts.sdkOpts.Language)),
		},
	}
	require.NoError(t, runner.ClientOptions.FlagSet().Set("server-address", info.ClientOptions.Address))
	require.NoError(t, runner.ClientOptions.FlagSet().Set("namespace", info.ClientOptions.Namespace))

	workerErrCh := make(chan error, 1)
	go func() {
		defer close(workerErrCh)
		workerErrCh <- runner.Run(ctx, workers.BaseDir(info.RootPath, opts.sdkOpts.Language))
	}()
	return workerErrCh
}
