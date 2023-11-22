package kitchensink

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowInput) (interface{}, error) {
	b, _ := json.Marshal(params)
	workflow.GetLogger(ctx).Debug("Started kitchen sink workflow", "params", string(b))
	if params == nil {
		return nil, nil
	}

	// Handle initial set
	if params.InitialActions != nil {
		for _, actionSet := range params.InitialActions {
			shouldReturn, ret, err := handleActionSet(ctx, actionSet)
			if shouldReturn {
				return ret, err
			}
		}
	}

	// Handle signal action sets
	actionSetCh := workflow.GetSignalChannel(ctx, "do_actions_signal")
	for {
		var actionSet kitchensink.ActionSet
		actionSetCh.Receive(ctx, &actionSet)
		if shouldReturn, ret, err := handleActionSet(ctx, &actionSet); shouldReturn {
			return ret, err
		}
	}
}

func handleActionSet(
	ctx workflow.Context,
	set *kitchensink.ActionSet,
) (shouldReturn bool, returnValue interface{}, err error) {
	// If these are non-concurrent, just execute and return if requested
	if !set.Concurrent {
		for _, action := range set.Actions {
			if shouldReturn, returnValue, err = handleAction(ctx, action); shouldReturn {
				return
			}
		}
		return
	}
	// With a concurrent set, we'll use a coroutine for each, only updating the
	// return values if we should return, then awaiting on that or completion
	var actionsCompleted int
	for _, action := range set.Actions {
		action := action
		workflow.Go(ctx, func(ctx workflow.Context) {
			if maybeShouldReturn, maybeReturnValue, maybeErr := handleAction(ctx, action); maybeShouldReturn {
				shouldReturn, returnValue, err = maybeShouldReturn, maybeReturnValue, maybeErr
			}
			actionsCompleted++
		})
	}
	awaitErr := workflow.Await(ctx, func() bool {
		return shouldReturn || actionsCompleted == len(set.Actions)
	})
	if awaitErr != nil {
		return true, nil, fmt.Errorf("failed waiting on actions: %w", err)
	}
	return
}

func handleAction(
	ctx workflow.Context,
	action *kitchensink.Action,
) (shouldReturn bool, returnValue interface{}, err error) {
	if rr := action.GetReturnResult(); rr != nil {
		return true, rr.ReturnThis, nil
	} else if re := action.GetReturnError(); re != nil {
		return true, nil, temporal.NewApplicationError(re.Failure.Message, "")
	} else if can := action.GetContinueAsNew(); can != nil {
		return true, nil, workflow.NewContinueAsNewError(ctx, KitchenSinkWorkflow, can.Arguments)
	} else if timer := action.GetTimer(); timer != nil {
		if err := workflow.Sleep(ctx, time.Duration(timer.Milliseconds)*time.Millisecond); err != nil {
			return true, nil, err
		}
	} else if act := action.GetExecActivity(); act != nil {
		return launchActivity(ctx, action.GetExecActivity())
	} else if child := action.GetExecChildWorkflow(); child != nil {
		// Use name if present, otherwise use this one
		childType := "kitchenSink"
		if child.WorkflowType != "" {
			childType = child.WorkflowType
		}
		err := workflow.ExecuteChildWorkflow(ctx, childType, child.GetInput()).Get(ctx, nil)
		return false, nil, err
	} else if action.GetNestedActionSet() != nil {
		return handleActionSet(ctx, action.GetNestedActionSet())
	} else {
		return true, nil, fmt.Errorf("unrecognized action")
	}
	return false, nil, nil
}

func launchActivity(ctx workflow.Context, act *kitchensink.ExecuteActivityAction) (bool, interface{}, error) {
	var actCtx workflow.Context
	if act.GetIsLocal() != nil {
		opts := workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
		}
		actCtx = workflow.WithLocalActivityOptions(ctx, opts)
	} else {
		waitForCancel := false
		if remote := act.GetRemote(); remote != nil {
			if remote.GetCancellationType() == kitchensink.ActivityCancellationType_WAIT_CANCELLATION_COMPLETED {
				waitForCancel = true
			}
		}
		opts := workflow.ActivityOptions{
			TaskQueue:              act.TaskQueue,
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			ScheduleToStartTimeout: act.ScheduleToStartTimeout.AsDuration(),
			WaitForCancellation:    waitForCancel,
			HeartbeatTimeout:       act.HeartbeatTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
		}
		actCtx = workflow.WithActivityOptions(ctx, opts)
	}
	err := workflow.ExecuteActivity(actCtx, act.ActivityType, act.Arguments).Get(actCtx, nil)
	return false, nil, err
}

// Noop is used as a no-op activity
func Noop(_ context.Context) error {
	return nil
}

func convertFromPBRetryPolicy(retryPolicy *common.RetryPolicy) *temporal.RetryPolicy {
	if retryPolicy == nil {
		return nil
	}

	p := temporal.RetryPolicy{
		BackoffCoefficient:     retryPolicy.BackoffCoefficient,
		MaximumAttempts:        retryPolicy.MaximumAttempts,
		NonRetryableErrorTypes: retryPolicy.NonRetryableErrorTypes,
	}

	// Avoid nil pointer dereferences
	if v := retryPolicy.MaximumInterval; v != nil {
		p.MaximumInterval = *v
	}
	if v := retryPolicy.InitialInterval; v != nil {
		p.InitialInterval = *v
	}

	return &p
}
