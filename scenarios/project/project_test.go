package project

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
	api "github.com/temporalio/omes/workers/go/harness/api"
	sdkclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func TestValidateRequiresLanguage(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"project-name": "helloworld",
		},
	})
	require.EqualError(t, err, "--option language=<lang> is required")
}

func TestValidateAcceptsSupportedLanguages(t *testing.T) {
	for _, lang := range []string{"go", "java", "python", "ruby", "typescript", "dotnet"} {
		t.Run(lang, func(t *testing.T) {
			opts, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
				ScenarioOptions: map[string]string{
					"language":     lang,
					"project-name": "helloworld",
				},
			})
			require.NoError(t, err)
			require.Equal(t, lang, opts.sdkOpts.Language.String())
			require.Equal(t, "helloworld", opts.projectName)
		})
	}
}

func TestValidateRequiresProjectNameWithPrebuilt(t *testing.T) {
	_, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"language":             "python",
			"prebuilt-project-dir": "workers/python/project-build-runner-helloworld",
		},
	})
	require.EqualError(t, err, "--option project-name=<name> is required")
}

func TestValidateReadsProjectConfigAndTimeout(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "project-config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"hello":"world"}`), 0644))

	opts, err := (&projectScenarioExecutor{}).validate(loadgen.ScenarioInfo{
		ScenarioOptions: map[string]string{
			"language":                     "go",
			"project-name":                 "helloworld",
			"project-config-file":          configPath,
			"project-server-ready-timeout": "250ms",
			"version":                      "v1.2.3",
		},
	})
	require.NoError(t, err)
	require.Equal(t, []byte(`{"hello":"world"}`), opts.configJSON)
	require.Equal(t, 250*time.Millisecond, opts.projectServerReadyTimeout)
	require.Equal(t, "v1.2.3", opts.sdkOpts.Version)
}

func TestProjectProcessArgs(t *testing.T) {
	tests := []struct {
		lang clioptions.Language
		want []string
	}{
		{
			lang: clioptions.LangPython,
			want: []string{"apps.registry", "--app", "helloworld", "project-server", "--port", "4321"},
		},
		{
			lang: clioptions.LangTypeScript,
			want: []string{"./tslib/apps/registry.js", "--app", "helloworld", "project-server", "--port", "4321"},
		},
		{
			lang: clioptions.LangGo,
			want: []string{"--app", "helloworld", "project-server", "--port", "4321"},
		},
		{
			lang: clioptions.LangJava,
			want: []string{"--app", "helloworld", "project-server", "--port", "4321"},
		},
		{
			lang: clioptions.LangDotNet,
			want: []string{"--app", "helloworld", "project-server", "--port", "4321"},
		},
		{
			lang: clioptions.LangRuby,
			want: []string{"--app", "helloworld", "project-server", "--port", "4321"},
		},
	}

	for _, test := range tests {
		t.Run(test.lang.String(), func(t *testing.T) {
			require.Equal(t, test.want, projectProcessArgs(test.lang, "helloworld", 4321))
		})
	}
}

func TestProjectScenarioRunUsesPrebuiltProjectWhenProvided(t *testing.T) {
	restore := replaceProjectHooks(t)
	handle := &recordingProjectHandle{}
	restore.newHandle = func(context.Context, int, *api.InitRequest, time.Duration) (projectHandleClient, error) {
		return handle, nil
	}

	var buildCalled bool
	var loadCalled bool
	restore.build = func(context.Context, string, projectScenarioOptions, *zap.SugaredLogger) (sdkbuild.Program, error) {
		buildCalled = true
		return fakeProgram{}, nil
	}
	restore.load = func(dir, repoRoot string, lang clioptions.Language) (sdkbuild.Program, error) {
		loadCalled = true
		require.Equal(t, "/tmp/prebuilt", dir)
		require.Equal(t, "/repo", repoRoot)
		require.Equal(t, clioptions.LangRuby, lang)
		return fakeProgram{}, nil
	}
	restore.start = func(context.Context, sdkbuild.Program, *zap.SugaredLogger, clioptions.Language, string, int) (*exec.Cmd, error) {
		return nil, nil
	}

	info := testScenarioInfo("ruby")
	info.RootPath = "/repo"
	info.ScenarioOptions["prebuilt-project-dir"] = "/tmp/prebuilt"
	require.NoError(t, (&projectScenarioExecutor{}).Run(t.Context(), info))
	require.False(t, buildCalled)
	require.True(t, loadCalled)
	require.Equal(t, []int64{1}, handle.iterations)
	require.True(t, handle.closed)
}

