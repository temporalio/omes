package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func KitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowParams) (interface{}, error) {
	b, _ := json.Marshal(params)
	workflow.GetLogger(ctx).Info("Started kitchen sink workflow", "params", string(b))

	// Handle all initial actions
	for _, action := range params.Actions {
		if shouldReturn, ret, err := handleAction(ctx, params, action); shouldReturn {
			return ret, err
		}
	}

	// Handle signal actions
	if params.ActionSignal != "" {
		actionCh := workflow.GetSignalChannel(ctx, params.ActionSignal)
		for {
			var action kitchensink.Action
			actionCh.Receive(ctx, &action)
			if shouldReturn, ret, err := handleAction(ctx, params, &action); shouldReturn {
				return ret, err
			}
		}
	}

	return nil, nil
}

func handleAction(
	ctx workflow.Context,
	params *kitchensink.WorkflowParams,
	action *kitchensink.Action,
) (bool, interface{}, error) {
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
		return true, lastResponse, lastErr

	default:
		return true, nil, fmt.Errorf("unrecognized action")
	}
	return false, nil, nil
}
