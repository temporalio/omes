package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/temporalio/omes/loadgen/journal"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/proto"
)

const startIntentMemo = "omes.original-start"

type originalStart struct {
	intent string
	action *WithStartClientAction
	update client.WorkflowUpdateHandle
}

func (e *ClientActionsExecutor) freezeStart(action *WithStartClientAction) {
	e.originalStart = &originalStart{intent: uuid.NewString()}
	e.setOriginalStartOptions()
	if upd := action.GetDoUpdate(); upd != nil && upd.UpdateId == "" {
		upd.UpdateId = uuid.NewString()
	}
	if action != nil {
		e.originalStart.action = proto.Clone(action).(*WithStartClientAction)
	}
}

func (e *ClientActionsExecutor) setOriginalStartOptions() {
	memo := make(map[string]interface{}, len(e.WorkflowOptions.Memo)+1)
	for k, v := range e.WorkflowOptions.Memo {
		memo[k] = v
	}
	memo[startIntentMemo] = e.originalStart.intent
	e.WorkflowOptions.Memo = memo
	e.WorkflowOptions.WorkflowExecutionErrorWhenAlreadyStarted = true
	e.WorkflowOptions.WorkflowIDReusePolicy = enumspb.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
	e.WorkflowOptions.WorkflowIDConflictPolicy = enumspb.WORKFLOW_ID_CONFLICT_POLICY_FAIL
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
	if runID == "" {
		runID = attrs.GetOriginalExecutionRunId()
		if runID == "" {
			return errors.New("discovered workflow history has no run ID")
		}
	}
	if first := attrs.GetFirstExecutionRunId(); first != "" && first != runID {
		return e.attachOriginalStart(ctx, first)
	}
	if attrs.GetMemo().GetFields()[startIntentMemo] == nil {
		return errors.New("existing workflow has no original start identity")
	}
	var intent string
	if err := converter.GetDefaultDataConverter().FromPayload(attrs.GetMemo().GetFields()[startIntentMemo], &intent); err != nil {
		return fmt.Errorf("reading original start identity: %w", err)
	}
	if intent != e.originalStart.intent ||
		attrs.GetWorkflowType().GetName() != e.WorkflowType ||
		attrs.GetTaskQueue().GetName() != e.WorkflowOptions.TaskQueue {
		return errors.New("existing workflow does not match the original start intent")
	}
	// Bind the history-verified run, never a later execution selected by workflow ID.
	e.Handle = e.Client.GetWorkflow(ctx, e.WorkflowOptions.ID, runID)
	return nil
}

// RestoreOriginalRun verifies the saved execution and prepares its exact update handle.
func (e *ClientActionsExecutor) RestoreOriginalRun(ctx context.Context, runID string) error {
	if e.originalStart == nil || runID == "" || e.Handle != nil {
		return errors.New("invalid original run binding")
	}
	if err := e.attachOriginalStart(ctx, runID); err != nil {
		return err
	}
	if update := e.originalStart.action.GetDoUpdate(); update != nil {
		e.originalStart.update = e.Client.GetWorkflowUpdateHandle(client.GetWorkflowUpdateHandleOptions{
			WorkflowID: e.WorkflowOptions.ID, RunID: e.Handle.GetRunID(), UpdateID: update.UpdateId,
		})
	}
	e.runID = e.Handle.GetRunID()
	return nil
}

// StartJournaled calls issue before each start or update RPC and bind before waiting for an update outcome.
func (e *ClientActionsExecutor) StartJournaled(ctx context.Context, previouslyIssued bool, issue func() error, bind func(string) error) error {
	if e.originalStart == nil {
		return errors.New("journaled start requires frozen original intent")
	}
	update := e.originalStart.action.GetDoUpdate()
	if update == nil {
		if e.Handle == nil {
			if err := issue(); err != nil {
				return err
			}
		}
		if err := e.Start(ctx, nil); err != nil {
			return err
		}
		return bind(e.Handle.GetRunID())
	}
	if err := e.bindJournaledUpdate(ctx, update, previouslyIssued, issue); err != nil {
		return err
	}
	e.runID = e.Handle.GetRunID()
	if err := bind(e.runID); err != nil {
		return err
	}
	err := e.awaitJournaledUpdate(ctx, update, issue)
	var notReady *serviceerror.WorkflowNotReady
	if errors.As(err, &notReady) {
		return e.describeRejectedJournaledUpdate(ctx, update.UpdateId, err)
	}
	var application *temporal.ApplicationError
	var canceled *temporal.CanceledError
	var timeout *temporal.TimeoutError
	if errors.As(err, &application) || errors.As(err, &canceled) || errors.As(err, &timeout) {
		if update.FailureExpected {
			return nil
		}
		return errors.Join(journal.ErrTerminal, err)
	}
	return err
}

