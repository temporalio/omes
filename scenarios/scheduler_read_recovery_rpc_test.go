package scenarios

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type schedulerReadService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	mu                                   sync.Mutex
	schedule                             *schedulepb.Schedule
	id                                   string
	creates, describes, updates, deletes int
	childDeadlineMode                    string
	firstChildDeadline                   time.Time
	unavailableResponses                 int
}

func (s *schedulerReadService) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (s *schedulerReadService) CreateSchedule(_ context.Context, r *workflowservice.CreateScheduleRequest) (*workflowservice.CreateScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creates++
	s.id = r.ScheduleId
	s.schedule = proto.Clone(r.Schedule).(*schedulepb.Schedule)
	return &workflowservice.CreateScheduleResponse{}, nil
}
func (s *schedulerReadService) DescribeSchedule(ctx context.Context, r *workflowservice.DescribeScheduleRequest) (*workflowservice.DescribeScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.describes++
	if s.childDeadlineMode != "" {
		if s.firstChildDeadline.IsZero() {
			s.firstChildDeadline, _ = ctx.Deadline()
		}
		if s.childDeadlineMode == "call" && s.describes == 1 {
			<-ctx.Done()
			return nil, status.Error(codes.Canceled, "SDK child context ended")
		}
		if s.childDeadlineMode == "retry_backoff" && time.Now().Before(s.firstChildDeadline) {
			s.unavailableResponses++
			return nil, status.Error(codes.Unavailable, "retry until SDK child context expires")
		}
	} else if s.describes == 1 {
		return nil, status.Error(codes.DeadlineExceeded, "scripted server persistence timeout")
	}
	if s.schedule == nil || s.id != r.ScheduleId {
		return nil, status.Error(codes.NotFound, "missing")
	}
	return &workflowservice.DescribeScheduleResponse{Schedule: proto.Clone(s.schedule).(*schedulepb.Schedule), Info: &schedulepb.ScheduleInfo{}}, nil
}
func (s *schedulerReadService) UpdateSchedule(_ context.Context, r *workflowservice.UpdateScheduleRequest) (*workflowservice.UpdateScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates++
	s.schedule = proto.Clone(r.Schedule).(*schedulepb.Schedule)
	return &workflowservice.UpdateScheduleResponse{}, nil
}
func (s *schedulerReadService) DeleteSchedule(_ context.Context, r *workflowservice.DeleteScheduleRequest) (*workflowservice.DeleteScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes++
	s.schedule = nil
	return &workflowservice.DeleteScheduleResponse{}, nil
}

func TestSchedulerReadRecoverySDK(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		t.Run(name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			service := &schedulerReadService{}
			server := grpc.NewServer()
			workflowservice.RegisterWorkflowServiceServer(server, service)
			served := make(chan error, 1)
			go func() { served <- server.Serve(listener) }()
			t.Cleanup(func() { server.Stop(); require.NoError(t, <-served) })
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			c, err := client.DialContext(ctx, client.Options{HostPort: listener.Addr().String(), Namespace: "read-recovery"})
			require.NoError(t, err)
			defer c.Close()
			info := schedulerReadInfo(&schedulerReadAPI{}, enabled)
			info.Client = c
			err = new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
			if enabled {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "scripted server persistence timeout")
			}
			service.mu.Lock()
			defer service.mu.Unlock()
			require.Equal(t, 1, service.creates)
			require.Equal(t, 1, service.updates)
			require.Equal(t, 1, service.deletes)
			require.Nil(t, service.schedule)
			// Update's SDK implementation performs its own Describe before writing.
			wantDescribes := 2
			if enabled {
				wantDescribes = 3
			}
			require.Equal(t, wantDescribes, service.describes)
		})
	}
}

func TestSchedulerReadRecoverySDKChildDeadline(t *testing.T) {
	for _, mode := range []string{"call", "retry_backoff"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			service := &schedulerReadService{childDeadlineMode: mode}
			server := grpc.NewServer()
			workflowservice.RegisterWorkflowServiceServer(server, service)
			served := make(chan error, 1)
			go func() { served <- server.Serve(listener) }()
			t.Cleanup(func() { server.Stop(); require.NoError(t, <-served) })
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			c, err := client.DialContext(ctx, client.Options{HostPort: listener.Addr().String(), Namespace: "read-recovery"})
			require.NoError(t, err)
			defer c.Close()
			info := schedulerReadInfo(&schedulerReadAPI{}, true)
			info.Client = c
			err = new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
			require.NoError(t, err)
			require.NoError(t, ctx.Err())
			service.mu.Lock()
			defer service.mu.Unlock()
			require.Equal(t, 1, service.creates)
			require.Equal(t, 1, service.updates)
			require.Equal(t, 1, service.deletes)
			require.Nil(t, service.schedule)
			if mode == "call" {
				require.Equal(t, 3, service.describes)
			} else {
				require.Greater(t, service.unavailableResponses, 1)
				require.Equal(t, service.unavailableResponses+2, service.describes)
			}
		})
	}
}
