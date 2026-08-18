package kitchensink

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	operatorCommandIdentity         = "omes"
	operatorCommandReason           = "omes standalone activity operator command workload"
	operatorCommandStateWaitTimeout = time.Minute
)

func (e *ClientActionsExecutor) executeStandaloneActivityOperatorCommands(
	ctx context.Context,
	action *DoStandaloneActivityOperatorCommands,
) error {
	act := action.GetActivity()
	if act == nil {
		return fmt.Errorf("DoStandaloneActivityOperatorCommands.activity is required")
	}
	if act.TaskQueue == "" {
		return fmt.Errorf("DoStandaloneActivityOperatorCommands.activity.task_queue is required")
	}
	commandType := action.GetCommandType()
	switch commandType {
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
		DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
		DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE:
	default:
		return fmt.Errorf("unsupported standalone activity operator command type %s", commandType)
	}

	activityType, args := ActivityNameAndArgs(act)
	activityID := fmt.Sprintf("standalone-activity-operator-%s", uuid.NewString())
	handle, err := e.Client.ExecuteActivity(ctx, client.StartActivityOptions{
		ID:                     activityID,
		TaskQueue:              act.TaskQueue,
		ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
		StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
		HeartbeatTimeout:       act.HeartbeatTimeout.AsDuration(),
		RetryPolicy:            ConvertFromPBRetryPolicy(act.RetryPolicy),
	}, activityType, args...)
	if err != nil {
		return fmt.Errorf("start standalone activity for operator commands: %w", err)
	}

	target := standaloneActivityOperatorTarget{
		namespace:  e.Namespace,
		activityID: activityID,
		runID:      handle.GetRunID(),
	}
	service := e.Client.WorkflowService()
	started, err := waitForStandaloneActivityState(ctx, service, target,
		func(info *activitypb.ActivityExecutionInfo) bool {
			return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING &&
				info.GetRunState() == enumspb.PENDING_ACTIVITY_STATE_STARTED
		})
	if err != nil {
		return fmt.Errorf("wait for standalone activity to start: %w", err)
	}

	switch commandType {
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE:
		err = exercisePauseUnpause(ctx, service, target)
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET:
		err = exerciseReset(ctx, service, target, started)
	case DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE:
		err = exerciseUpdate(ctx, service, target)
	}
	if err != nil {
		return err
	}
	if err := handle.Get(ctx, nil); err != nil {
		return fmt.Errorf("get standalone activity after operator commands: %w", err)
	}
	return nil
}

func exercisePauseUnpause(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
) error {
	if _, err := service.PauseActivityExecution(ctx, &workflowservicepb.PauseActivityExecutionRequest{
		Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
		Identity: operatorCommandIdentity, Reason: operatorCommandReason,
		ResourceId: target.resourceID(), RequestId: uuid.NewString(),
	}); err != nil {
		return fmt.Errorf("pause standalone activity: %w", err)
	}
	if _, err := waitForStandaloneActivityState(ctx, service, target,
		func(info *activitypb.ActivityExecutionInfo) bool {
			return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED &&
				info.GetRunState() == enumspb.PENDING_ACTIVITY_STATE_PAUSED
		}); err != nil {
		return fmt.Errorf("verify standalone activity paused: %w", err)
	}

	if _, err := service.UnpauseActivityExecution(ctx, &workflowservicepb.UnpauseActivityExecutionRequest{
		Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
		Identity: operatorCommandIdentity, Reason: operatorCommandReason,
		ResourceId: target.resourceID(), RequestId: uuid.NewString(),
	}); err != nil {
		return fmt.Errorf("unpause standalone activity: %w", err)
	}
	if _, err := waitForStandaloneActivityState(ctx, service, target,
		func(info *activitypb.ActivityExecutionInfo) bool {
			return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING &&
				info.GetRunState() == enumspb.PENDING_ACTIVITY_STATE_STARTED
		}); err != nil {
		return fmt.Errorf("verify standalone activity unpaused: %w", err)
	}
	return nil
}