func (e *ClientActionsExecutor) describeRejectedJournaledUpdate(ctx context.Context, updateID string, err error) error {
	probeCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	execution := &commonpb.WorkflowExecution{WorkflowId: e.WorkflowOptions.ID, RunId: e.runID}
	resp, probeErr := e.Client.WorkflowService().GetWorkflowExecutionHistoryReverse(probeCtx, &workflowservicepb.GetWorkflowExecutionHistoryReverseRequest{
		Namespace: e.Namespace, Execution: execution, MaximumPageSize: 32,
	})
	if probeErr == nil {
		for _, event := range resp.GetHistory().GetEvents() {
			if failed := event.GetWorkflowTaskFailedEventAttributes(); failed != nil {
				return fmt.Errorf("update %s on workflow %s run %s (workflow task failure event %d, cause %s): %w",
					updateID, execution.WorkflowId, execution.RunId, event.GetEventId(), failed.GetCause(), err)
			}
		}
	}
	return fmt.Errorf("update %s on workflow %s run %s: %w", updateID, execution.WorkflowId, execution.RunId, err)
}

func (e *ClientActionsExecutor) bindJournaledUpdate(ctx context.Context, update *DoUpdate, previouslyIssued bool, issue func() error) error {
	if e.Handle != nil {
		return nil
	}
	if previouslyIssued {
		err := e.attachOriginalStart(ctx, "")
		if err == nil {
			return nil
		}
		var missing *serviceerror.NotFound
		if !errors.As(err, &missing) {
			return err
		}
	}

	opts, err := e.originalUpdateOptions(update)
	if err != nil {
		return err
	}
	if err := issue(); err != nil {
		return err
	}
	op := e.Client.NewWithStartWorkflowOperation(e.WorkflowOptions, e.WorkflowType, e.WorkflowInput)
	handle, err := e.Client.UpdateWithStartWorkflow(ctx, client.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: op, UpdateOptions: opts,
	})
	if err == nil {
		if handle == nil || handle.RunID() == "" || handle.WorkflowID() != e.WorkflowOptions.ID || handle.UpdateID() != update.UpdateId {
			return errors.New("update with start returned an unexpected execution or update identity")
		}
		e.Handle = e.Client.GetWorkflow(ctx, handle.WorkflowID(), handle.RunID())
		e.originalStart.update = handle
		return nil
	}
	var already *serviceerror.WorkflowExecutionAlreadyStarted
	if errors.As(err, &already) && already.RunId != "" {
		return e.attachOriginalStart(ctx, already.RunId)
	}
	// The SDK can receive the start response before the update response is lost.
	receiptCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if receipt, receiptErr := op.Get(receiptCtx); receiptErr == nil && receipt != nil {
		return e.attachOriginalStart(ctx, receipt.GetRunID())
	}
	return fmt.Errorf("update with start: %w", err)
}

func (e *ClientActionsExecutor) awaitJournaledUpdate(ctx context.Context, update *DoUpdate, issue func() error) error {
	handle := e.originalStart.update
	if handle == nil {
		handle = e.Client.GetWorkflowUpdateHandle(client.GetWorkflowUpdateHandleOptions{
			WorkflowID: e.WorkflowOptions.ID, RunID: e.runID, UpdateID: update.UpdateId,
		})
	}
	err := handle.Get(ctx, nil)
	var missing *serviceerror.NotFound
	if !errors.As(err, &missing) {
		return err
	}
	// A missing update can only be submitted while the verified original is open.
	description, describeErr := e.Client.DescribeWorkflowExecution(ctx, e.WorkflowOptions.ID, e.runID)
	if describeErr != nil {
		return errors.Join(err, describeErr)
	}
	if description.GetWorkflowExecutionInfo().GetStatus() != enumspb.WORKFLOW_EXECUTION_STATUS_RUNNING {
		return err
	}
	opts, err := e.originalUpdateOptions(update)
	if err != nil {
		return err
	}
	opts.RunID = e.runID
	if err := issue(); err != nil {
		return err
	}
	handle, err = e.Client.UpdateWorkflow(ctx, opts)
	if err != nil {
		return err
	}
	e.originalStart.update = handle
	return handle.Get(ctx, nil)
}

func (e *ClientActionsExecutor) originalUpdateOptions(update *DoUpdate) (client.UpdateWorkflowOptions, error) {
	name, args, err := UpdateNameAndArgs(update)
	if err != nil {
		return client.UpdateWorkflowOptions{}, err
	}
	return client.UpdateWorkflowOptions{
		WorkflowID: e.WorkflowOptions.ID, UpdateID: update.UpdateId,
		UpdateName: name, Args: args, WaitForStage: client.WorkflowUpdateStageAccepted,
	}, nil
}
