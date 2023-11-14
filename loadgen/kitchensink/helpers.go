package kitchensink

import (
	"context"
	"fmt"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"
)

func NoOpSingleActivityActionSet() *ActionSet {
	return &ActionSet{
		Actions: []*Action{
			{
				Variant: &Action_ExecActivity{
					ExecActivity: &ExecuteActivityAction{
						ActivityType:        "noop",
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
	errs, ctx := errgroup.WithContext(ctx)
	for _, action := range actionSet.Actions {
		if actionSet.Concurrent {
			errs.Go(func() error { return e.executeClientAction(ctx, action) })
		} else {
			if err := e.executeClientAction(ctx, action); err != nil {
				return err
			}
		}
	}
	return errs.Wait()
}

// Run a specific client action -
func (e *ClientActionsExecutor) executeClientAction(ctx context.Context, action *ClientAction) error {
	if action.Variant == nil {
		return fmt.Errorf("client action variant must be set")
	}

	var err error
	if sig := action.GetDoSignal(); sig != nil {
		if actionSet := sig.GetDoActions(); actionSet != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, "do_actions_signal", actionSet)
		} else if handler := sig.GetCustom(); handler != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
		} else {
			return fmt.Errorf("do_signal must recognizable variant")
		}
	} else if action.GetDoUpdate() != nil {
		panic("todo")
	} else if action.GetDoQuery() != nil {
		panic("todo")
	} else if action.GetNestedActions() != nil {
		err = e.executeClientActionSet(ctx, action.GetNestedActions())
	}

	return err
}
