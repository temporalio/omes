package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	activitypb "go.temporal.io/api/activity/v1"
	batchpb "go.temporal.io/api/batch/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

const (
	activityBatchReason       = "omes standalone activity batch operation workload"
	activityBatchWaitTimeout  = 2 * time.Minute
	activityBatchPollInterval = time.Second
)

func (e *ClientActionsExecutor) executeStandaloneActivityBatchOperations(
	ctx context.Context,
	action *DoStandaloneActivityBatchOperations,
) error {
	act := action.GetActivity()
	if act == nil {
		return fmt.Errorf("DoStandaloneActivityBatchOperations.activity is required")
	}

	if act.GetTaskQueue() == "" {
		return fmt.Errorf("DoStandaloneActivityBatchOperations.activity.task_queue is required")
	}

	if action.GetBatchSize() <= 0 {
		return fmt.Errorf("DoStandaloneActivityBatchOperations.batch_size must be positive, got %d",
			action.GetBatchSize())
	}

	switch action.GetOperationType() {
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
		DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE,
		DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE:
	default:
		return fmt.Errorf("unsupported standalone activity batch operation type %s",
			action.GetOperationType())
	}

	activityType, args := ActivityNameAndArgs(act)
	targets := make([]standaloneActivityOperatorTarget, 0, action.GetBatchSize())
	for range action.GetBatchSize() {
		activityID := "standalone-activity-batch-" + uuid.NewString()
		handle, err := e.Client.ExecuteActivity(ctx, client.StartActivityOptions{
			ID:                     activityID,
			TaskQueue:              act.GetTaskQueue(),
			ScheduleToCloseTimeout: act.GetScheduleToCloseTimeout().AsDuration(),
			ScheduleToStartTimeout: act.GetScheduleToStartTimeout().AsDuration(),
			StartToCloseTimeout:    act.GetStartToCloseTimeout().AsDuration(),
			HeartbeatTimeout:       act.GetHeartbeatTimeout().AsDuration(),
			RetryPolicy:            ConvertFromPBRetryPolicy(act.GetRetryPolicy()),
		}, activityType, args...)
		if err != nil {
			return fmt.Errorf("start standalone activity for batch operation: %w", err)
		}

		targets = append(targets, standaloneActivityOperatorTarget{
			namespace:  e.Namespace,
			activityID: handle.GetID(),
			runID:      handle.GetRunID(),
		})
	}

	service := e.Client.WorkflowService()
	startWaitCtx, cancelStartWait := context.WithTimeout(ctx, operatorCommandStateWaitTimeout)
	defer cancelStartWait()

	for _, target := range targets {
		if _, err := waitForStandaloneActivityState(startWaitCtx, service, target,
			func(info *activitypb.ActivityExecutionInfo) bool {
				return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING &&
					info.GetRunState() == enumspb.PENDING_ACTIVITY_STATE_STARTED
			}); err != nil {
			return fmt.Errorf("wait for standalone activity batch target to start: %w", err)
		}
	}

	jobID := "omes-standalone-activity-batch-" + uuid.NewString()
	request := &workflowservicepb.StartBatchOperationRequest{
		Namespace:        e.Namespace,
		JobId:            jobID,
		Reason:           activityBatchReason,
		TargetExecutions: make([]*commonpb.Execution, 0, len(targets)),
	}
	for _, target := range targets {
		request.TargetExecutions = append(request.TargetExecutions, &commonpb.Execution{
			Type:       enumspb.EXECUTION_TYPE_ACTIVITY,
			BusinessId: target.activityID,
			RunId:      target.runID,
		})
	}

	switch action.GetOperationType() {
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL:
		request.Operation = &workflowservicepb.StartBatchOperationRequest_CancelActivitiesOperation{
			CancelActivitiesOperation: &batchpb.BatchOperationCancelActivities{
				Identity: operatorCommandIdentity,
				Reason:   activityBatchReason,
			},
		}
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE:
		request.Operation = &workflowservicepb.StartBatchOperationRequest_TerminateActivitiesOperation{
			TerminateActivitiesOperation: &batchpb.BatchOperationTerminateActivities{
				Identity: operatorCommandIdentity,
				Reason:   activityBatchReason,
			},
		}
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE:
		request.Operation = &workflowservicepb.StartBatchOperationRequest_DeleteActivitiesOperation{
			DeleteActivitiesOperation: &batchpb.BatchOperationDeleteActivities{},
		}
	}

	if _, err := service.StartBatchOperation(ctx, request); err != nil {
		return fmt.Errorf("start standalone activity batch operation: %w", err)
	}

	if err := waitForActivityBatchOperation(ctx, service, e.Namespace, jobID, len(targets)); err != nil {
		return err
	}

	if err := verifyActivityBatchTarget(ctx, service, targets[0], action.GetOperationType()); err != nil {
		return err
	}

	return nil
}

