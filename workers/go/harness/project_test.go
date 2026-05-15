package harness

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os/exec"
	"strings"
	"testing"

	"github.com/temporalio/omes/workers/go/harness/api"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/testsuite"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type clientStub struct {
	sdkclient.Client
}

func (clientStub) Close() {}

const projectHarnessEchoWorkflowName = "ProjectHarnessEchoWorkflow"

func projectHarnessEchoWorkflow(_ workflow.Context, payload string) (string, error) {
	return payload, nil
}

func TestProjectServerExecutesWorkflowAgainstRealTemporalServer(t *testing.T) {
	ctx := context.Background()
	taskQueue := "project-harness-e2e"
	type initEvent struct {
		client  sdkclient.Client
		context ProjectInitContext
	}
	type executeEvent struct {
		client  sdkclient.Client
		context ProjectExecuteContext
		result  string
	}
	var initSeen initEvent
	var executeSeen executeEvent

	startDevServer := func() *testsuite.DevServer {
		temporalPath, err := exec.LookPath("temporal")
		if err != nil {
			t.Skipf("temporal CLI not available: %v", err)
		}
		devServer, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
			ExistingPath:  temporalPath,
			ClientOptions: &sdkclient.Options{Namespace: "default"},
			LogLevel:      "error",
			Stdout:        io.Discard,
			Stderr:        io.Discard,
		})
		if err != nil {
			t.Fatalf("failed to start dev server: %v", err)
		}
		t.Cleanup(func() { _ = devServer.Stop() })
		return devServer
	}
	startEchoWorker := func(client sdkclient.Client) {
		worker := sdkworker.New(client, taskQueue, sdkworker.Options{})
		worker.RegisterWorkflowWithOptions(projectHarnessEchoWorkflow, workflow.RegisterOptions{Name: projectHarnessEchoWorkflowName})
		if err := worker.Start(); err != nil {
			t.Fatalf("failed to start worker: %v", err)
		}
		t.Cleanup(worker.Stop)
	}
	initProject := func(projectClient api.ProjectServiceClient, serverAddress string) {
		_, err := projectClient.Init(ctx, &api.InitRequest{
			ExecutionId: "exec-id",
			RunId:       "run-id",
			TaskQueue:   taskQueue,
			ConnectOptions: &api.ConnectOptions{
				Namespace:     "default",
				ServerAddress: serverAddress,
			},
			ConfigJson: []byte("{\"hello\":\"world\"}"),
		})
		if err != nil {
			t.Fatalf("Init returned error: %v", err)
		}
	}
	executeProject := func(projectClient api.ProjectServiceClient) {
		_, err := projectClient.Execute(ctx, &api.ExecuteRequest{
			Iteration: 7,
			TaskQueue: taskQueue,
			Payload:   []byte("payload"),
		})
		if err != nil {
			t.Fatalf("Execute returned error: %v", err)
		}
	}
	assertExpectedEvents := func() {
		if initSeen.client == nil || executeSeen.client == nil {
			t.Fatalf("expected both handlers to receive a client")
		}
		if initSeen.client != executeSeen.client {
			t.Fatalf("expected execute handler to reuse cached client")
		}
		if initSeen.context.Run.RunID != "run-id" ||
			initSeen.context.Run.ExecutionID != "exec-id" ||
			initSeen.context.TaskQueue != taskQueue ||
			string(initSeen.context.ConfigJSON) != "{\"hello\":\"world\"}" {
			t.Fatalf("unexpected init context: %+v", initSeen.context)
		}
		if executeSeen.context.Run.RunID != "run-id" ||
			executeSeen.context.Run.ExecutionID != "exec-id" ||
			executeSeen.context.TaskQueue != taskQueue ||
			executeSeen.context.Iteration != 7 ||
			string(executeSeen.context.Payload) != "payload" {
			t.Fatalf("unexpected execute context: %+v", executeSeen.context)
		}
		if executeSeen.result != "payload" {
			t.Fatalf("unexpected workflow result: %q", executeSeen.result)
		}
	}

	devServer := startDevServer()
	startEchoWorker(devServer.Client())

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen for grpc server: %v", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	api.RegisterProjectServiceServer(server, newProjectServiceServer(ProjectHandlers{
		Init: func(client sdkclient.Client, context ProjectInitContext) error {
			initSeen = initEvent{client: client, context: context}
			return nil
		},
		Execute: func(client sdkclient.Client, context ProjectExecuteContext) error {
			run, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
				ID:        fmt.Sprintf("%s-%d", context.Run.ExecutionID, context.Iteration),
				TaskQueue: context.TaskQueue,
			}, projectHarnessEchoWorkflowName, string(context.Payload))
			if err != nil {
				return err
			}
			var result string
			if err := run.Get(ctx, &result); err != nil {
				return err
			}
			executeSeen = executeEvent{client: client, context: context, result: result}
			return nil
		},
	}, DefaultClientFactory, zap.NewNop().Sugar()))
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to dial grpc server: %v", err)
	}
	defer conn.Close()

	projectClient := api.NewProjectServiceClient(conn)
	initProject(projectClient, devServer.FrontendHostPort())
	executeProject(projectClient)
	assertExpectedEvents()
}

