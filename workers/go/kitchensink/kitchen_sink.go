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

type KSWorkflowState struct {
	workflowState *kitchensink.WorkflowState
}

func KitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowInput) (*common.Payload, error) {
	workflow.GetLogger(ctx).Debug("Started kitchen sink workflow")

	state := KSWorkflowState{
		workflowState: &kitchensink.WorkflowState{},
	}
	queryErr := workflow.SetQueryHandler(ctx, "report_state",
		func(input interface{}) (*kitchensink.WorkflowState, error) {
			return state.workflowState, nil
		})
	if queryErr != nil {
		return nil, queryErr
	}

	updateErr := workflow.SetUpdateHandlerWithOptions(ctx, "do_actions_update",
		func(ctx workflow.Context, actions *kitchensink.DoActionsUpdate) (rval interface{}, err error) {
			rval, err = state.handleActionSet(ctx, actions.GetDoActions())
			if rval == nil {
				rval = &state.workflowState
			}
			return rval, err
		}, workflow.UpdateHandlerOptions{
			Validator: func(ctx workflow.Context, actions *kitchensink.DoActionsUpdate) error {
				if actions.GetRejectMe() != nil {
					return errors.New("rejected")
				}
				return nil
			},
		})
	if updateErr != nil {
		return nil, updateErr
	}

	// Handle initial set
	if params != nil && params.InitialActions != nil {
		for _, actionSet := range params.InitialActions {
			if ret, err := state.handleActionSet(ctx, actionSet); ret != nil || err != nil {
				workflow.GetLogger(ctx).Debug("Finishing early", "ret", ret, "err", err)
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
				ret, err := state.handleActionSet(ctx, actionSet)
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

func (ws *KSWorkflowState) handleActionSet(
	ctx workflow.Context,
	set *kitchensink.ActionSet,
) (returnValue *common.Payload, err error) {
	// If these are non-concurrent, just execute and return if requested
	if !set.Concurrent {
		for _, action := range set.Actions {
			if returnValue, err = ws.handleAction(ctx, action); returnValue != nil || err != nil {
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
			if maybeReturnValue, maybeErr := ws.handleAction(ctx, action); maybeReturnValue != nil || maybeErr != nil {
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

func (ws *KSWorkflowState) handleAction(
	ctx workflow.Context,
	action *kitchensink.Action,
) (*common.Payload, error) {
	if rr := action.GetReturnResult(); rr != nil {
		return rr.ReturnThis, nil
	} else if re := action.GetReturnError(); re != nil {
		return nil, temporal.NewApplicationError(re.Failure.Message, "")
	} else if can := action.GetContinueAsNew(); can != nil {
		// Use string arg to avoid the SDK trying to convert payload to input type
		return nil, workflow.NewContinueAsNewError(ctx, "kitchenSink", can.GetArguments()[0])
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
		err := withAwaitableChoiceCustom(ctx, func(ctx workflow.Context) workflow.ChildWorkflowFuture {
			return workflow.ExecuteChildWorkflow(ctx, childType, child.GetInput()[0])
		}, child.AwaitableChoice,
			func(ctx workflow.Context, fut workflow.ChildWorkflowFuture) error {
				return fut.GetChildWorkflowExecution().Get(ctx, nil)
			},
			func(ctx workflow.Context, fut workflow.ChildWorkflowFuture) error {
				return fut.Get(ctx, nil)
			},
		)
		return nil, err
	} else if patch := action.GetSetPatchMarker(); patch != nil {
		if workflow.GetVersion(ctx, patch.GetPatchId(), workflow.DefaultVersion, 1) == 1 {
			return ws.handleAction(ctx, patch.GetInnerAction())
		}
	} else if setWfState := action.GetSetWorkflowState(); setWfState != nil {
		ws.workflowState = setWfState
	} else if awaitState := action.GetAwaitWorkflowState(); awaitState != nil {
		err := workflow.Await(ctx, func() bool {
			if val, ok := ws.workflowState.Kvs[awaitState.Key]; ok {
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
		return ws.handleActionSet(ctx, action.GetNestedActionSet())
	} else {
		return nil, fmt.Errorf("unrecognized action")
	}
	return nil, nil
}

func launchActivity(ctx workflow.Context, act *kitchensink.ExecuteActivityAction) error {
	actType := "noop"
	args := make([]interface{}, 0)
	if delay := act.GetDelay(); delay != nil {
		actType = "delay"
		args = append(args, delay.AsDuration())
	}
	if act.GetIsLocal() != nil {
		opts := workflow.LocalActivityOptions{
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
		}
		actCtx := workflow.WithLocalActivityOptions(ctx, opts)

		return withAwaitableChoice(actCtx, func(ctx workflow.Context) workflow.Future {
			return workflow.ExecuteLocalActivity(ctx, actType, args...)
		}, act.GetAwaitableChoice())
	} else {
		waitForCancel := false
		if remote := act.GetRemote(); remote != nil {
			if remote.GetCancellationType() == kitchensink.ActivityCancellationType_WAIT_CANCELLATION_COMPLETED {
				waitForCancel = true
			}
		}

		var priority temporal.Priority
		if pk := act.GetPriorityKey(); pk != 0 {
			priority.PriorityKey = int(pk)
		}
		if fk := act.GetFairnessKey(); fk != "" {
			return fmt.Errorf("fairness key is not supported yet")
		}
		if fw := act.GetFairnessWeight(); fw > 0 {
			return fmt.Errorf("fairness weight is not supported yet")
		}

		opts := workflow.ActivityOptions{
			TaskQueue:              act.TaskQueue,
			ScheduleToCloseTimeout: act.ScheduleToCloseTimeout.AsDuration(),
			StartToCloseTimeout:    act.StartToCloseTimeout.AsDuration(),
			ScheduleToStartTimeout: act.ScheduleToStartTimeout.AsDuration(),
			WaitForCancellation:    waitForCancel,
			HeartbeatTimeout:       act.HeartbeatTimeout.AsDuration(),
			RetryPolicy:            convertFromPBRetryPolicy(act.GetRetryPolicy()),
			Priority:               priority,
		}
		actCtx := workflow.WithActivityOptions(ctx, opts)
		return withAwaitableChoice(actCtx, func(ctx workflow.Context) workflow.Future {
			return workflow.ExecuteActivity(ctx, actType, args...)
		}, act.GetAwaitableChoice())
	}
}

func withAwaitableChoice[F workflow.Future](
	ctx workflow.Context,
	starter func(workflow.Context) F,
	awaitChoice *kitchensink.AwaitableChoice,
) error {
	return withAwaitableChoiceCustom(ctx, starter, awaitChoice,
		func(ctx workflow.Context, fut F) error {
			_ = workflow.Sleep(ctx, 1)
			return nil
		},
		func(ctx workflow.Context, fut F) error {
			return fut.Get(ctx, nil)
		})
}

func withAwaitableChoiceCustom[F workflow.Future](
	ctx workflow.Context,
	starter func(workflow.Context) F,
	awaitChoice *kitchensink.AwaitableChoice,
	afterStartedWaiter func(workflow.Context, F) error,
	afterCompletedWaiter func(workflow.Context, F) error,
) error {
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
		err = afterStartedWaiter(ctx, fut)
		if err != nil {
			return err
		}
		cancel()
		didCancel = true
		err = fut.Get(ctx, nil)
	} else if awaitChoice.GetCancelAfterCompleted() != nil {
		res := afterCompletedWaiter(ctx, fut)
		cancel()
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

// Delay runs for the provided delay period
func Delay(_ context.Context, delayFor time.Duration) error {
	time.Sleep(delayFor)
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

	if v := retryPolicy.MaximumInterval; v != nil {
		p.MaximumInterval = v.AsDuration()
	}
	if v := retryPolicy.InitialInterval; v != nil {
		p.InitialInterval = v.AsDuration()
	}

	return &p
}

type ReturnOrErr struct {
	retme *common.Payload
	err   error
}
