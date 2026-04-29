package harness

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/workers/go/harness/api"
	sdkclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProjectRunMetadata struct {
	RunID       string
	ExecutionID string
}

type ProjectInitContext struct {
	Logger     *zap.SugaredLogger
	Run        ProjectRunMetadata
	TaskQueue  string
	ConfigJSON []byte
}

type ProjectExecuteContext struct {
	Logger    *zap.SugaredLogger
	Run       ProjectRunMetadata
	TaskQueue string
	Iteration int64
	Payload   []byte
}

type ProjectExecuteHandler func(sdkclient.Client, ProjectExecuteContext) error
type ProjectInitHandler func(sdkclient.Client, ProjectInitContext) error

type ProjectHandlers struct {
	Execute ProjectExecuteHandler
	Init    ProjectInitHandler
}

type projectServiceServer struct {
	api.UnimplementedProjectServiceServer

	handlers      ProjectHandlers
	clientFactory ClientFactory
	logger        *zap.SugaredLogger

	client sdkclient.Client
	run    *ProjectRunMetadata
}

func newProjectServiceServer(handlers ProjectHandlers, clientFactory ClientFactory, logger *zap.SugaredLogger) *projectServiceServer {
	if logger == nil {
		nop := zap.NewNop()
		logger = nop.Sugar()
	}
	return &projectServiceServer{
		handlers:      handlers,
		clientFactory: clientFactory,
		logger:        logger,
	}
}

func (s *projectServiceServer) Init(ctx context.Context, request *api.InitRequest) (*api.InitResponse, error) {
	if request.GetTaskQueue() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_queue required")
	}
	if request.GetExecutionId() == "" {
		return nil, status.Error(codes.InvalidArgument, "execution_id required")
	}
	if request.GetRunId() == "" {
		return nil, status.Error(codes.InvalidArgument, "run_id required")
	}
	connectOptions := request.GetConnectOptions()
	if connectOptions == nil || connectOptions.GetServerAddress() == "" {
		return nil, status.Error(codes.InvalidArgument, "server_address required")
	}
	if connectOptions.GetNamespace() == "" {
		return nil, status.Error(codes.InvalidArgument, "namespace required")
	}

	config, err := buildClientConfig(clientConfigOptions{
		Logger:                  s.logger,
		ServerAddress:           connectOptions.GetServerAddress(),
		Namespace:               connectOptions.GetNamespace(),
		AuthHeader:              connectOptions.GetAuthHeader(),
		EnableTLS:               connectOptions.GetEnableTls(),
		TLSCertPath:             connectOptions.GetTlsCertPath(),
		TLSKeyPath:              connectOptions.GetTlsKeyPath(),
		TLSServerName:           connectOptions.GetTlsServerName(),
		DisableHostVerification: connectOptions.GetDisableHostVerification(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	client, err := s.clientFactory(config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create client: %v", err)
	}

	run := ProjectRunMetadata{
		RunID:       request.GetRunId(),
		ExecutionID: request.GetExecutionId(),
	}
	if s.handlers.Init != nil {
		if err := s.handlers.Init(client, ProjectInitContext{
			Logger:     s.logger,
			Run:        run,
			TaskQueue:  request.GetTaskQueue(),
			ConfigJSON: request.GetConfigJson(),
		}); err != nil {
			if client != nil {
				client.Close()
			}
			return nil, status.Errorf(codes.Internal, "init handler failed: %v", err)
		}
	}

	s.client = client
	s.run = &run
	return &api.InitResponse{}, nil
}

func (s *projectServiceServer) Execute(ctx context.Context, request *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if request.GetTaskQueue() == "" {
		return nil, status.Error(codes.InvalidArgument, "task_queue required")
	}
	if s.client == nil || s.run == nil {
		return nil, status.Error(codes.FailedPrecondition, "Init must be called before Execute")
	}
	if err := s.handlers.Execute(s.client, ProjectExecuteContext{
		Logger:    s.logger,
		Run:       *s.run,
		TaskQueue: request.GetTaskQueue(),
		Iteration: request.GetIteration(),
		Payload:   request.GetPayload(),
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "execute handler failed: %v", err)
	}
	return &api.ExecuteResponse{}, nil
}

func runProjectServerCLI(handlers ProjectHandlers, clientFactory ClientFactory, argv []string) error {
	if handlers.Execute == nil {
		return fmt.Errorf("project handlers must provide an Execute handler")
	}
	if clientFactory == nil {
		return fmt.Errorf("client factory is required")
	}
	flagSet := pflag.NewFlagSet("project-server", pflag.ContinueOnError)
	port := flagSet.Int("port", 8080, "gRPC listen port")
	if err := flagSet.Parse(argv); err != nil {
		return err
	}
	logger, err := configureLogger("info", "console")
	if err != nil {
		return err
	}
	defer logger.Sync()

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", *port))
	if err != nil {
		return err
	}
	server := grpc.NewServer()
	api.RegisterProjectServiceServer(server, newProjectServiceServer(handlers, clientFactory, logger))
	logger.Infof("Project server listening on port %d", *port)

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- server.Serve(listener)
	}()

	select {
	case err := <-serveErrCh:
		return err
	case <-runContext.Done():
		logger.Info("Shutdown signal received, stopping project server")
		server.GracefulStop()
		return <-serveErrCh
	}
}
