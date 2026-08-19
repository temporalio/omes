package kitchensink

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestStandaloneActivityBatchOperationsTargetEveryStartedActivity(t *testing.T) {
	tests := []struct {
		name          string
		operationType DoStandaloneActivityBatchOperations_OperationType
		wantStatus    enumspb.ActivityExecutionStatus
		wantNotFound  bool
	}{
		{
			name:          "cancel",
			operationType: DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
			wantStatus:    enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED,
		},
		{
			name:          "terminate",
			operationType: DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE,
			wantStatus:    enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED,
		},
		{
			name:          "delete",
			operationType: DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE,
			wantNotFound:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &activityBatchTestService{
				t:             t,
				operationType: tt.operationType,
				batchSize:     5,
				wantStatus:    tt.wantStatus,
				wantNotFound:  tt.wantNotFound,
			}
			testClient := &activityBatchTestClient{service: service}
			executor := &ClientActionsExecutor{Client: testClient, Namespace: "test-namespace"}

			err := executor.executeStandaloneActivityBatchOperations(context.Background(),
				&DoStandaloneActivityBatchOperations{
					OperationType: tt.operationType,
					BatchSize:     5,
					Activity: &ExecuteActivityAction{
						ActivityType:        &ExecuteActivityAction_Heartbeat{},
						TaskQueue:           "worker-task-queue",
						StartToCloseTimeout: durationpb.New(2 * time.Minute),
						HeartbeatTimeout:    durationpb.New(3 * time.Second),
					},
				})

			require.NoError(t, err)
			require.Len(t, testClient.startedIDs, 5)
			require.Equal(t, "worker-task-queue", testClient.startedTaskQueue)
			require.True(t, service.batchStarted)
			require.True(t, service.batchDescribed)
			require.True(t, service.targetVerified)
		})
	}
}

