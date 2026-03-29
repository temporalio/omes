package harness

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/temporalio/omes/workers/go/projects/api"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	OmesSearchAttributeKey = "OmesExecutionID"
	defaultHost            = "0.0.0.0"
)

// ClientFunc creates a Temporal client from the given options. Called by both
// the worker subprocess and the project-server Init handler.
type ClientFunc func(client.Options, ClientConfig) (client.Client, error)

// ClientConfig is passed to ClientFunc with minimal connection-time data.
type ClientConfig struct {
	TaskQueue string
}

// InitFunc performs project-specific setup after the client is created.
// Only called by the project-server Init gRPC handler, not by the worker.
type InitFunc func(context.Context, client.Client, InitConfig) error

// InitConfig is passed to InitFunc with run-level configuration.
type InitConfig struct {
	ConfigJSON  []byte // from --config-file, project-specific schema
	TaskQueue   string
	RunID       string
	ExecutionID string
}

// ExecuteFunc is called once per Execute RPC (one per iteration).
type ExecuteFunc func(context.Context, client.Client, ExecuteInfo) error

// ExecuteInfo is passed to ExecuteFunc for each iteration.
type ExecuteInfo struct {
	TaskQueue   string
	ExecutionID string
	RunID       string
	Iteration   int64
	Payload     []byte // from executor, executor-specific schema
}

// WorkerFunc runs the worker.
type WorkerFunc func(client.Client, WorkerConfig) error

// WorkerConfig is passed to WorkerFunc.
type WorkerConfig struct {
	TaskQueue         string
	PromListenAddress string
}

// Harness bridges the omes CLI with project test code.
type Harness struct {
	runCtx      *runContext
	client      client.Client
	clientFunc  ClientFunc
	initFunc    InitFunc
	workerFunc  WorkerFunc
	executeFunc ExecuteFunc
	api.UnsafeProjectServiceServer
}

func New() *Harness {
	return &Harness{}
}

func (h *Harness) RegisterClient(fn ClientFunc) {
	if h.clientFunc != nil {
		log.Fatalf("client factory already registered")
	}
	h.clientFunc = fn
}

func (h *Harness) OnInit(fn InitFunc) {
	if h.initFunc != nil {
		log.Fatalf("init handler already registered")
	}
	h.initFunc = fn
}

func (h *Harness) RegisterWorker(fn WorkerFunc) {
	if h.workerFunc != nil {
		log.Fatalf("worker already registered")
	}
	h.workerFunc = fn
}

func (h *Harness) OnExecute(fn ExecuteFunc) {
	if h.executeFunc != nil {
		log.Fatalf("execute handler already registered")
	}
	h.executeFunc = fn
}

