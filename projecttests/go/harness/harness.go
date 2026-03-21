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
	"sync"

	"github.com/temporalio/omes/projecttests/go/harness/api"
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

// ExecuteFunc is called once per Execute RPC (one per iteration).
type ExecuteFunc func(context.Context, *Config) error

// InitFunc is called once during the Init RPC.
type InitFunc func(*InitConfig) error

// WorkerFunc is the function type for running the worker.
type WorkerFunc func(*WorkerConfig) error

// Config is the per-iteration data passed to ExecuteFunc.
type Config struct {
	ConnectionOptions client.Options
	TaskQueue         string
	RunID             string
	ExecutionID       string
	Iteration         int64
	Payload           []byte // per-iteration data from executor
}

// InitConfig is the data passed to InitFunc during the Init RPC.
type InitConfig struct {
	ConnectionOptions client.Options
	TaskQueue         string
	RunID             string
	ExecutionID       string
	ProjectConfig     []byte // from --config file
}

// WorkerConfig is passed to given worker function.
type WorkerConfig struct {
	// ConnectionOptions are starter-provided defaults for client.Dial.
	ConnectionOptions client.Options
	// TaskQueue for the worker
	TaskQueue string
	// PromListenAddress is the optional Prometheus metrics endpoint address
	PromListenAddress string
}

// Harness is the core type that bridges the omes CLI with project test code.
// It dispatches to either a gRPC server (for Init/Execute RPCs) or a worker process.
type Harness struct {
	mu          sync.RWMutex
	runCtx      *runContext
	workerFunc  WorkerFunc
	initFunc    InitFunc
	executeFunc ExecuteFunc
	api.UnsafeProjectServiceServer
}

func New() *Harness {
	return &Harness{}
}

func (h *Harness) RegisterWorker(workerFunc WorkerFunc) {
	if h.workerFunc != nil {
		log.Fatalf("worker already registered")
	}
	h.workerFunc = workerFunc
}

func (h *Harness) OnInit(fn InitFunc) {
	if h.initFunc != nil {
		log.Fatalf("init handler already registered")
	}
	h.initFunc = fn
}

func (h *Harness) OnExecute(fn ExecuteFunc) {
	if h.executeFunc != nil {
		log.Fatalf("execute handler already registered")
	}
	h.executeFunc = fn
}

func (h *Harness) Run() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: <program> [project-server|worker] ...")
		os.Exit(1)
	}
	cmd := os.Args[1]
	os.Args = append(os.Args[:1], os.Args[2:]...) // Remove subcommand for flag parsing

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

	if req.RegisterSearchAttributes {
		err := s.registerSearchAttributes(ctx, opts)
		if err != nil {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}

	// Hold the write lock through initFunc so that Execute RPCs block
	// until initialization is fully complete.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initFunc != nil {
		initConfig := &InitConfig{
			ConnectionOptions: opts,
			TaskQueue:         req.GetTaskQueue(),
			RunID:             req.GetRunId(),
			ExecutionID:       req.GetExecutionId(),
			ProjectConfig:     req.GetConfig(),
		}
		if err := s.initFunc(initConfig); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	s.runCtx = &runContext{
		ConnectionOptions: opts,
		TaskQueue:         req.GetTaskQueue(),
		ExecutionID:       req.GetExecutionId(),
		RunID:             req.GetRunId(),
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

	runCtx, initialized := s.getRunContext()
	if !initialized {
		return nil, status.Error(codes.FailedPrecondition, "must initialize harness before calling execute")
	}

	config := &Config{
		ConnectionOptions: runCtx.ConnectionOptions,
		TaskQueue:         runCtx.TaskQueue,
		RunID:             runCtx.RunID,
		ExecutionID:       runCtx.ExecutionID,
		Iteration:         req.Iteration,
		Payload:           req.Payload,
	}

	if err := s.executeFunc(ctx, config); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &api.ExecuteResponse{}, nil
}

// --- Internal helpers ---

type runContext struct {
	ConnectionOptions client.Options
	TaskQueue         string
	ExecutionID       string
	RunID             string
}

func (s *Harness) getRunContext() (runContext, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.runCtx == nil {
		return runContext{}, false
	}
	return *s.runCtx, true
}

func (h *Harness) startGrpcServer() {
	if h.executeFunc == nil {
		log.Fatalf("Attempted to start server, but no execute handler was registered")
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

func (s *Harness) registerSearchAttributes(ctx context.Context, opts client.Options) error {
	clientConn, err := client.Dial(opts)
	if err != nil {
		return fmt.Errorf("failed to connect to Temporal for search attribute registration: %w", err)
	}
	defer clientConn.Close()

	_, err = clientConn.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: map[string]enums.IndexedValueType{
			"KS_Keyword":           enums.INDEXED_VALUE_TYPE_KEYWORD,
			"KS_Int":               enums.INDEXED_VALUE_TYPE_INT,
			OmesSearchAttributeKey: enums.INDEXED_VALUE_TYPE_KEYWORD,
		},
		Namespace: opts.Namespace,
	})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return nil
		}
		// Some Temporal server versions return this as a non-AlreadyExists error.
		if strings.Contains(err.Error(), "attributes mapping unavailable") {
			return nil
		}
		return fmt.Errorf("failed to register search attributes: %w", err)
	}
	return nil
}