func TestStandaloneActivityBatchOperationsRejectInvalidInputBeforeStarting(t *testing.T) {
	tests := []struct {
		name   string
		action *DoStandaloneActivityBatchOperations
	}{
		{
			name: "unspecified operation",
			action: &DoStandaloneActivityBatchOperations{
				OperationType: DoStandaloneActivityBatchOperations_OPERATION_TYPE_UNSPECIFIED,
				BatchSize:     5,
				Activity:      &ExecuteActivityAction{TaskQueue: "worker-task-queue"},
			},
		},
		{
			name: "zero batch size",
			action: &DoStandaloneActivityBatchOperations{
				OperationType: DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
				BatchSize:     0,
				Activity:      &ExecuteActivityAction{TaskQueue: "worker-task-queue"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testClient := &activityBatchTestClient{}
			executor := &ClientActionsExecutor{Client: testClient, Namespace: "test-namespace"}

			err := executor.executeStandaloneActivityBatchOperations(context.Background(), tt.action)

			require.Error(t, err)
			require.Empty(t, testClient.startedIDs)
		})
	}
}

func TestWaitForActivityBatchOperationRejectsFailedTargets(t *testing.T) {
	service := &completedActivityBatchTestService{
		response: &workflowservicepb.DescribeBatchOperationResponse{
			State:                  enumspb.BATCH_OPERATION_STATE_COMPLETED,
			TotalOperationCount:    5,
			CompleteOperationCount: 4,
			FailureOperationCount:  1,
		},
	}

	err := waitForActivityBatchOperation(context.Background(), service,
		"test-namespace", "batch-job", 5)

	require.ErrorContains(t, err, "total=5 completed=4 failed=1")
}

type activityBatchTestClient struct {
	client.Client
	service          *activityBatchTestService
	startedIDs       []string
	startedTaskQueue string
}

type completedActivityBatchTestService struct {
	workflowservicepb.WorkflowServiceClient
	response *workflowservicepb.DescribeBatchOperationResponse
}

func (s *completedActivityBatchTestService) DescribeBatchOperation(
	_ context.Context,
	_ *workflowservicepb.DescribeBatchOperationRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeBatchOperationResponse, error) {
	return s.response, nil
}

func (c *activityBatchTestClient) ExecuteActivity(
	_ context.Context,
	options client.StartActivityOptions,
	_ any,
	_ ...any,
) (client.ActivityHandle, error) {
	activityID := fmt.Sprintf("activity-%d", len(c.startedIDs))
	c.startedIDs = append(c.startedIDs, activityID)
	c.startedTaskQueue = options.TaskQueue
	return &activityBatchTestHandle{id: activityID}, nil
}

func (c *activityBatchTestClient) WorkflowService() workflowservicepb.WorkflowServiceClient {
	return c.service
}

type activityBatchTestHandle struct{ id string }

func (h *activityBatchTestHandle) GetID() string                { return h.id }
func (h *activityBatchTestHandle) GetRunID() string             { return "run-" + h.id }
func (*activityBatchTestHandle) Get(context.Context, any) error { return nil }
func (*activityBatchTestHandle) Describe(context.Context, client.DescribeActivityOptions) (*client.ActivityExecutionDescription, error) {
	return nil, nil
}
func (*activityBatchTestHandle) Cancel(context.Context, client.CancelActivityOptions) error {
	return nil
}
func (*activityBatchTestHandle) Terminate(context.Context, client.TerminateActivityOptions) error {
	return nil
}

type activityBatchTestService struct {
	workflowservicepb.WorkflowServiceClient
	t              *testing.T
	operationType  DoStandaloneActivityBatchOperations_OperationType
	batchSize      int
	wantStatus     enumspb.ActivityExecutionStatus
	wantNotFound   bool
	describeCalls  map[string]int
	batchStarted   bool
	batchDescribed bool
	targetVerified bool
}

func (s *activityBatchTestService) DescribeActivityExecution(
	_ context.Context,
	r *workflowservicepb.DescribeActivityExecutionRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	require.Equal(s.t, "test-namespace", r.GetNamespace())
	require.Equal(s.t, "run-"+r.GetActivityId(), r.GetRunId())
	if s.describeCalls == nil {
		s.describeCalls = make(map[string]int)
	}

	s.describeCalls[r.GetActivityId()]++
	if s.describeCalls[r.GetActivityId()] == 1 {
		return operatorCommandDescribe(enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING,
			enumspb.PENDING_ACTIVITY_STATE_STARTED, 1, 3*time.Second), nil
	}

	require.Equal(s.t, "activity-0", r.GetActivityId(), "only one representative target should be verified")
	s.targetVerified = true

	if s.wantNotFound {
		return nil, serviceerror.NewNotFound("activity deleted")
	}

	return operatorCommandDescribe(s.wantStatus,
		enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED, 2, 3*time.Second), nil
}

func (s *activityBatchTestService) StartBatchOperation(
	_ context.Context,
	r *workflowservicepb.StartBatchOperationRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.StartBatchOperationResponse, error) {
	require.Equal(s.t, "test-namespace", r.GetNamespace())
	require.NotEmpty(s.t, r.GetJobId())
	require.NotEmpty(s.t, r.GetReason())
	require.Len(s.t, r.GetTargetExecutions(), s.batchSize)
	require.Len(s.t, s.describeCalls, s.batchSize,
		"every target should be running before the batch starts")
	for i := range s.batchSize {
		require.Equal(s.t, 1, s.describeCalls[fmt.Sprintf("activity-%d", i)],
			"each target should be described as running exactly once before the batch starts")
	}

	for i, execution := range r.GetTargetExecutions() {
		require.Equal(s.t, enumspb.EXECUTION_TYPE_ACTIVITY, execution.GetType())
		require.Equal(s.t, fmt.Sprintf("activity-%d", i), execution.GetBusinessId())
		require.Equal(s.t, fmt.Sprintf("run-activity-%d", i), execution.GetRunId())
	}

	s.requireOperation(r)
	s.batchStarted = true

	return &workflowservicepb.StartBatchOperationResponse{}, nil
}

func (s *activityBatchTestService) DescribeBatchOperation(
	_ context.Context,
	r *workflowservicepb.DescribeBatchOperationRequest,
	_ ...grpc.CallOption,
) (*workflowservicepb.DescribeBatchOperationResponse, error) {
	require.True(s.t, s.batchStarted)
	require.Equal(s.t, "test-namespace", r.GetNamespace())
	require.NotEmpty(s.t, r.GetJobId())
	s.batchDescribed = true

	return &workflowservicepb.DescribeBatchOperationResponse{
		State:                  enumspb.BATCH_OPERATION_STATE_COMPLETED,
		TotalOperationCount:    int64(s.batchSize),
		CompleteOperationCount: int64(s.batchSize),
		FailureOperationCount:  0,
	}, nil
}

func (s *activityBatchTestService) requireOperation(r *workflowservicepb.StartBatchOperationRequest) {
	switch s.operationType {
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL:
		operation := r.GetCancelActivitiesOperation()
		require.NotNil(s.t, operation)
		require.Equal(s.t, "omes", operation.GetIdentity())
		require.NotEmpty(s.t, operation.GetReason())

	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE:
		operation := r.GetTerminateActivitiesOperation()
		require.NotNil(s.t, operation)
		require.Equal(s.t, "omes", operation.GetIdentity())
		require.NotEmpty(s.t, operation.GetReason())

	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE:
		require.NotNil(s.t, r.GetDeleteActivitiesOperation())

	default:
		require.Fail(s.t, "unexpected operation type")
	}
}
