package kitchensink

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowParams) (interface{}, error) {
	b, _ := json.Marshal(params)
	workflow.GetLogger(ctx).Debug("Started kitchen sink workflow", "params", string(b))
	if params == nil {
		return nil, nil
	}

	// Handle initial set
	shouldReturn, ret, err := handleActionSet(ctx, params, &params.ActionSet)
	if shouldReturn {
		return ret, err
	}

	// Handle signal action sets
	if params.ActionSetSignal != "" {
		actionSetCh := workflow.GetSignalChannel(ctx, params.ActionSetSignal)
		for {
			var actionSet kitchensink.ActionSet
			actionSetCh.Receive(ctx, &actionSet)
			if shouldReturn, ret, err = handleActionSet(ctx, params, &actionSet); shouldReturn {
				return ret, err
			}
		}
	}

	return ret, err
}

func handleActionSet(
	ctx workflow.Context,
	params *kitchensink.WorkflowParams,
	set *kitchensink.ActionSet,
) (shouldReturn bool, returnValue interface{}, err error) {
	// If these are non-concurrent, just execute and return if requested
	if !set.Concurrent {
		for _, action := range set.Actions {
			if shouldReturn, returnValue, err = handleAction(ctx, params, action); shouldReturn {
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
			if maybeShouldReturn, maybeReturnValue, maybeErr := handleAction(ctx, params, action); maybeShouldReturn {
				shouldReturn, returnValue, err = maybeShouldReturn, maybeReturnValue, maybeErr
			}
			actionsCompleted++
		})
	}
	awaitErr := workflow.Await(ctx, func() bool { return shouldReturn || actionsCompleted == len(set.Actions) })
	if awaitErr != nil {
		return true, nil, fmt.Errorf("failed waiting on actions: %w", err)
	}
	return
}

func handleAction(
	ctx workflow.Context,
	params *kitchensink.WorkflowParams,
	action *kitchensink.Action,
) (shouldReturn bool, returnValue interface{}, err error) {
	info := workflow.GetInfo(ctx)
	switch {
	case action.Result != nil:
		if action.Result.RunID {
			return true, info.WorkflowExecution.RunID, nil
		}
		return true, action.Result.Value, nil

	case action.Error != nil:
		if action.Error.Attempt {
			return true, nil, fmt.Errorf("attempt %v", info.Attempt)
		}
		var details []interface{}
		if action.Error.Details != nil {
			details = append(details, action.Error.Details)
		}
		return true, nil, temporal.NewApplicationError(action.Error.Message, "", details...)

	case action.ContinueAsNew != nil:
		if action.ContinueAsNew.WhileAboveZero > 0 {
			action.ContinueAsNew.WhileAboveZero--
			return true, nil, workflow.NewContinueAsNewError(ctx, KitchenSinkWorkflow, params)
		}

	case action.Sleep != nil:
		if err := workflow.Sleep(ctx, time.Duration(action.Sleep.Millis)*time.Millisecond); err != nil {
			return true, nil, err
		}

	case action.QueryHandler != nil:
		err := workflow.SetQueryHandler(ctx, action.QueryHandler.Name, func(arg string) (string, error) { return arg, nil })
		if err != nil {
			return true, nil, err
		}

	case action.Signal != nil:
		workflow.GetSignalChannel(ctx, action.Signal.Name).Receive(ctx, nil)

	case action.ExecuteActivity != nil:
		opts := workflow.ActivityOptions{
			TaskQueue:              action.ExecuteActivity.TaskQueue,
			ScheduleToCloseTimeout: time.Duration(action.ExecuteActivity.ScheduleToCloseTimeoutMS) * time.Millisecond,
			StartToCloseTimeout:    time.Duration(action.ExecuteActivity.StartToCloseTimeoutMS) * time.Millisecond,
			ScheduleToStartTimeout: time.Duration(action.ExecuteActivity.ScheduleToStartTimeoutMS) * time.Millisecond,
			WaitForCancellation:    action.ExecuteActivity.WaitForCancellation,
			HeartbeatTimeout:       time.Duration(action.ExecuteActivity.HeartbeatTimeoutMS) * time.Millisecond,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval:        1 * time.Millisecond,
				BackoffCoefficient:     1.01,
				MaximumInterval:        2 * time.Millisecond,
				MaximumAttempts:        1,
				NonRetryableErrorTypes: action.ExecuteActivity.NonRetryableErrorTypes,
			},
		}
		if opts.StartToCloseTimeout == 0 && opts.ScheduleToCloseTimeout == 0 {
			opts.ScheduleToCloseTimeout = 3 * time.Minute
		}
		if action.ExecuteActivity.RetryMaxAttempts > 1 {
			opts.RetryPolicy.MaximumAttempts = int32(action.ExecuteActivity.RetryMaxAttempts)
		}
		var lastErr error
		var lastResponse string
		count := action.ExecuteActivity.Count
		if count == 0 {
			count = 1
		}
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) { lastErr = fmt.Errorf("context closed") })
		for i := 0; i < count; i++ {
			actCtx := workflow.WithActivityOptions(ctx, opts)
			if action.ExecuteActivity.CancelAfterMS > 0 {
				var cancel workflow.CancelFunc
				actCtx, cancel = workflow.WithCancel(actCtx)
				workflow.Go(actCtx, func(actCtx workflow.Context) {
					workflow.Sleep(actCtx, time.Duration(action.ExecuteActivity.CancelAfterMS)*time.Millisecond)
					cancel()
				})
			}
			args := action.ExecuteActivity.Args
			if action.ExecuteActivity.IndexAsArg {
				args = []interface{}{i}
			}
			sel.AddFuture(workflow.ExecuteActivity(actCtx, action.ExecuteActivity.Name, args...),
				func(fut workflow.Future) { lastErr = fut.Get(actCtx, &lastResponse) })
		}
		for i := 0; i < count && lastErr == nil; i++ {
			sel.Select(ctx)
		}
		return lastErr != nil, lastResponse, lastErr

	case action.ExecuteChildWorkflow != nil:
		// Use name if present, otherwise use this one
		var childWorkflow interface{} = KitchenSinkWorkflow
		if action.ExecuteChildWorkflow.Name != "" {
			childWorkflow = action.ExecuteChildWorkflow.Name
		}
		// Use params if given, otherwise use args (or nothing)
		args := action.ExecuteChildWorkflow.Args
		if action.ExecuteChildWorkflow.Params != nil {
			if len(args) > 0 {
				return true, nil, fmt.Errorf("cannot have child args and child workflow params")
			}
			args = []interface{}{action.ExecuteChildWorkflow.Params}
		}
		// Start all children on selector
		var lastErr error
		count := action.ExecuteChildWorkflow.Count
		if count == 0 {
			count = 1
		}
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(ctx.Done(), func(workflow.ReceiveChannel, bool) { lastErr = fmt.Errorf("context closed") })
		for i := 0; i < count; i++ {
			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{})
			sel.AddFuture(workflow.ExecuteChildWorkflow(childCtx, childWorkflow, args...),
				func(fut workflow.Future) { lastErr = fut.Get(childCtx, nil) })
		}
		for i := 0; i < count && lastErr == nil; i++ {
			sel.Select(ctx)
		}
		return lastErr != nil, nil, lastErr

	case action.NestedActionSet != nil:
		return handleActionSet(ctx, params, action.NestedActionSet)

	default:
		return true, nil, fmt.Errorf("unrecognized action")
	}
	return false, nil, nil
}

func Noop(_ context.Context) error {
	return nil
}
