package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

const (
	operatorCommandIdentity         = "omes"
	operatorCommandReason           = "omes standalone activity operator command workload"
	operatorCommandStateWaitTimeout = time.Minute
	operatorCommandCleanupTimeout   = 10 * time.Second
)

func (e *ClientActionsExecutor) executeStandaloneActivityOperatorCommands(
	ctx context.Context,
	action *DoStandaloneActivityOperatorCommands,
) (retErr error) {
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
	target := standaloneActivityOperatorTarget{namespace: e.Namespace, activityID: activityID}
	service := e.Client.WorkflowService()
	// Arm cleanup before Start: a failed response may still have created the
	// activity. Its unique ID safely identifies this attempt even without a run ID.
	defer func() {
		if retErr == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), operatorCommandCleanupTimeout)
		defer cancel()
		if err := cleanupStandaloneOperatorActivity(cleanupCtx, service, target); err != nil {
			retErr = errors.Join(retErr, fmt.Errorf("clean up standalone activity %q: %w", activityID, err))
		}
	}()

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

	target.runID = handle.GetRunID()
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

func cleanupStandaloneOperatorActivity(
	ctx context.Context,
	service workflowservicepb.WorkflowServiceClient,
	target standaloneActivityOperatorTarget,
) error {
	isClosed := func() (bool, error) {
		for {
			response, err := service.DescribeActivityExecution(ctx, &workflowservicepb.DescribeActivityExecutionRequest{
				Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
			})
			if serviceerror.ToStatus(err).Code() == codes.NotFound {
				if target.runID != "" {
					return true, nil
				}
				// A lost Start response does not establish that the activity is
				// absent. Allow an accepted start to become observable, bounded
				// by the cleanup deadline, rather than silently leaving it behind.
				select {
				case <-time.After(100 * time.Millisecond):
					continue
				case <-ctx.Done():
					return false, fmt.Errorf("cannot resolve activity after failed start: %w", ctx.Err())
				}
			}
			if err != nil {
				return false, err
			}
			if response.GetInfo() == nil {
				return false, fmt.Errorf("DescribeActivityExecution returned no activity info during cleanup")
			}
			target.runID = response.GetInfo().GetRunId()
			switch response.GetInfo().GetStatus() {
			case enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED, enumspb.ACTIVITY_EXECUTION_STATUS_FAILED,
				enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED, enumspb.ACTIVITY_EXECUTION_STATUS_TIMED_OUT,
				enumspb.ACTIVITY_EXECUTION_STATUS_TERMINATED:
				return true, nil
			}
			return false, nil
		}
	}

	if closed, err := isClosed(); closed || err != nil {
		return err
	}
	_, err := service.TerminateActivityExecution(ctx, &workflowservicepb.TerminateActivityExecutionRequest{
		Namespace: target.namespace, ActivityId: target.activityID, RunId: target.runID,
		Identity: operatorCommandIdentity, Reason: "omes operator sequence failed or was cancelled",
		RequestId: uuid.NewString(),
	})
	if serviceerror.ToStatus(err).Code() == codes.NotFound {
		return nil
	}
	if serviceerror.ToStatus(err).Code() == codes.FailedPrecondition {
		// Completion may have raced with termination. Preserve an already closed
		// activity's result, but do not hide an unrelated cleanup failure.
		if closed, describeErr := isClosed(); closed && describeErr == nil {
			return nil
		}
	}
	return err
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
			if len(longPollToken) > 0 {
				// Long-poll expiry is an empty successful response. Keep the
				// token and retry, backing off when the server's deadline buffer
				// makes polls expire immediately near the caller's deadline.
				select {
				case <-time.After(100 * time.Millisecond):
					continue
				case <-ctx.Done():
					return nil, fmt.Errorf("wait for standalone activity state: %w", ctx.Err())
				}
			}
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