func TestProjectScenarioRunDrivesProjectServerProcess(t *testing.T) {
	eventsPath := filepath.Join(t.TempDir(), "events.jsonl")
	restore := replaceProjectHooks(t)
	restore.build = func(context.Context, string, projectScenarioOptions, *zap.SugaredLogger) (sdkbuild.Program, error) {
		return helperProgram{eventsPath: eventsPath}, nil
	}

	configPath := filepath.Join(t.TempDir(), "project-config.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"project":"helloworld"}`), 0644))

	info := testScenarioInfo("typescript")
	info.Configuration.Iterations = 2
	info.ScenarioOptions["project-config-file"] = configPath
	require.NoError(t, (&projectScenarioExecutor{}).Run(t.Context(), info))

	events := readHelperEvents(t, eventsPath)
	require.Len(t, events, 3)
	require.Equal(t, "init", events[0].Kind)
	require.Equal(t, info.ExecutionID, events[0].ExecutionID)
	require.Equal(t, info.RunID, events[0].RunID)
	require.Equal(t, loadgen.TaskQueueForRun(info.RunID), events[0].TaskQueue)
	require.Equal(t, "127.0.0.1:7233", events[0].ServerAddress)
	require.Equal(t, "default", events[0].Namespace)
	require.Equal(t, `{"project":"helloworld"}`, events[0].ConfigJSON)

	require.Equal(t, "execute", events[1].Kind)
	require.Equal(t, int64(1), events[1].Iteration)
	require.Equal(t, loadgen.TaskQueueForRun(info.RunID), events[1].TaskQueue)
	require.Equal(t, "execute", events[2].Kind)
	require.Equal(t, int64(2), events[2].Iteration)
}

func TestProjectScenarioHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_PROJECT_HELPER") != "1" {
		return
	}
	if err := runProjectScenarioHelperProcess(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

type projectHooks struct {
	build     func(context.Context, string, projectScenarioOptions, *zap.SugaredLogger) (sdkbuild.Program, error)
	load      func(string, string, clioptions.Language) (sdkbuild.Program, error)
	start     func(context.Context, sdkbuild.Program, *zap.SugaredLogger, clioptions.Language, string, int) (*exec.Cmd, error)
	newHandle func(context.Context, int, *api.InitRequest, time.Duration) (projectHandleClient, error)
}

func replaceProjectHooks(t *testing.T) *projectHooks {
	t.Helper()
	originalBuild := buildProjectProgram
	originalLoad := loadPrebuiltProgram
	originalStart := startProjectProgram
	originalNewHandle := newProjectHandleFn

	hooks := &projectHooks{}
	hooks.build = originalBuild
	hooks.load = originalLoad
	hooks.start = originalStart
	hooks.newHandle = originalNewHandle
	buildProjectProgram = func(ctx context.Context, repoRoot string, opts projectScenarioOptions, logger *zap.SugaredLogger) (sdkbuild.Program, error) {
		return hooks.build(ctx, repoRoot, opts, logger)
	}
	loadPrebuiltProgram = func(dir, repoRoot string, lang clioptions.Language) (sdkbuild.Program, error) {
		return hooks.load(dir, repoRoot, lang)
	}
	startProjectProgram = func(ctx context.Context, prog sdkbuild.Program, logger *zap.SugaredLogger, lang clioptions.Language, appName string, port int) (*exec.Cmd, error) {
		return hooks.start(ctx, prog, logger, lang, appName, port)
	}
	newProjectHandleFn = func(ctx context.Context, port int, req *api.InitRequest, readyTimeout time.Duration) (projectHandleClient, error) {
		return hooks.newHandle(ctx, port, req, readyTimeout)
	}

	t.Cleanup(func() {
		buildProjectProgram = originalBuild
		loadPrebuiltProgram = originalLoad
		startProjectProgram = originalStart
		newProjectHandleFn = originalNewHandle
	})
	return hooks
}

func testScenarioInfo(lang string) loadgen.ScenarioInfo {
	return loadgen.ScenarioInfo{
		ScenarioName:   "project",
		RunID:          fmt.Sprintf("test-%s", lang),
		ExecutionID:    fmt.Sprintf("exec-%s", lang),
		MetricsHandler: sdkclient.MetricsNopHandler,
		Logger:         zap.NewNop().Sugar(),
		ClientOptions: clioptions.ClientOptions{
			Address:   "127.0.0.1:7233",
			Namespace: "default",
		},
		Configuration: loadgen.RunConfiguration{
			Iterations:                    1,
			MaxConcurrent:                 1,
			DoNotRegisterSearchAttributes: true,
		},
		ScenarioOptions: map[string]string{
			"language":                     lang,
			"project-name":                 "helloworld",
			"project-server-ready-timeout": "5s",
		},
		Namespace: "default",
		RootPath:  "/repo",
	}
}

type recordingProjectHandle struct {
	iterations []int64
	closed     bool
}

func (h *recordingProjectHandle) close() error {
	h.closed = true
	return nil
}

func (h *recordingProjectHandle) execute(_ context.Context, req *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	h.iterations = append(h.iterations, req.GetIteration())
	return &api.ExecuteResponse{}, nil
}

type fakeProgram struct{}

func (fakeProgram) Dir() string {
	return "/tmp/fake-project"
}

func (fakeProgram) NewCommand(context.Context, ...string) (*exec.Cmd, error) {
	return nil, nil
}

type helperProgram struct {
	eventsPath string
}

func (p helperProgram) Dir() string {
	return "/tmp/helper-project"
}

func (p helperProgram) NewCommand(ctx context.Context, args ...string) (*exec.Cmd, error) {
	cmdArgs := append([]string{"-test.run=TestProjectScenarioHelperProcess", "--"}, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cmdArgs...)
	cmd.Env = append(os.Environ(),
		"GO_WANT_PROJECT_HELPER=1",
		"PROJECT_HELPER_EVENTS="+p.eventsPath,
	)
	return cmd, nil
}

type helperEvent struct {
	Kind          string `json:"kind"`
	ExecutionID   string `json:"execution_id,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	TaskQueue     string `json:"task_queue,omitempty"`
	ServerAddress string `json:"server_address,omitempty"`
	Namespace     string `json:"namespace,omitempty"`
	ConfigJSON    string `json:"config_json,omitempty"`
	Iteration     int64  `json:"iteration,omitempty"`
	Payload       string `json:"payload,omitempty"`
}

func readHelperEvents(t *testing.T, path string) []helperEvent {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()

	var events []helperEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event helperEvent
		require.NoError(t, json.Unmarshal(scanner.Bytes(), &event))
		events = append(events, event)
	}
	require.NoError(t, scanner.Err())
	return events
}

func runProjectScenarioHelperProcess() error {
	eventsPath := os.Getenv("PROJECT_HELPER_EVENTS")
	if eventsPath == "" {
		return fmt.Errorf("PROJECT_HELPER_EVENTS is required")
	}
	args := helperArgs(os.Args)
	if err := requireHelperArg(args, "--app", "helloworld"); err != nil {
		return err
	}
	if !containsArg(args, "project-server") {
		return fmt.Errorf("project-server command not found in %v", args)
	}
	port, err := helperPort(args)
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	api.RegisterProjectServiceServer(server, &helperProjectService{eventsPath: eventsPath})
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case err := <-serveErr:
		if err == grpc.ErrServerStopped {
			return nil
		}
		return err
	case <-signalCtx.Done():
		server.GracefulStop()
		err := <-serveErr
		if err == grpc.ErrServerStopped {
			return nil
		}
		return err
	}
}

type helperProjectService struct {
	api.UnimplementedProjectServiceServer
	eventsPath string
	mu         sync.Mutex
}

func (s *helperProjectService) Init(_ context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	connect := req.GetConnectOptions()
	if err := s.writeEvent(helperEvent{
		Kind:          "init",
		ExecutionID:   req.GetExecutionId(),
		RunID:         req.GetRunId(),
		TaskQueue:     req.GetTaskQueue(),
		ServerAddress: connect.GetServerAddress(),
		Namespace:     connect.GetNamespace(),
		ConfigJSON:    string(req.GetConfigJson()),
	}); err != nil {
		return nil, err
	}
	return &api.InitResponse{}, nil
}

func (s *helperProjectService) Execute(_ context.Context, req *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if err := s.writeEvent(helperEvent{
		Kind:      "execute",
		TaskQueue: req.GetTaskQueue(),
		Iteration: req.GetIteration(),
		Payload:   string(req.GetPayload()),
	}); err != nil {
		return nil, err
	}
	return &api.ExecuteResponse{}, nil
}

func (s *helperProjectService) writeEvent(event helperEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(s.eventsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(append(data, '\n'))
	return err
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}
	return nil
}

func helperPort(args []string) (int, error) {
	for i, arg := range args {
		if arg == "--port" && i+1 < len(args) {
			var port int
			_, err := fmt.Sscanf(args[i+1], "%d", &port)
			return port, err
		}
	}
	return 0, fmt.Errorf("--port not found in %v", args)
}

func requireHelperArg(args []string, name, value string) error {
	for i, arg := range args {
		if arg == name && i+1 < len(args) {
			if args[i+1] == value {
				return nil
			}
			return fmt.Errorf("%s value %q, want %q", name, args[i+1], value)
		}
	}
	return fmt.Errorf("%s not found in %v", name, args)
}

func containsArg(args []string, want string) bool {
	for _, arg := range args {
		if strings.EqualFold(arg, want) {
			return true
		}
	}
	return false
}
