package scenarios

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/internal/workertest"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type scheduleResponseLossProxy struct {
	workflowservice.UnimplementedWorkflowServiceServer
	backend workflowservice.WorkflowServiceClient
	mu      sync.Mutex
	calls   map[string]int
}

func scheduleProxyContext(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	return metadata.NewOutgoingContext(ctx, metadata.MD{"temporal-experiment": md.Get("temporal-experiment")})
}
func (p *scheduleResponseLossProxy) lose(op string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls[op]++
	return p.calls[op] == 1
}
func (p *scheduleResponseLossProxy) GetSystemInfo(ctx context.Context, r *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	v, err := p.backend.GetSystemInfo(scheduleProxyContext(ctx), r)
	return v, serviceerror.ToStatus(err).Err()
}
func (p *scheduleResponseLossProxy) CreateSchedule(ctx context.Context, r *workflowservice.CreateScheduleRequest) (*workflowservice.CreateScheduleResponse, error) {
	v, err := p.backend.CreateSchedule(scheduleProxyContext(ctx), r)
	if err != nil {
		return nil, serviceerror.ToStatus(err).Err()
	}
	if p.lose("create") {
		return nil, status.Error(codes.DeadlineExceeded, "response lost after real server create")
	}
	return v, nil
}
func (p *scheduleResponseLossProxy) DescribeSchedule(ctx context.Context, r *workflowservice.DescribeScheduleRequest) (*workflowservice.DescribeScheduleResponse, error) {
	v, err := p.backend.DescribeSchedule(scheduleProxyContext(ctx), r)
	return v, serviceerror.ToStatus(err).Err()
}
func (p *scheduleResponseLossProxy) UpdateSchedule(ctx context.Context, r *workflowservice.UpdateScheduleRequest) (*workflowservice.UpdateScheduleResponse, error) {
	v, err := p.backend.UpdateSchedule(scheduleProxyContext(ctx), r)
	if err != nil {
		return nil, serviceerror.ToStatus(err).Err()
	}
	if p.lose("update") {
		return nil, status.Error(codes.DeadlineExceeded, "response lost after real server update")
	}
	return v, nil
}
func (p *scheduleResponseLossProxy) DeleteSchedule(ctx context.Context, r *workflowservice.DeleteScheduleRequest) (*workflowservice.DeleteScheduleResponse, error) {
	v, err := p.backend.DeleteSchedule(scheduleProxyContext(ctx), r)
	if err != nil {
		return nil, serviceerror.ToStatus(err).Err()
	}
	if p.lose("delete") {
		return nil, status.Error(codes.DeadlineExceeded, "response lost after real server delete")
	}
	return v, nil
}

func TestSchedulerOperationRecoveryLocalServer(t *testing.T) {
	server := workertest.StartDevServer(t, workertest.WithDynamicConfig(map[string]any{"frontend.allowedExperiments": []string{"*"}}))
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	backend, err := client.DialContext(ctx, client.Options{HostPort: server.FrontendHostPort(), Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(backend.Close)
	for _, experiment := range []string{"false", "true"} {
		t.Run("chasm_requested_"+experiment, func(t *testing.T) {
			p := &scheduleResponseLossProxy{backend: backend.WorkflowService(), calls: map[string]int{}}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			rpc := grpc.NewServer()
			workflowservice.RegisterWorkflowServiceServer(rpc, p)
			done := make(chan error, 1)
			go func() { done <- rpc.Serve(listener) }()
			t.Cleanup(func() { rpc.Stop(); require.NoError(t, <-done) })
			c, err := client.DialContext(ctx, client.Options{HostPort: listener.Addr().String(), Namespace: "default"})
			require.NoError(t, err)
			t.Cleanup(c.Close)
			core, logs := observer.New(zap.InfoLevel)
			info := scheduleRecoveryInfo(c, zap.New(core).Sugar(), true, map[string]string{
				EnableChasmSchedulerFlag: experiment, SchedulerDurationPerIterationFlag: "30s",
			})
			require.NoError(t, new(SchedulerExecutor).Execute(ctx, info.NewRun(1)))
			p.mu.Lock()
			defer p.mu.Unlock()
			require.Equal(t, map[string]int{"create": 1, "update": 1, "delete": 1}, p.calls)
			outcomes := map[string]bool{}
			for _, row := range logs.All() {
				if s, ok := row.ContextMap()["outcome"].(string); ok {
					outcomes[s] = true
				}
			}
			for _, outcome := range []scheduleOutcome{outcomeObservedCreate, outcomeObservedWitness, outcomeObservedAbsence} {
				require.True(t, outcomes[string(outcome)], outcome)
			}
		})
	}
}
