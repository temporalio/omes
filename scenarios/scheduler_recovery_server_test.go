package scenarios

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/journal"
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
	for _, tc := range []struct {
		name, chasm string
		journaled   bool
	}{
		{name: "classic", chasm: "false"},
		{name: "chasm", chasm: "true"},
		{name: "journaled_chasm", chasm: "true", journaled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
			options := map[string]string{EnableChasmSchedulerFlag: tc.chasm, SchedulerDurationPerIterationFlag: "30s"}
			scheduleCount, updatesPerSchedule := 1, 1
			if tc.journaled {
				const lanes, schedulesPerLane = 15, 10
				scheduleCount, updatesPerSchedule = lanes*schedulesPerLane, 3
				options[ScheduleCreationPerIterationFlag] = "10"
				options[ScheduleReadsPerCreationFlag] = "3"
				options[ScheduleUpdatesPerCreationFlag] = "3"
				options[SchedulerRecoveryTimeoutFlag] = "30s"
				options[SchedulerRecoveryBackoffFlag] = "100ms"
				options[OperationIntervalFlag] = "50ms"
				options[WaitTimeBeforeCleanupFlag] = "25s"
				info := scheduleRecoveryInfo(c, zap.New(core).Sugar(), true, options)
				info.ScenarioName = "scheduler_stress"
				info.Namespace = "default"
				info.MetricsHandler = client.MetricsNopHandler
				info.Configuration = loadgen.RunConfiguration{Iterations: lanes, MaxConcurrent: lanes, DoNotRegisterSearchAttributes: true}
				ids := make([]string, lanes)
				for i := range ids {
					ids[i] = fmt.Sprintf("scheduler/%d", i)
				}
				info.Recovery = &loadgen.RecoveryOptions{
					Client: &scenarioJournal{}, Epoch: 1, Contract: []byte("scheduler production shape"),
					Streams:       map[string][]string{info.ScenarioName: ids},
					GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(90 * time.Second),
				}
				executor := loadgen.GenericExecutor{Execute: func(ctx context.Context, run *loadgen.Run) error {
					return new(SchedulerExecutor).Execute(ctx, run)
				}}
				require.NoError(t, executor.Run(ctx, info))
				counts, err := info.Recovery.Totals(ctx, info.ScenarioName)
				require.NoError(t, err)
				require.Equal(t, journal.Counts{Allocated: lanes, Succeeded: lanes}, counts)
			} else {
				info := scheduleRecoveryInfo(c, zap.New(core).Sugar(), true, options)
				require.NoError(t, new(SchedulerExecutor).Execute(ctx, info.NewRun(1)))
			}
			p.mu.Lock()
			defer p.mu.Unlock()
			require.Equal(t, map[string]int{"create": scheduleCount, "update": scheduleCount * updatesPerSchedule, "delete": scheduleCount}, p.calls)
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
