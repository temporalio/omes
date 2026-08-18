package kitchensink

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStandaloneActivityOperatorCommandTypesStartOnWorker(t *testing.T) {
	tests := []struct {
		name           string
		commandType    DoStandaloneActivityOperatorCommands_CommandType
		wantCalls      []string
		wantRequestIDs int
	}{
		{
			name:           "pause and unpause",
			commandType:    DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
			wantCalls:      []string{"describe-started", "pause", "describe-paused", "unpause", "describe-restarted", "get"},
			wantRequestIDs: 2,
		},
		{
			name:           "reset",
			commandType:    DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
			wantCalls:      []string{"describe-started", "reset", "describe-reset-requested", "describe-reset-started", "get"},
			wantRequestIDs: 1,
		},
		{
			name:           "update",
			commandType:    DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE,
			wantCalls:      []string{"describe-started", "update", "get"},
			wantRequestIDs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &operatorCommandTestService{t: t, commandType: tt.commandType}
			testClient := &operatorCommandTestClient{service: service}
			executor := &ClientActionsExecutor{Client: testClient, Namespace: "test-namespace"}

			err := executor.executeStandaloneActivityOperatorCommands(context.Background(),
				&DoStandaloneActivityOperatorCommands{
					CommandType: tt.commandType,
					Activity: &ExecuteActivityAction{
						ActivityType:        &ExecuteActivityAction_Heartbeat{},
						TaskQueue:           "worker-task-queue",
						StartToCloseTimeout: durationpb.New(15 * time.Second),
						HeartbeatTimeout:    durationpb.New(3 * time.Second),
					},
				})

			require.NoError(t, err)
			require.Equal(t, "worker-task-queue", testClient.startedTaskQueue)
			require.Equal(t, tt.wantCalls, service.calls)
			require.Len(t, service.requestIDs, tt.wantRequestIDs)
			require.Len(t, uniqueStrings(service.requestIDs), tt.wantRequestIDs)
			require.True(t, testClient.gotResult)
		})
	}
}

func TestStandaloneActivityOperatorCommandRejectsUnsupportedTypeBeforeStarting(t *testing.T) {
	testClient := &operatorCommandTestClient{service: &operatorCommandTestService{t: t}}
	executor := &ClientActionsExecutor{Client: testClient, Namespace: "test-namespace"}

	err := executor.executeStandaloneActivityOperatorCommands(context.Background(),
		&DoStandaloneActivityOperatorCommands{
			CommandType: DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UNSPECIFIED,
			Activity: &ExecuteActivityAction{
				ActivityType:        &ExecuteActivityAction_Heartbeat{},
				TaskQueue:           "worker-task-queue",
				StartToCloseTimeout: durationpb.New(15 * time.Second),
				HeartbeatTimeout:    durationpb.New(3 * time.Second),
			},
		})

	require.ErrorContains(t, err, "unsupported standalone activity operator command type")
	require.Empty(t, testClient.startedTaskQueue,
		"invalid input must not start a standalone activity")
}

func TestWaitForStandaloneActivityStateHasBoundedContext(t *testing.T) {
	service := &stateWaitDeadlineTestService{}

	startedAt := time.Now()
	_, err := waitForStandaloneActivityState(context.Background(), service,
		standaloneActivityOperatorTarget{
			namespace:  "test-namespace",
			activityID: "activity-id",
			runID:      "activity-run-id",
		}, func(*activitypb.ActivityExecutionInfo) bool { return false })
	returnedAt := time.Now()

	require.ErrorIs(t, err, context.Canceled)
	require.True(t, service.hasDeadline, "Describe should receive a bounded context")
	require.False(t, service.deadline.Before(startedAt.Add(operatorCommandStateWaitTimeout)))
	require.False(t, service.deadline.After(returnedAt.Add(operatorCommandStateWaitTimeout)))
}