func waitForActivityBatchOperation(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	namespace string,
	jobID string,
	wantCount int,
) error {
	ctx, cancel := context.WithTimeout(ctx, activityBatchWaitTimeout)
	defer cancel()

	for {
		response, err := service.DescribeBatchOperation(ctx,
			&workflowservicepb.DescribeBatchOperationRequest{Namespace: namespace, JobId: jobID})
		if err != nil {
			return fmt.Errorf("describe standalone activity batch operation job_id=%q: %w", jobID, err)
		}

		switch response.GetState() {
		case enumspb.BATCH_OPERATION_STATE_COMPLETED:
			if response.GetTotalOperationCount() != int64(wantCount) ||
				response.GetCompleteOperationCount() != int64(wantCount) ||
				response.GetFailureOperationCount() != 0 {
				return fmt.Errorf(
					"standalone activity batch operation job_id=%q completed with total=%d completed=%d failed=%d, want total=%d completed=%d failed=0",
					jobID, response.GetTotalOperationCount(), response.GetCompleteOperationCount(),
					response.GetFailureOperationCount(), wantCount, wantCount)
			}

			return nil
		case enumspb.BATCH_OPERATION_STATE_RUNNING:
			timer := time.NewTimer(activityBatchPollInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("wait for standalone activity batch operation job_id=%q: %w", jobID, ctx.Err())
			case <-timer.C:
			}
		case enumspb.BATCH_OPERATION_STATE_FAILED:
			return fmt.Errorf("standalone activity batch operation job_id=%q failed: %s",
				jobID, response.GetReason())
		default:
			return fmt.Errorf("standalone activity batch operation job_id=%q has unexpected state %s",
				jobID, response.GetState())
		}
	}
}

func verifyActivityBatchTarget(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
	operationType DoStandaloneActivityBatchOperations_OperationType,
) error {
	switch operationType {
	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL:
		if _, err := waitForStandaloneActivityState(ctx, service, target,
			func(info *activitypb.ActivityExecutionInfo) bool {
				return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED
			}); err != nil {
			return fmt.Errorf("verify batch-canceled standalone activity: %w", err)
		}

	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE:
		if _, err := waitForStandaloneActivityState(ctx, service, target,
			func(info *activitypb.ActivityExecutionInfo) bool {
				return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED
			}); err != nil {
			return fmt.Errorf("verify batch-terminated standalone activity: %w", err)
		}

	case DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE:
		_, err := service.DescribeActivityExecution(ctx,
			&workflowservicepb.DescribeActivityExecutionRequest{
				Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
			})

		var notFound *serviceerror.NotFound
		if errors.As(err, &notFound) {
			return nil
		}

		if err != nil {
			return fmt.Errorf("verify batch-deleted standalone activity: %w", err)
		}

		return fmt.Errorf("verify batch-deleted standalone activity: activity still exists")
	default:
		return fmt.Errorf("unsupported standalone activity batch operation type %s", operationType)
	}
	return nil
}
