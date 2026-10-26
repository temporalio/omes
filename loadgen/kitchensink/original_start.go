package kitchensink

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
)

const startIntentMemo = "omes.original-start"

type originalStart struct {
	intent       string
	action       *WithStartClientAction
	update       client.WorkflowUpdateHandle
	withStartErr error
}

func (e *ClientActionsExecutor) freezeStart(action *WithStartClientAction) {
	e.originalStart = &originalStart{intent: uuid.NewString()}
	memo := make(map[string]interface{}, len(e.WorkflowOptions.Memo)+1)
	for k, v := range e.WorkflowOptions.Memo {
		memo[k] = v
	}
	memo[startIntentMemo] = e.originalStart.intent
	e.WorkflowOptions.Memo = memo
	e.WorkflowOptions.WorkflowExecutionErrorWhenAlreadyStarted = true
	e.WorkflowOptions.WorkflowIDReusePolicy = enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
	e.WorkflowOptions.WorkflowIDConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL
	if upd := action.GetDoUpdate(); upd != nil && upd.UpdateId == "" {
		upd.UpdateId = uuid.NewString()
	}
	if action != nil {
		e.originalStart.action = proto.Clone(action).(*WithStartClientAction)
	}
}

func (e *ClientActionsExecutor) attachOriginalStart(ctx context.Context, runID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	history := e.Client.GetWorkflowHistory(ctx, e.WorkflowOptions.ID, runID, false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	event, err := history.Next()
	if err != nil {
		return err
	}
	attrs := event.GetWorkflowExecutionStartedEventAttributes()
	if attrs == nil {
		return errors.New("original start history is missing its start event")
	}
	if attrs.GetMemo().GetFields()[startIntentMemo] == nil {
		return errors.New("existing workflow has no original start identity")
	}
	var intent string
	if err := converter.GetDefaultDataConverter().FromPayload(attrs.GetMemo().GetFields()[startIntentMemo], &intent); err != nil {
		return fmt.Errorf("reading original start identity: %w", err)
	}
	input, err := converter.GetDefaultDataConverter().ToPayloads(e.WorkflowInput)
	if err != nil {
		return err
	}
	if intent != e.originalStart.intent || attrs.GetWorkflowType().GetName() != e.WorkflowType || attrs.GetTaskQueue().GetName() != e.WorkflowOptions.TaskQueue || !proto.Equal(attrs.GetInput(), input) {
		return errors.New("existing workflow does not match the original start intent")
	}
	// Bind the history-verified run, never a later execution selected by workflow ID.
	e.Handle = e.Client.GetWorkflow(ctx, e.WorkflowOptions.ID, runID)
	return nil
}
