package kitchensink

import (
	"context"
	"fmt"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"
	"time"
)

// Must match string used in rust generator
const nonexistentHandler = "nonexistent handler on purpose"

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

type ClientActionsExecutor struct {
	Client     client.Client
	WorkflowID string
	RunID      string
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
					return fmt.Errorf("failed to execute client action %v: %w", action, err)
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
	return nil
}

// Run a specific client action -
func (e *ClientActionsExecutor) executeClientAction(ctx context.Context, action *ClientAction) error {
	if action.Variant == nil {
		return fmt.Errorf("client action variant must be set")
	}

	var err error
	if sig := action.GetDoSignal(); sig != nil {
		if sigActions := sig.GetDoSignalActions(); sigActions != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, "do_actions_signal", sigActions)
		} else if handler := sig.GetCustom(); handler != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
		} else {
			return fmt.Errorf("do_signal must recognizable variant")
		}
		return err
	} else if update := action.GetDoUpdate(); update != nil {
		if actionsUpdate := update.GetDoActions(); actionsUpdate != nil {
			_, err = e.Client.UpdateWorkflow(ctx, e.WorkflowID, e.RunID, "do_actions_update", actionsUpdate)
		} else if handler := update.GetCustom(); handler != nil {
			_, err = e.Client.UpdateWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
		} else {
			return fmt.Errorf("do_update must recognizable variant")
		}
		if update.FailureExpected {
			err = nil
		}
		return err
	} else if query := action.GetDoQuery(); query != nil {
		if query.GetReportState() != nil {
			// TODO: Use args
			_, err = e.Client.QueryWorkflow(ctx, e.WorkflowID, e.RunID, "report_state", nil)
		} else if handler := query.GetCustom(); handler != nil {
			_, err = e.Client.QueryWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
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