func (h *Harness) Run() {
	if h.clientFunc == nil {
		log.Fatalf("no client factory registered; call RegisterClient before Run")
	}
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: <program> [project-server|worker] ...")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...)

	switch cmd {
	case "project-server":
		h.startGrpcServer()
	case "worker":
		h.startWorker()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

// Init handles the Init gRPC call from the omes CLI.
func (s *Harness) Init(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	if err := validateInit(req); err != nil {
		return nil, err
	}

	co := req.GetConnectOptions()
	opts, err := buildClientOptions(
		co.GetServerAddress(),
		co.GetNamespace(),
		co.GetAuthHeader(),
		tlsOptions{
			EnableTLS:               co.GetEnableTls(),
			CertPath:                co.GetTlsCertPath(),
			KeyPath:                 co.GetTlsKeyPath(),
			ServerName:              co.GetTlsServerName(),
			DisableHostVerification: co.GetDisableHostVerification(),
		},
	)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := s.clientFunc(opts, ClientConfig{
		TaskQueue: req.GetTaskQueue(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("client creation failed: %v", err))
	}
	s.client = c

	if s.initFunc != nil {
		if err := s.initFunc(ctx, c, InitConfig{
			ConfigJSON:  req.GetConfigJson(),
			TaskQueue:   req.GetTaskQueue(),
			RunID:       req.GetRunId(),
			ExecutionID: req.GetExecutionId(),
		}); err != nil {
			return nil, status.Error(codes.Internal, fmt.Sprintf("init failed: %v", err))
		}
	}

	if req.RegisterSearchAttributes {
		if err := s.registerSearchAttributes(ctx, co.GetNamespace()); err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}

	s.runCtx = &runContext{
		TaskQueue:   req.GetTaskQueue(),
		ExecutionID: req.GetExecutionId(),
		RunID:       req.GetRunId(),
	}

	return &api.InitResponse{}, nil
}

// Execute handles the Execute gRPC call from the omes CLI.
func (s *Harness) Execute(ctx context.Context, req *api.ExecuteRequest) (*api.ExecuteResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "execute request is nil")
	}
	if s.executeFunc == nil {
		return nil, status.Error(codes.FailedPrecondition, "no execute handler registered")
	}
	if s.client == nil {
		return nil, status.Error(codes.FailedPrecondition, "must initialize harness before calling execute")
	}

	if err := s.executeFunc(ctx, s.client, ExecuteInfo{
		TaskQueue:   s.runCtx.TaskQueue,
		ExecutionID: s.runCtx.ExecutionID,
		RunID:       s.runCtx.RunID,
		Iteration:   req.Iteration,
		Payload:     req.Payload,
	}); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.ExecuteResponse{}, nil
}

// --- Internal helpers ---

type runContext struct {
	TaskQueue   string
	ExecutionID string
	RunID       string
}

func (h *Harness) startGrpcServer() {
	if h.executeFunc == nil {
		log.Fatalf("Attempted to start server, but no execute handler was registered")
	}
	if h.workerFunc == nil {
		log.Fatalf("Attempted to start server, but no worker handler was registered")
	}

	fs := flag.NewFlagSet("client", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP port")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", defaultHost, *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	api.RegisterProjectServiceServer(grpcServer, h)
	if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

func validateInit(req *api.InitRequest) error {
	if req == nil {
		return status.Error(codes.InvalidArgument, "request: required")
	}
	if req.GetExecutionId() == "" {
		return status.Error(codes.InvalidArgument, "execution_id: required")
	}
	if req.GetRunId() == "" {
		return status.Error(codes.InvalidArgument, "run_id: required")
	}
	if req.GetTaskQueue() == "" {
		return status.Error(codes.InvalidArgument, "task_queue: required")
	}
	co := req.GetConnectOptions()
	if co == nil {
		return status.Error(codes.InvalidArgument, "connect_options: required")
	}
	if co.GetNamespace() == "" {
		return status.Error(codes.InvalidArgument, "connect_options.namespace: required")
	}
	if co.GetServerAddress() == "" {
		return status.Error(codes.InvalidArgument, "connect_options.server_address: required")
	}
	return nil
}

type staticHeadersProvider map[string]string

func (s staticHeadersProvider) GetHeaders(context.Context) (map[string]string, error) {
	return s, nil
}

type tlsOptions struct {
	EnableTLS               bool
	CertPath                string
	KeyPath                 string
	ServerName              string
	DisableHostVerification bool
}

func buildClientOptions(
	serverAddress string,
	namespace string,
	authHeader string,
	tlsOpts tlsOptions,
) (client.Options, error) {
	opts := client.Options{
		HostPort:  serverAddress,
		Namespace: namespace,
	}
	if authHeader != "" {
		opts.HeadersProvider = staticHeadersProvider{
			"Authorization": authHeader,
		}
	}
	tlsCfg, err := loadTLSConfig(tlsOpts)
	if err != nil {
		return client.Options{}, fmt.Errorf("failed to load TLS config: %w", err)
	}
	if tlsCfg != nil {
		opts.ConnectionOptions.TLS = tlsCfg
	}
	return opts, nil
}

func loadTLSConfig(opts tlsOptions) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.DisableHostVerification,
		ServerName:         opts.ServerName,
		MinVersion:         tls.VersionTLS13,
	}
	if opts.CertPath != "" {
		if opts.KeyPath == "" {
			return nil, errors.New("got TLS cert with no key")
		}
		cert, err := tls.LoadX509KeyPair(opts.CertPath, opts.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		return tlsConfig, nil
	} else if opts.KeyPath != "" {
		return nil, errors.New("got TLS key with no cert")
	}
	if opts.EnableTLS {
		return tlsConfig, nil
	}
	return nil, nil
}

func (s *Harness) registerSearchAttributes(ctx context.Context, namespace string) error {
	_, err := s.client.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: map[string]enums.IndexedValueType{
			"KS_Keyword":           enums.INDEXED_VALUE_TYPE_KEYWORD,
			"KS_Int":               enums.INDEXED_VALUE_TYPE_INT,
			OmesSearchAttributeKey: enums.INDEXED_VALUE_TYPE_KEYWORD,
		},
		Namespace: namespace,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
		if strings.Contains(err.Error(), "attributes mapping unavailable") {
			return nil
		}
		return fmt.Errorf("failed to register search attributes: %w", err)
	}
	return nil
}
