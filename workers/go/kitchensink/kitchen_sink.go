package kitchensink

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowInput) (*common.Payload, error) {
	workflow.GetLogger(ctx).Debug("Started kitchen sink workflow")

	workflowState := &kitchensink.WorkflowState{}
	queryErr := workflow.SetQueryHandler(ctx, "report_state",
		func(input interface{}) (*kitchensink.WorkflowState, error) {
			return workflowState, nil
		})
	if queryErr != nil {
		return nil, queryErr
	}

	// Handle initial set
	if params.InitialActions != nil {
		for _, actionSet := range params.InitialActions {
			if ret, err := handleActionSet(ctx, actionSet); ret != nil || err != nil {
				workflow.GetLogger(ctx).Warn("Finishing early", "ret", ret, "err", err)
				return ret, err
			}
		}
	}

	// Handle signal action sets
	signalActionsChan := workflow.GetSignalChannel(ctx, "do_actions_signal")
	retOrErrChan := workflow.NewChannel(ctx)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			var sigActions kitchensink.DoSignal_DoSignalActions
			signalActionsChan.Receive(ctx, &sigActions)
			actionSet := sigActions.GetDoActionsInMain()
			if actionSet == nil {
				actionSet = sigActions.GetDoActions()
			}
			workflow.Go(ctx, func(ctx workflow.Context) {
				ret, err := handleActionSet(ctx, actionSet)
				if ret != nil || err != nil {
					retOrErrChan.Send(ctx, ReturnOrErr{ret, err})
				}
			})
		}
	})
	for {
		var retOrErr ReturnOrErr
		retOrErrChan.Receive(ctx, &retOrErr)
		workflow.GetLogger(ctx).Warn("Finishing workflow", "retOrErr", retOrErr)
		return retOrErr.retme, retOrErr.err
	}
}

func handleActionSet(
	ctx workflow.Context,
	set *kitchensink.ActionSet,
) (returnValue *common.Payload, err error) {
	// If these are non-concurrent, just execute and return if requested
	if !set.Concurrent {
		for _, action := range set.Actions {
			if returnValue, err = handleAction(ctx, action); returnValue != nil || err != nil {
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
			if maybeReturnValue, maybeErr := handleAction(ctx, action); maybeReturnValue != nil || maybeErr != nil {
				returnValue, err = maybeReturnValue, maybeErr
			}
			actionsCompleted++
		})
	}
	awaitErr := workflow.Await(ctx, func() bool {
		return (returnValue != nil || err != nil) || actionsCompleted == len(set.Actions)
	})
	if awaitErr != nil {
		return nil, fmt.Errorf("failed waiting on actions: %w", err)
	}
	return
}

func handleAction(
	ctx workflow.Context,
	action *kitchensink.Action,
) (returnValue *common.Payload, err error) {
	if rr := action.GetReturnResult(); rr != nil {
		return rr.ReturnThis, nil
	} else if re := action.GetReturnError(); re != nil {
		return nil, temporal.NewApplicationError(re.Failure.Message, "")
	} else if can := action.GetContinueAsNew(); can != nil {
		return nil, workflow.NewContinueAsNewError(ctx, KitchenSinkWorkflow, can.Arguments)
	} else if timer := action.GetTimer(); timer != nil {
		if err := workflow.Sleep(ctx, time.Duration(timer.Milliseconds)*time.Millisecond); err != nil {
			return nil, err
		}
	} else if act := action.GetExecActivity(); act != nil {
		return nil, launchActivity(ctx, action.GetExecActivity())
	} else if child := action.GetExecChildWorkflow(); child != nil {
		// Use name if present, otherwise use this one
		childType := "kitchenSink"
		if child.WorkflowType != "" {
			childType = child.WorkflowType
		}
		err := workflow.ExecuteChildWorkflow(ctx, childType, child.GetInput()).Get(ctx, nil)
		return nil, err
	} else if action.GetNestedActionSet() != nil {
		return handleActionSet(ctx, action.GetNestedActionSet())
	} else {
		return nil, fmt.Errorf("unrecognized action")
	}
	return nil, nil
}

func launchActivity(ctx workflow.Context, act *kitchensink.ExecuteActivityAction) error {
	if act.GetIsLocal() != nil {
		opts := workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
		}
		actCtx := workflow.WithLocalActivityOptions(ctx, opts)
		return workflow.ExecuteLocalActivity(actCtx, act.ActivityType, act.Arguments).Get(actCtx, nil)
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
		actCtx := workflow.WithActivityOptions(ctx, opts)
		return workflow.ExecuteActivity(actCtx, act.ActivityType, act.Arguments).Get(actCtx, nil)
	}
}

// Noop is used as a no-op activity
func Noop(_ context.Context, _ []*common.Payload) error {
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

type ReturnOrErr struct {
	retme *common.Payload
	err   error
}
