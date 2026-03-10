package harness

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/temporalio/omes/projecttests/go/harness/api"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const OmesSearchAttributeKey = "OmesExecutionID"

type Harness struct {
	mu          sync.RWMutex
	runCtx      *runContext
	workerFunc  WorkerFunc
	initFunc    InitFunc
	executeFunc ExecuteFunc
	api.UnsafeProjectServiceServer
}

type runContext struct {
	ConnectionOptions client.Options
	TaskQueue         string
	ExecutionID       string
	RunID             string
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

func (s *Harness) Init(ctx context.Context, req *api.InitRequest) (*api.InitResponse, error) {
	if err := validateInit(req); err != nil {
		return nil, err
	}

	co := req.GetConnectOptions()
	opts, err := buildClientOptions(
		co.GetServerAddress(),
		co.GetNamespace(),
		co.GetAuthHeader(),
		TLSOptions{
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

func (s *Harness) getRunContext() (runContext, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.runCtx == nil {
		return runContext{}, false
	}
	return *s.runCtx, true
}

func (s *Harness) registerSearchAttributes(ctx context.Context, opts client.Options) error {
	clientConn, err := client.Dial(opts)
	if err != nil {
		return fmt.Errorf("failed to connect to Temporal for search attribute registration: %w", err)
	}
	defer clientConn.Close()

	_, err = clientConn.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: map[string]enums.IndexedValueType{
			"KS_Keyword":              enums.INDEXED_VALUE_TYPE_KEYWORD,
			"KS_Int":                  enums.INDEXED_VALUE_TYPE_INT,
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
