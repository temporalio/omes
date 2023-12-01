package kitchensink

import (
	"context"
	"errors"
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
	if params != nil && params.InitialActions != nil {
		for _, actionSet := range params.InitialActions {
			if ret, err := handleActionSet(ctx, workflowState, actionSet); ret != nil || err != nil {
				workflow.GetLogger(ctx).Info("Finishing early", "ret", ret, "err", err)
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
				ret, err := handleActionSet(ctx, workflowState, actionSet)
				if ret != nil || err != nil {
					retOrErrChan.Send(ctx, ReturnOrErr{ret, err})
				}
			})
		}
	})
	for {
		var retOrErr ReturnOrErr
		retOrErrChan.Receive(ctx, &retOrErr)
		workflow.GetLogger(ctx).Info("Finishing workflow", "retOrErr", retOrErr)
		return retOrErr.retme, retOrErr.err
	}
}

func handleActionSet(
	ctx workflow.Context,
	workflowState *kitchensink.WorkflowState,
	set *kitchensink.ActionSet,
) (returnValue *common.Payload, err error) {
	// If these are non-concurrent, just execute and return if requested
	if !set.Concurrent {
		for _, action := range set.Actions {
			if returnValue, err = handleAction(ctx, workflowState, action); returnValue != nil || err != nil {
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
			if maybeReturnValue, maybeErr := handleAction(ctx, workflowState, action); maybeReturnValue != nil || maybeErr != nil {
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
	workflowState *kitchensink.WorkflowState,
	action *kitchensink.Action,
) (*common.Payload, error) {
	if rr := action.GetReturnResult(); rr != nil {
		return rr.ReturnThis, nil
	} else if re := action.GetReturnError(); re != nil {
		return nil, temporal.NewApplicationError(re.Failure.Message, "")
	} else if can := action.GetContinueAsNew(); can != nil {
		return nil, workflow.NewContinueAsNewError(ctx, KitchenSinkWorkflow, can.Arguments)
	} else if timer := action.GetTimer(); timer != nil {
		return nil, withAwaitableChoice(ctx, func(ctx workflow.Context) workflow.Future {
			fut, setter := workflow.NewFuture(ctx)
			workflow.Go(ctx, func(ctx workflow.Context) {
				_ = workflow.Sleep(ctx, time.Duration(timer.Milliseconds)*time.Millisecond)
				setter.Set(struct{}{}, nil)
			})
			return fut
		}, timer.AwaitableChoice)
	} else if act := action.GetExecActivity(); act != nil {
		return nil, launchActivity(ctx, action.GetExecActivity())
	} else if child := action.GetExecChildWorkflow(); child != nil {
		// Use name if present, otherwise use this one
		childType := "kitchenSink"
		if child.WorkflowType != "" {
			childType = child.WorkflowType
		}
		err := withAwaitableChoice(ctx, func(ctx workflow.Context) workflow.Future {
			return workflow.ExecuteChildWorkflow(ctx, childType, child.GetInput()[0])
		}, child.AwaitableChoice)
		return nil, err
	} else if patch := action.GetSetPatchMarker(); patch != nil {
		if workflow.GetVersion(ctx, patch.GetPatchId(), workflow.DefaultVersion, 1) == 1 {
			return handleAction(ctx, workflowState, patch.GetInnerAction())
		}
	} else if setWfState := action.GetSetWorkflowState(); setWfState != nil {
		workflowState = setWfState
	} else if awaitState := action.GetAwaitWorkflowState(); awaitState != nil {
		err := workflow.Await(ctx, func() bool {
			if val, ok := workflowState.Kvs[awaitState.Key]; ok {
				return val == awaitState.Value
			}
			return false
		})
		return nil, err
	} else if upsertMemo := action.GetUpsertMemo(); upsertMemo != nil {
		convertedMap := make(map[string]interface{}, len(upsertMemo.GetUpsertedMemo().Fields))
		for k, v := range upsertMemo.GetUpsertedMemo().Fields {
			convertedMap[k] = v
		}
		err := workflow.UpsertMemo(ctx, convertedMap)
		return nil, err
	} else if upsertSA := action.GetUpsertSearchAttributes(); upsertSA != nil {
		convertedMap := make(map[string]interface{}, len(upsertSA.GetSearchAttributes()))
		for k, v := range upsertSA.GetSearchAttributes() {
			convertedMap[k] = v
		}
		err := workflow.UpsertSearchAttributes(ctx, convertedMap)
		return nil, err
	} else if action.GetNestedActionSet() != nil {
		return handleActionSet(ctx, workflowState, action.GetNestedActionSet())
	} else {
		return nil, fmt.Errorf("unrecognized action")
	}
	return nil, nil
}

func launchActivity(ctx workflow.Context, act *kitchensink.ExecuteActivityAction) error {
	args := act.GetArguments()
	if len(args) == 0 {
		args = nil
	}
	if act.GetIsLocal() != nil {
		opts := workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
		}
		actCtx := workflow.WithLocalActivityOptions(ctx, opts)
		return withAwaitableChoice(actCtx, func(ctx workflow.Context) workflow.Future {
			return workflow.ExecuteLocalActivity(ctx, act.ActivityType, args)
		}, act.GetAwaitableChoice())
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
		return withAwaitableChoice(actCtx, func(ctx workflow.Context) workflow.Future {
			return workflow.ExecuteActivity(ctx, act.ActivityType, args)
		}, act.GetAwaitableChoice())
	}
}

func withAwaitableChoice(ctx workflow.Context, starter func(workflow.Context) workflow.Future,
	awaitChoice *kitchensink.AwaitableChoice) error {
	cancelCtx, cancel := workflow.WithCancel(ctx)
	fut := starter(cancelCtx)
	var err error
	didCancel := false
	if awaitChoice.GetAbandon() != nil {
		return nil
	} else if awaitChoice.GetCancelBeforeStarted() != nil {
		cancel()
		didCancel = true
		err = fut.Get(ctx, nil)
	} else if awaitChoice.GetCancelAfterStarted() != nil {
		_ = workflow.Sleep(ctx, time.Duration(1))
		cancel()
		didCancel = true
		err = fut.Get(ctx, nil)
	} else if awaitChoice.GetCancelAfterCompleted() != nil {
		res := fut.Get(ctx, nil)
		cancel()
		didCancel = true
		err = res
	} else {
		err = fut.Get(ctx, nil)
	}

	// If we intentionally cancelled we want to swallow the cancel error to avoid bombing out the
	// whole workflow
	var canceledErr *temporal.CanceledError
	if didCancel && errors.As(err, &canceledErr) {
		err = nil
	}
	return err
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
