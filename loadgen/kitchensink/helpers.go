package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"
)

func NoOpSingleActivityActionSet() *ActionSet {
	return &ActionSet{
		Actions: []*Action{
			{
				Variant: &Action_ExecActivity{
					ExecActivity: &ExecuteActivityAction{
						ActivityType:        &ExecuteActivityAction_Noop{},
						StartToCloseTimeout: &durationpb.Duration{Seconds: 5},
					},
				},
			},
			{
				Variant: &Action_ReturnResult{
					ReturnResult: &ReturnResultAction{
						ReturnThis: &common.Payload{},
					},
				},
			},
		},
	}
}

func ResourceConsumingActivity(bytesToAllocate uint64, cpuYieldEveryNIters uint32, cpuYieldForMs uint32, runForSeconds int64) *Action {
	return &Action{
		Variant: &Action_ExecActivity{
			ExecActivity: &ExecuteActivityAction{
				ActivityType: &ExecuteActivityAction_Resources{
					Resources: &ExecuteActivityAction_ResourcesActivity{
						BytesToAllocate:          bytesToAllocate,
						CpuYieldEveryNIterations: cpuYieldEveryNIters,
						CpuYieldForMs:            cpuYieldForMs,
						RunFor:                   &durationpb.Duration{Seconds: runForSeconds},
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: runForSeconds * 2},
				RetryPolicy: &common.RetryPolicy{
					MaximumAttempts:    1,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

type ClientActionsExecutor struct {
	Client        client.Client
	StartOptions  client.StartWorkflowOptions
	WorkflowType  string
	WorkflowInput *WorkflowInput
	Handle        client.WorkflowRun
	runID         string
}

func (e *ClientActionsExecutor) Start(
	ctx context.Context,
	withStartAction *WithStartClientAction,
) error {
	var err error
	if withStartAction == nil {
		e.Handle, err = e.Client.ExecuteWorkflow(ctx, e.StartOptions, e.WorkflowType, e.WorkflowInput)
	} else if sig := withStartAction.GetDoSignal(); sig != nil {
		e.Handle, err = e.executeSignalAction(ctx, sig)
	} else if upd := withStartAction.GetDoUpdate(); upd != nil {
		e.Handle, err = e.executeUpdateAction(ctx, upd)
	} else {
		return fmt.Errorf("unsupported with_start_action: %v", withStartAction.String())
	}
	if err != nil {
		return fmt.Errorf("failed to start kitchen sink workflow: %w", err)
	}
	e.runID = e.Handle.GetRunID()
	return nil
}

func (e *ClientActionsExecutor) ExecuteClientSequence(ctx context.Context, clientSeq *ClientSequence) error {
	for _, actionSet := range clientSeq.ActionSets {
		if err := e.executeClientActionSet(ctx, actionSet); err != nil {
			return err
		}
	}
	return nil
}

func (e *ClientActionsExecutor) executeClientActionSet(ctx context.Context, actionSet *ClientActionSet) error {
	errs, errGroupCtx := errgroup.WithContext(ctx)
	for _, action := range actionSet.Actions {
		if actionSet.Concurrent {
			action := action
			errs.Go(func() error {
				err := e.executeClientAction(errGroupCtx, action)
				if err != nil {
					return fmt.Errorf("failed to execute concurrent client action %v: %w", action, err)
				}
				return nil
			})
		} else {
			if err := e.executeClientAction(ctx, action); err != nil {
				return fmt.Errorf("failed to execute client action %v: %w", action, err)
			}
		}
	}
	if actionSet.Concurrent {
		if err := errs.Wait(); err != nil {
			return err
		}
		if actionSet.WaitAtEnd != nil {
			select {
			case <-time.After(actionSet.WaitAtEnd.AsDuration()):
			case <-ctx.Done():
				return fmt.Errorf("context done while waiting for end %w", ctx.Err())
			}
		}
	}
	if actionSet.GetWaitForCurrentRunToFinishAtEnd() {
		err := e.Client.GetWorkflow(ctx, e.StartOptions.ID, e.runID).
			GetWithOptions(ctx, nil, client.WorkflowRunGetOptions{DisableFollowingRuns: true})
		var canErr *workflow.ContinueAsNewError
		if err != nil && !errors.As(err, &canErr) {
			return err
		}
		e.runID = e.Client.GetWorkflow(ctx, e.StartOptions.ID, "").GetRunID()
	}
	return nil
}

// Run a specific client action -
func (e *ClientActionsExecutor) executeClientAction(ctx context.Context, action *ClientAction) error {
	if action.Variant == nil {
		return fmt.Errorf("client action variant must be set")
	}

	var err error
	if sig := action.GetDoSignal(); sig != nil {
		_, err = e.executeSignalAction(ctx, sig)
		return err
	} else if update := action.GetDoUpdate(); update != nil {
		_, err = e.executeUpdateAction(ctx, update)
		return err
	} else if query := action.GetDoQuery(); query != nil {
		if query.GetReportState() != nil {
			// TODO: Use args
			_, err = e.Client.QueryWorkflow(ctx, e.StartOptions.ID, "", "report_state", nil)
		} else if handler := query.GetCustom(); handler != nil {
			_, err = e.Client.QueryWorkflow(ctx, e.StartOptions.ID, "", handler.Name, handler.Args)
		} else {
			return fmt.Errorf("do_query must recognizable variant")
		}
		if query.FailureExpected {
			err = nil
		}
		return err
	} else if action.GetNestedActions() != nil {
		err = e.executeClientActionSet(ctx, action.GetNestedActions())
		return err
	} else {
		return fmt.Errorf("client action must be set")
	}
}

func (e *ClientActionsExecutor) executeSignalAction(ctx context.Context, sig *DoSignal) (client.WorkflowRun, error) {
	var signalName string
	var signalArgs any
	if sigActions := sig.GetDoSignalActions(); sigActions != nil {
		signalName = "do_actions_signal"
		signalArgs = sigActions
	} else if handler := sig.GetCustom(); handler != nil {
		signalName = handler.Name
		signalArgs = handler.Args
	} else {
		return nil, fmt.Errorf("do_signal must recognizable variant")
	}

	if sig.WithStart {
		return e.Client.SignalWithStartWorkflow(
			ctx, e.StartOptions.ID, signalName, signalArgs, e.StartOptions, e.WorkflowType, e.WorkflowInput)
	}
	return nil, e.Client.SignalWorkflow(ctx, e.StartOptions.ID, "", signalName, signalArgs)
}

func (e *ClientActionsExecutor) executeUpdateAction(ctx context.Context, upd *DoUpdate) (run client.WorkflowRun, err error) {
	var opts client.UpdateWorkflowOptions
	if actionsUpdate := upd.GetDoActions(); actionsUpdate != nil {
		opts = client.UpdateWorkflowOptions{
			WorkflowID:   e.StartOptions.ID,
			UpdateName:   "do_actions_update",
			WaitForStage: client.WorkflowUpdateStageCompleted,
			Args:         []any{actionsUpdate},
		}
	} else if handler := upd.GetCustom(); handler != nil {
		opts = client.UpdateWorkflowOptions{
			WorkflowID:   e.StartOptions.ID,
			UpdateName:   handler.Name,
			WaitForStage: client.WorkflowUpdateStageCompleted,
			Args:         []any{handler.Args},
		}
	} else {
		return nil, fmt.Errorf("do_update must recognizable variant")
	}

	var handle client.WorkflowUpdateHandle
	if upd.WithStart {
		var updErr error
		startOpts := e.StartOptions
		startOpts.WorkflowIDConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING
		startWorkflowOp := e.Client.NewWithStartWorkflowOperation(startOpts, e.WorkflowType, e.WorkflowInput)
		handle, updErr = e.Client.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
			StartWorkflowOperation: startWorkflowOp,
			UpdateOptions:          opts,
		})
		run, err = startWorkflowOp.Get(ctx)
		if err == nil {
			err = updErr
		}
	} else {
		handle, err = e.Client.UpdateWorkflow(ctx, opts)
	}

	if err == nil {
		err = handle.Get(ctx, nil)
	}
	if upd.FailureExpected {
		err = nil
	}
	return run, err
}