func exerciseReset(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
	started *activitypb.ActivityExecutionInfo,
) error {
	if started.GetLastStartedTime() == nil {
		return fmt.Errorf("reset standalone activity: started activity is missing last started time")
	}
	previousStartedTime := started.GetLastStartedTime().AsTime()
	if _, err := service.ResetActivityExecution(ctx, &workflowservicepb.ResetActivityExecutionRequest{
		Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
		Identity: operatorCommandIdentity, ResourceId: target.resourceID(),
		RequestId: uuid.NewString(), ResetHeartbeat: true,
	}); err != nil {
		return fmt.Errorf("reset standalone activity: %w", err)
	}
	if _, err := waitForStandaloneActivityState(ctx, service, target,
		func(info *activitypb.ActivityExecutionInfo) bool {
			lastStartedTime := info.GetLastStartedTime()
			resetAttemptStarted := info.GetAttempt() == 1 &&
				lastStartedTime != nil &&
				lastStartedTime.AsTime().After(previousStartedTime)
			return info.GetStatus() == enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING &&
				info.GetRunState() == enumspb.PENDING_ACTIVITY_STATE_STARTED &&
				resetAttemptStarted
		}); err != nil {
		return fmt.Errorf("verify standalone activity reset: %w", err)
	}
	return nil
}

func exerciseUpdate(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
) error {
	const updatedHeartbeatTimeout = 4 * time.Second
	response, err := service.UpdateActivityExecutionOptions(ctx,
		&workflowservicepb.UpdateActivityExecutionOptionsRequest{
			Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
			Identity: operatorCommandIdentity, ResourceId: target.resourceID(),
			RequestId: uuid.NewString(),
			ActivityOptions: &activitypb.ActivityOptions{
				HeartbeatTimeout: durationpb.New(updatedHeartbeatTimeout),
			},
			UpdateMask: &fieldmaskpb.FieldMask{Paths: []string{"heartbeat_timeout"}},
		})
	if err != nil {
		return fmt.Errorf("update standalone activity options: %w", err)
	}
	if got := response.GetActivityOptions().GetHeartbeatTimeout().AsDuration(); got != updatedHeartbeatTimeout {
		return fmt.Errorf("update standalone activity options returned heartbeat timeout %v, want %v",
			got, updatedHeartbeatTimeout)
	}
	return nil
}

type standaloneActivityOperatorTarget struct {
	namespace  string
	activityID string
	runID      string
}

func (t standaloneActivityOperatorTarget) resourceID() string {
	return "activity:" + t.activityID
}

func waitForStandaloneActivityState(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
	matches func(*activitypb.ActivityExecutionInfo) bool,
) (*activitypb.ActivityExecutionInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, operatorCommandStateWaitTimeout)
	defer cancel()

	var longPollToken []byte
	var lastObserved *activitypb.ActivityExecutionInfo
	for {
		response, err := service.DescribeActivityExecution(ctx,
			&workflowservicepb.DescribeActivityExecutionRequest{
				Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
				LongPollToken: longPollToken,
			})
		if err != nil {
			if lastObserved == nil {
				return nil, fmt.Errorf(
					"describe standalone activity namespace=%q activity_id=%q run_id=%q: %w",
					target.namespace, target.activityID, target.runID, err)
			}
			return nil, fmt.Errorf(
				"describe standalone activity namespace=%q activity_id=%q run_id=%q after last observed state: status=%s run_state=%s transition_count=%d heartbeat_timeout=%s: %w",
				target.namespace, target.activityID, target.runID,
				lastObserved.GetStatus(), lastObserved.GetRunState(),
				lastObserved.GetStateTransitionCount(),
				lastObserved.GetHeartbeatTimeout().AsDuration(), err)
		}
		info := response.GetInfo()
		if info == nil {
			return nil, fmt.Errorf(
				"DescribeActivityExecution returned no activity info for namespace=%q activity_id=%q run_id=%q",
				target.namespace, target.activityID, target.runID)
		}
		lastObserved = info
		if matches(info) {
			return info, nil
		}
		longPollToken = response.GetLongPollToken()
		if len(longPollToken) == 0 {
			return nil, fmt.Errorf(
				"standalone activity namespace=%q activity_id=%q run_id=%q reached terminal state before expected state: status=%s run_state=%s transition_count=%d heartbeat_timeout=%s",
				target.namespace, target.activityID, target.runID,
				info.GetStatus(), info.GetRunState(), info.GetStateTransitionCount(),
				info.GetHeartbeatTimeout().AsDuration())
		}
	}
}