func TestWaitForStandaloneActivityStateErrorIncludesTargetAndLastObservedState(t *testing.T) {
	service := &stateWaitDiagnosticTestService{}

	_, err := waitForStandaloneActivityState(context.Background(), service,
		standaloneActivityOperatorTarget{
			namespace:  "test-namespace",
			activityID: "activity-id",
			runID:      "activity-run-id",
		}, func(*activitypb.ActivityExecutionInfo) bool { return false })

	require.ErrorIs(t, err, context.DeadlineExceeded)
	for _, want := range []string{
		`namespace="test-namespace"`,
		`activity_id="activity-id"`,
		`run_id="activity-run-id"`,
		"status=Running",
		"run_state=Started",
		"transition_count=7",
		"heartbeat_timeout=4s",
	} {
		require.ErrorContains(t, err, want)
	}
}

type stateWaitDeadlineTestService struct {
	workflowservicepb.WorkflowServiceClient
	deadline    time.Time
	hasDeadline bool
}

type stateWaitDiagnosticTestService struct {
	workflowservicepb.WorkflowServiceClient
	describeCalls int
}

func (s *stateWaitDiagnosticTestService) DescribeActivityExecution(
	_ context.Context,
	_ *workflowservicepb.DescribeActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	s.describeCalls++
	if s.describeCalls == 1 {
		response := operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
			enumspb.PENDING_ACTIVITY_STATE_STARTED, 7, 4*time.Second)
		response.LongPollToken = []byte("long-poll-token")
		return response, nil
	}
	return nil, context.DeadlineExceeded
}

func (s *stateWaitDeadlineTestService) DescribeActivityExecution(
	ctx context.Context,
	_ *workflowservicepb.DescribeActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	s.deadline, s.hasDeadline = ctx.Deadline()
	return nil, context.Canceled
}

type operatorCommandTestClient struct {
	client.Client
	service          *operatorCommandTestService
	startedTaskQueue string
	gotResult        bool
}

func (c *operatorCommandTestClient) ExecuteActivity(
	_ context.Context,
	options client.StartActivityOptions,
	_ any,
	_ ...any,
) (client.ActivityHandle, error) {
	c.startedTaskQueue = options.TaskQueue
	return &operatorCommandTestHandle{client: c}, nil
}

func (c *operatorCommandTestClient) WorkflowService() workflowservicepb.WorkflowServiceClient {
	return c.service
}

type operatorCommandTestHandle struct{ client *operatorCommandTestClient }

func (*operatorCommandTestHandle) GetID() string    { return "activity-id" }
func (*operatorCommandTestHandle) GetRunID() string { return "activity-run-id" }
func (h *operatorCommandTestHandle) Get(context.Context, any) error {
	h.client.gotResult = true
	h.client.service.calls = append(h.client.service.calls, "get")
	return nil
}
func (*operatorCommandTestHandle) Describe(context.Context, client.DescribeActivityOptions) (*client.ActivityExecutionDescription, error) {
	return nil, nil
}
func (*operatorCommandTestHandle) Cancel(context.Context, client.CancelActivityOptions) error {
	return nil
}
func (*operatorCommandTestHandle) Terminate(context.Context, client.TerminateActivityOptions) error {
	return nil
}

type operatorCommandTestService struct {
	workflowservicepb.WorkflowServiceClient
	t           *testing.T
	commandType DoStandaloneActivityOperatorCommands_CommandType
	phase       int
	calls       []string
	requestIDs  []string
}

