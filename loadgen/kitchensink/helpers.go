package kitchensink

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"
	"strings"
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
	println("!!!!!!!! Number of action sets ", len(clientSeq.ActionSets))
	for _, actionSet := range clientSeq.ActionSets {
		if err := e.executeClientActionSet(ctx, actionSet); err != nil {
			return err
		}
		println("!!!!!!!!! Done")
	}

	return nil
}

func (e *ClientActionsExecutor) executeClientActionSet(ctx context.Context, actionSet *ClientActionSet) error {
	println("!!!!!!!! Running action set is concurrent: ", actionSet.Concurrent)
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
			println("WAITING AT END", actionSet.WaitAtEnd.Seconds, " n ", actionSet.WaitAtEnd.Nanos)
			select {
			case <-time.After(actionSet.WaitAtEnd.AsDuration()):
				println("Done waiting")
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

	println("Running client action: ", proto.MarshalTextString(action))

	var err error
	if sig := action.GetDoSignal(); sig != nil {
		if actionSet := sig.GetDoActions(); actionSet != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, "do_actions_signal", actionSet)
		} else if handler := sig.GetCustom(); handler != nil {
			err = e.Client.SignalWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
		} else {
			return fmt.Errorf("do_signal must recognizable variant")
		}
	} else if update := action.GetDoUpdate(); update != nil {
		if actionSet := update.GetDoActions(); actionSet != nil {
			_, err = e.Client.UpdateWorkflow(ctx, e.WorkflowID, e.RunID, "do_actions_update", actionSet)
		} else if update.GetRejectMe() != nil {
			_, err = e.Client.UpdateWorkflow(ctx, e.WorkflowID, e.RunID, "always_reject", actionSet)
		} else if handler := update.GetCustom(); handler != nil {
			_, err = e.Client.UpdateWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
			err = clearErrorIfExpectedNotFound(handler, err)
		} else {
			return fmt.Errorf("do_update must recognizable variant")
		}
	} else if query := action.GetDoQuery(); query != nil {
		if query.GetReportState() != nil {
			// TODO: Use args
			_, err = e.Client.QueryWorkflow(ctx, e.WorkflowID, e.RunID, "report_state", nil)
		} else if handler := query.GetCustom(); handler != nil {
			_, err = e.Client.QueryWorkflow(ctx, e.WorkflowID, e.RunID, handler.Name, handler.Args)
			err = clearErrorIfExpectedNotFound(handler, err)
		} else {
			return fmt.Errorf("do_query must recognizable variant")
		}
	} else if action.GetNestedActions() != nil {
		err = e.executeClientActionSet(ctx, action.GetNestedActions())
	}
	println("!!!!!! CLIENT ACTION DONE")

	return err
}

func clearErrorIfExpectedNotFound(handler *HandlerInvocation, err error) error {
	if err != nil && handler.Name == nonexistentHandler {
		if strings.Contains(err.Error(), "not found") {
			err = nil
		}
	}
	return err
}