func TestProjectInitRejectsInvalidTLSConfiguration(t *testing.T) {
	server := newProjectServiceServer(ProjectHandlers{
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, nil, nil)

	_, err := server.Init(context.Background(), &api.InitRequest{
		ExecutionId: "exec-id",
		RunId:       "run-id",
		TaskQueue:   "task-queue",
		ConnectOptions: &api.ConnectOptions{
			Namespace:     "default",
			ServerAddress: "localhost:7233",
			EnableTls:     true,
			TlsCertPath:   "/tmp/cert.pem",
		},
	})
	if status.Code(err) != codes.InvalidArgument || status.Convert(err).Message() != "Client cert specified, but not client key!" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectInitPassesRunMetadataToHandler(t *testing.T) {
	sharedClient := clientStub{}
	var capturedConfig ClientConfig
	var initClient sdkclient.Client
	var initContext ProjectInitContext
	server := newProjectServiceServer(ProjectHandlers{
		Init: func(client sdkclient.Client, context ProjectInitContext) error {
			initClient = client
			initContext = context
			return nil
		},
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, func(config ClientConfig) (sdkclient.Client, error) {
		capturedConfig = config
		return sharedClient, nil
	}, nil)

	request := makeInitRequest()
	request.ConnectOptions.AuthHeader = "Bearer token"
	request.ConnectOptions.EnableTls = true
	request.ConnectOptions.TlsServerName = "server.local"
	request.ConnectOptions.DisableHostVerification = true

	_, err := server.Init(context.Background(), request)
	if err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if initClient != sharedClient {
		t.Fatalf("expected init handler to receive shared client")
	}
	if capturedConfig.TargetHost != "localhost:7233" ||
		capturedConfig.Namespace != "default" ||
		capturedConfig.APIKey != "token" ||
		capturedConfig.TLS == nil ||
		capturedConfig.TLS.ServerName != "server.local" ||
		len(capturedConfig.TLS.Certificates) != 0 {
		t.Fatalf("unexpected client config: %+v", capturedConfig)
	}
	if initContext.Run.RunID != "run-id" ||
		initContext.Run.ExecutionID != "exec-id" ||
		initContext.TaskQueue != "task-queue" ||
		string(initContext.ConfigJSON) != "{\"hello\":\"world\"}" {
		t.Fatalf("unexpected init context: %+v", initContext)
	}
}

func TestProjectExecuteRequiresInit(t *testing.T) {
	server := newProjectServiceServer(ProjectHandlers{
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, nil, nil)

	_, err := server.Execute(context.Background(), makeExecuteRequest())
	if status.Code(err) != codes.FailedPrecondition || status.Convert(err).Message() != "Init must be called before Execute" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectExecutePassesIterationPayloadAndRunMetadata(t *testing.T) {
	sharedClient := clientStub{}
	var executeClient sdkclient.Client
	var executeContext ProjectExecuteContext
	server := newProjectServiceServer(ProjectHandlers{
		Execute: func(client sdkclient.Client, context ProjectExecuteContext) error {
			executeClient = client
			executeContext = context
			return nil
		},
	}, func(ClientConfig) (sdkclient.Client, error) { return sharedClient, nil }, nil)

	if _, err := server.Init(context.Background(), makeInitRequest()); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	if _, err := server.Execute(context.Background(), makeExecuteRequest()); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if executeClient != sharedClient {
		t.Fatalf("expected execute handler to receive cached client")
	}
	if executeContext.Iteration != 7 ||
		string(executeContext.Payload) != "payload" ||
		executeContext.TaskQueue != "task-queue" ||
		executeContext.Run.RunID != "run-id" ||
		executeContext.Run.ExecutionID != "exec-id" {
		t.Fatalf("unexpected execute context: %+v", executeContext)
	}
}

func TestProjectClientFactoryFailureMapsToInternalError(t *testing.T) {
	server := newProjectServiceServer(ProjectHandlers{
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, func(ClientConfig) (sdkclient.Client, error) {
		return nil, errors.New("boom")
	}, nil)

	_, err := server.Init(context.Background(), makeInitRequest())
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "failed to create client: boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectInitHandlerFailureLeavesServerUninitialized(t *testing.T) {
	server := newProjectServiceServer(ProjectHandlers{
		Init: func(sdkclient.Client, ProjectInitContext) error {
			return errors.New("bad init")
		},
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, func(ClientConfig) (sdkclient.Client, error) { return clientStub{}, nil }, nil)

	_, err := server.Init(context.Background(), makeInitRequest())
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "init handler failed: bad init" {
		t.Fatalf("unexpected init error: %v", err)
	}

	_, err = server.Execute(context.Background(), makeExecuteRequest())
	if status.Code(err) != codes.FailedPrecondition || status.Convert(err).Message() != "Init must be called before Execute" {
		t.Fatalf("unexpected execute error: %v", err)
	}
}

func TestProjectExecuteHandlerFailureMapsToInternalError(t *testing.T) {
	server := newProjectServiceServer(ProjectHandlers{
		Execute: func(sdkclient.Client, ProjectExecuteContext) error {
			return errors.New("bad execute")
		},
	}, func(ClientConfig) (sdkclient.Client, error) { return clientStub{}, nil }, nil)

	if _, err := server.Init(context.Background(), makeInitRequest()); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}
	_, err := server.Execute(context.Background(), makeExecuteRequest())
	if status.Code(err) != codes.Internal || status.Convert(err).Message() != "execute handler failed: bad execute" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunProjectServerCLIRequiresExecuteHandler(t *testing.T) {
	err := runProjectServerCLI(ProjectHandlers{}, nil, []string{"--port", "0"})
	if err == nil {
		t.Fatal("expected error when Execute handler is missing, got nil")
	}
	if !strings.Contains(err.Error(), "Execute handler") {
		t.Fatalf("expected error mentioning Execute handler, got: %v", err)
	}
}

func TestRunProjectServerCLIRequiresClientFactory(t *testing.T) {
	err := runProjectServerCLI(ProjectHandlers{
		Execute: func(sdkclient.Client, ProjectExecuteContext) error { return nil },
	}, nil, []string{"--port", "0"})
	if err == nil {
		t.Fatal("expected error when client factory is missing, got nil")
	}
	if !strings.Contains(err.Error(), "client factory is required") {
		t.Fatalf("expected error mentioning client factory, got: %v", err)
	}
}

func makeInitRequest() *api.InitRequest {
	return &api.InitRequest{
		ExecutionId: "exec-id",
		RunId:       "run-id",
		TaskQueue:   "task-queue",
		ConnectOptions: &api.ConnectOptions{
			Namespace:     "default",
			ServerAddress: "localhost:7233",
		},
		ConfigJson: []byte("{\"hello\":\"world\"}"),
	}
}

func makeExecuteRequest() *api.ExecuteRequest {
	return &api.ExecuteRequest{
		Iteration: 7,
		TaskQueue: "task-queue",
		Payload:   []byte("payload"),
	}
}