func (s *operatorCommandTestService) DescribeActivityExecution(
	_ context.Context,
	_ *workflowservicepb.DescribeActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	if s.phase == 0 {
		s.phase = 1
		s.calls = append(s.calls, "describe-started")
		response := operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
			enumspb.PENDING_ACTIVITY_STATE_STARTED, 1, 3*time.Second)
		response.Info.Attempt = 1
		response.Info.LastStartedTime = timestamppb.New(time.Unix(100, 0))
		return response, nil
	}

	switch s.commandType {
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE:
		switch s.phase {
		case 2:
			s.phase = 3
			s.calls = append(s.calls, "describe-paused")
			return operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED,
				enumspb.PENDING_ACTIVITY_STATE_PAUSED, 2, 3*time.Second), nil
		case 4:
			s.phase = 5
			s.calls = append(s.calls, "describe-restarted")
			return operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
				enumspb.PENDING_ACTIVITY_STATE_STARTED, 3, 3*time.Second), nil
		}
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET:
		switch s.phase {
		case 2:
			s.phase = 3
			s.calls = append(s.calls, "describe-reset-requested")
			response := operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
				enumspb.PENDING_ACTIVITY_STATE_STARTED, 2, 3*time.Second)
			response.Info.Attempt = 1
			response.Info.LastStartedTime = timestamppb.New(time.Unix(100, 0))
			response.LongPollToken = []byte("reset-requested")
			return response, nil
		case 3:
			s.phase = 4
			s.calls = append(s.calls, "describe-reset-started")
			response := operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
				enumspb.PENDING_ACTIVITY_STATE_STARTED, 4, 3*time.Second)
			response.Info.Attempt = 1
			response.Info.LastStartedTime = timestamppb.New(time.Unix(101, 0))
			return response, nil
		}
	}
	return nil, fmt.Errorf("unexpected describe at phase %d for %s", s.phase, s.commandType)
}

func (s *operatorCommandTestService) PauseActivityExecution(
	_ context.Context,
	r *workflowservicepb.PauseActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.PauseActivityExecutionResponse, error) {
	s.requireCommandTypeAndPhase(DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE, 1)
	s.phase = 2
	s.calls = append(s.calls, "pause")
	s.recordRequestID(r.RequestId)
	return &workflowservicepb.PauseActivityExecutionResponse{}, nil
}

func (s *operatorCommandTestService) UnpauseActivityExecution(
	_ context.Context,
	r *workflowservicepb.UnpauseActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.UnpauseActivityExecutionResponse, error) {
	s.requireCommandTypeAndPhase(DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE, 3)
	s.phase = 4
	s.calls = append(s.calls, "unpause")
	s.recordRequestID(r.RequestId)
	return &workflowservicepb.UnpauseActivityExecutionResponse{}, nil
}

func (s *operatorCommandTestService) ResetActivityExecution(
	_ context.Context,
	r *workflowservicepb.ResetActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.ResetActivityExecutionResponse, error) {
	s.requireCommandTypeAndPhase(DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET, 1)
	s.phase = 2
	s.calls = append(s.calls, "reset")
	s.recordRequestID(r.RequestId)
	require.True(s.t, r.ResetHeartbeat)
	return &workflowservicepb.ResetActivityExecutionResponse{}, nil
}

func (s *operatorCommandTestService) UpdateActivityExecutionOptions(
	_ context.Context,
	r *workflowservicepb.UpdateActivityExecutionOptionsRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.UpdateActivityExecutionOptionsResponse, error) {
	s.requireCommandTypeAndPhase(DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE, 1)
	s.phase = 2
	s.calls = append(s.calls, "update")
	s.recordRequestID(r.RequestId)
	require.Equal(s.t, []string{"heartbeat_timeout"}, r.UpdateMask.GetPaths())
	require.Equal(s.t, 4*time.Second, r.ActivityOptions.GetHeartbeatTimeout().AsDuration())
	return &workflowservicepb.UpdateActivityExecutionOptionsResponse{ActivityOptions: r.ActivityOptions}, nil
}

func (s *operatorCommandTestService) requireCommandTypeAndPhase(
	commandType DoStandaloneActivityOperatorCommands_CommandType,
	phase int,
) {
	require.Equal(s.t, commandType, s.commandType)
	require.Equal(s.t, phase, s.phase)
}

func (s *operatorCommandTestService) recordRequestID(requestID string) {
	require.NotEmpty(s.t, requestID)
	s.requestIDs = append(s.requestIDs, requestID)
}

func operatorCommandDescribe(
	status enumspb.ActivityExecutionStatus,
	runState enumspb.PendingActivityState,
	transitionCount int64,
	heartbeatTimeout time.Duration,
) *workflowservicepb.DescribeActivityExecutionResponse {
	return &workflowservicepb.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{
		Status:               status,
		RunState:             runState,
		StateTransitionCount: transitionCount,
		HeartbeatTimeout:     durationpb.New(heartbeatTimeout),
	}}
}

func uniqueStrings(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
