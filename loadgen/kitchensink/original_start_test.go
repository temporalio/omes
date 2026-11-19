package kitchensink

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/devserver"
	"github.com/temporalio/omes/internal/versions"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type responseLossClient struct {
	client.Client
	starts      int
	multis      int
	original    client.WorkflowRun
	failMulti   bool
	failUpdate  bool
	updateError error
	loseMulti   bool
	loseOutcome bool
	updateID    string
}

func (c *responseLossClient) ExecuteWorkflow(ctx context.Context, opts client.StartWorkflowOptions, wf interface{}, args ...interface{}) (client.WorkflowRun, error) {
	c.starts++
	h, err := c.Client.ExecuteWorkflow(ctx, opts, wf, args...)
	if err == nil && c.starts == 1 {
		c.original = h
		return nil, serviceerror.NewDeadlineExceeded("lost accepted start reply")
	}
	return h, err
}
func (c *responseLossClient) UpdateWithStartWorkflow(ctx context.Context, opts client.UpdateWithStartWorkflowOptions) (client.WorkflowUpdateHandle, error) {
	c.multis++
	c.updateID = opts.UpdateOptions.UpdateID
	if c.failMulti && c.multis == 1 {
		return nil, serviceerror.NewUnavailable("start request not sent")
	}
	h, err := c.Client.UpdateWithStartWorkflow(ctx, opts)
	if err != nil {
		return nil, err
	}
	if c.loseMulti && c.multis == 1 {
		return nil, serviceerror.NewDeadlineExceeded("lost multi-operation reply")
	}
	if c.loseOutcome {
		return &outcomeLossHandle{WorkflowUpdateHandle: h}, nil
	}
	return h, nil
}

func (c *responseLossClient) UpdateWorkflow(ctx context.Context, opts client.UpdateWorkflowOptions) (client.WorkflowUpdateHandle, error) {
	if c.failUpdate {
		c.failUpdate = false
		if c.updateError != nil {
			return nil, c.updateError
		}
		return nil, serviceerror.NewUnavailable("update request not sent")
	}
	return c.Client.UpdateWorkflow(ctx, opts)
}

type outcomeLossHandle struct {
	client.WorkflowUpdateHandle
	calls atomic.Int32
}

func (h *outcomeLossHandle) Get(ctx context.Context, v interface{}) error {
	if h.calls.Add(1) == 1 {
		return serviceerror.NewDeadlineExceeded("lost update outcome reply")
	}
	return h.WorkflowUpdateHandle.Get(ctx, v)
}

func originalStartWorkflow(ctx workflow.Context, input *WorkflowInput) error {
	updates := 0
	if err := workflow.SetQueryHandler(ctx, "updates", func() (int, error) { return updates, nil }); err != nil {
		return err
	}
	if err := workflow.SetUpdateHandler(ctx, "do_actions_update", func(ctx workflow.Context, _ *DoActionsUpdate) error {
		updates++
		if input.ExpectedSignalCount == 42 {
			return temporal.NewApplicationError("update failed", "UpdateFailure")
		}
		return workflow.Sleep(ctx, 50*time.Millisecond)
	}); err != nil {
		return err
	}
	workflow.GetSignalChannel(ctx, "finish").Receive(ctx, nil)
	if input.ExpectedSignalCount == 73 {
		return workflow.NewContinueAsNewError(ctx, "kitchenSink", &WorkflowInput{ExpectedSignalCount: 74})
	}
	if input.ExpectedSignalCount < 0 {
		return temporal.NewNonRetryableApplicationError("original target failure", "TargetFailure", nil)
	}
	return nil
}

func TestOriginalStartProtocol(t *testing.T) {
	ref, err := versions.Get("SERVER_VERSION")
	require.NoError(t, err)
	server, err := devserver.Start(t.Context(), devserver.Options{Ref: ref})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	c, err := client.DialContext(ctx, client.Options{HostPort: server.FrontendHostPort(), Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	queue := "original-start-" + uuid.NewString()
	w := worker.New(c, queue, worker.Options{})
	w.RegisterWorkflowWithOptions(originalStartWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	require.NoError(t, w.Start())
	defer w.Stop()
	newExecutor := func(cl client.Client) *ClientActionsExecutor {
		e := &ClientActionsExecutor{
			Client:        cl,
			Namespace:     "default",
			WorkflowType:  "kitchenSink",
			WorkflowInput: &WorkflowInput{},
			WorkflowOptions: client.StartWorkflowOptions{
				ID:                                       uuid.NewString(),
				TaskQueue:                                queue,
				WorkflowExecutionErrorWhenAlreadyStarted: true,
			},
		}
		e.RecoverOriginalStart = true
		return e
	}
	t.Run("restore_saved_or_unacknowledged_binding_after_continue_as_new", func(t *testing.T) {
		original := newExecutor(c)
		original.WorkflowInput.ExpectedSignalCount = 73
		spec, err := original.PrepareOriginalStart(nil)
		require.NoError(t, err)
		require.NoError(t, original.Start(ctx, nil))
		firstRun := original.Handle.GetRunID()
		require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, firstRun, "finish", nil))
		require.Eventually(t, func() bool {
			description, err := c.DescribeWorkflowExecution(ctx, spec.WorkflowID, "")
			return err == nil && description.WorkflowExecutionInfo.Execution.RunId != firstRun
		}, 5*time.Second, 10*time.Millisecond)
		for _, saved := range []bool{true, false} {
			restored := newExecutor(c)
			restored.WorkflowOptions = original.WorkflowOptions
			require.NoError(t, restored.RestoreOriginalStart(spec))
			if saved {
				require.NoError(t, restored.RestoreOriginalRun(ctx, firstRun))
			}
			require.NoError(t, restored.Start(ctx, nil))
			require.Equal(t, firstRun, restored.Handle.GetRunID())
		}
		require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, "", "finish", nil))
		require.NoError(t, original.Handle.Get(ctx, nil))
	})
	t.Run("restore_completed_update_with_start_without_resending", func(t *testing.T) {
		original := newExecutor(c)
		action := &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}}}}}
		spec, err := original.PrepareOriginalStart(action)
		require.NoError(t, err)
		require.NoError(t, original.Start(ctx, nil))
		firstRun := original.Handle.GetRunID()
		require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, firstRun, "finish", nil))
		require.NoError(t, original.Handle.Get(ctx, nil))
		transport := &responseLossClient{Client: c, failUpdate: true}
		restored := newExecutor(transport)
		restored.WorkflowOptions = original.WorkflowOptions
		require.NoError(t, restored.RestoreOriginalStart(spec))
		require.NoError(t, restored.RestoreOriginalRun(ctx, firstRun))
		require.NoError(t, restored.Start(ctx, nil))
		require.NoError(t, restored.Handle.Get(ctx, nil))
		require.Zero(t, transport.starts)
		require.Zero(t, transport.multis)
		require.True(t, transport.failUpdate)
	})
	t.Run("journal binds before polling and restores a closed accepted update", func(t *testing.T) {
		transport := &responseLossClient{Client: c, loseOutcome: true}
		original := newExecutor(transport)
		action := &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}}}}}
		spec, err := original.PrepareOriginalStart(action)
		require.NoError(t, err)
		var bound string
		issues := 0
		issue := func() error { issues++; return nil }
		err = original.StartJournaled(ctx, false, issue, func(id string) error {
			bound = id
			require.Zero(t, original.originalStart.update.(*outcomeLossHandle).calls.Load())
			return nil
		})
		require.ErrorContains(t, err, "lost update outcome reply")
		require.NotEmpty(t, bound)
		require.NoError(t, c.GetWorkflowUpdateHandle(client.GetWorkflowUpdateHandleOptions{
			WorkflowID: spec.WorkflowID, RunID: bound, UpdateID: transport.updateID,
		}).Get(ctx, nil))
		require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, bound, "finish", nil))
		require.NoError(t, c.GetWorkflow(ctx, spec.WorkflowID, bound).Get(ctx, nil))

		restored := newExecutor(transport)
		restored.WorkflowOptions = original.WorkflowOptions
		require.NoError(t, restored.RestoreOriginalStart(spec))
		require.NoError(t, restored.RestoreOriginalRun(ctx, bound))
		require.NoError(t, restored.StartJournaled(ctx, true, issue, func(id string) error {
			require.Equal(t, bound, id)
			return nil
		}))
		require.Equal(t, 1, issues)
		require.Equal(t, 1, transport.multis)
	})
	t.Run("journal recovers the start receipt when the combined reply is lost", func(t *testing.T) {
		transport := &responseLossClient{Client: c, loseMulti: true}
		e := newExecutor(transport)
		action := &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}}}}}
		spec, err := e.PrepareOriginalStart(action)
		require.NoError(t, err)
		issues := 0
		var bound string
		require.NoError(t, e.StartJournaled(ctx, false, func() error {
			issues++
			return nil
		}, func(id string) error {
			bound = id
			return nil
		}))
		require.NotEmpty(t, bound)
		require.Equal(t, 1, issues)
		require.Equal(t, 1, transport.multis)
		q, err := c.QueryWorkflow(ctx, spec.WorkflowID, bound, "updates")
		require.NoError(t, err)
		var applied int
		require.NoError(t, q.Get(&applied))
		require.Equal(t, 1, applied)
		require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, bound, "finish", nil))
		require.NoError(t, c.GetWorkflow(ctx, spec.WorkflowID, bound).Get(ctx, nil))
	})
	t.Run("journal discovers the original when the update was never accepted", func(t *testing.T) {
		for _, closed := range []bool{false, true} {
			name := "open"
			if closed {
				name = "closed"
			}
			t.Run(name, func(t *testing.T) {
				transport := &responseLossClient{Client: c, failUpdate: true, updateError: serviceerror.NewWorkflowNotReady("Workflow Task failed")}
				original := newExecutor(transport)
				action := &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}}}}}
				spec, err := original.PrepareOriginalStart(action)
				require.NoError(t, err)
				// The start ran but the update never arrived and the journal has no start receipt.
				target, err := c.ExecuteWorkflow(ctx, original.WorkflowOptions, original.WorkflowType, original.WorkflowInput)
				require.NoError(t, err)
				finish := func() {
					require.NoError(t, c.SignalWorkflow(ctx, spec.WorkflowID, target.GetRunID(), "finish", nil))
					require.NoError(t, target.Get(ctx, nil))
				}
				if closed {
					finish()
				}
				issues := 0
				issue := func() error { issues++; return nil }
				bind := func(id string) error {
					require.Equal(t, target.GetRunID(), id)
					return nil
				}
				err = original.StartJournaled(ctx, true, issue, bind)
				if closed {
					var notFound *serviceerror.NotFound
					require.ErrorAs(t, err, &notFound)
					require.Zero(t, issues)
				} else {
					var notReady *serviceerror.WorkflowNotReady
					require.ErrorAs(t, err, &notReady)
					require.NoError(t, original.StartJournaled(ctx, true, issue, bind))
					require.Equal(t, 2, issues)
					q, err := c.QueryWorkflow(ctx, spec.WorkflowID, target.GetRunID(), "updates")
					require.NoError(t, err)
					var applied int
					require.NoError(t, q.Get(&applied))
					require.Equal(t, 1, applied)
					finish()
				}
				require.Zero(t, transport.multis)
			})
		}
	})
	t.Run("late_self_update_cannot_replace_or_update_foreign_run", func(t *testing.T) {
		id := uuid.NewString()
		original, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: id, TaskQueue: queue}, "kitchenSink", &WorkflowInput{})
		require.NoError(t, err)
		require.NoError(t, c.SignalWorkflow(ctx, id, original.GetRunID(), "finish", nil))
		require.NoError(t, original.Get(ctx, nil))
		e := &ClientActionsExecutor{
			Client:          c,
			WorkflowOptions: client.StartWorkflowOptions{ID: id},
			Execution:       &ClientExecution{RunID: original.GetRunID(), UpdateIDScope: "original/activity-1"},
		}
		seq := ClientActions(selfUpdate())
		require.Error(t, e.ExecuteClientSequence(ctx, seq))
		foreign, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
			ID: id, TaskQueue: queue, WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		}, "kitchenSink", &WorkflowInput{})
		require.NoError(t, err)
		require.Error(t, e.ExecuteClientSequence(ctx, seq))
		value, err := c.QueryWorkflow(ctx, id, foreign.GetRunID(), "updates")
		require.NoError(t, err)
		var updates int
		require.NoError(t, value.Get(&updates))
		require.Zero(t, updates)
		require.NoError(t, c.SignalWorkflow(ctx, id, foreign.GetRunID(), "finish", nil))
		require.NoError(t, foreign.Get(ctx, nil))
	})
	for _, state := range []string{"running", "completed", "failed"} {
		t.Run("lost_response_"+state, func(t *testing.T) {
			lost := &responseLossClient{Client: c}
			e := newExecutor(lost)
			if state == "failed" {
				e.WorkflowInput.ExpectedSignalCount = -1
			}
			require.Error(t, e.Start(ctx, nil))
			original := lost.original
			require.NotNil(t, original)
			if state != "running" {
				require.NoError(t, c.SignalWorkflow(ctx, original.GetID(), original.GetRunID(), "finish", nil))
				getErr := original.Get(ctx, nil)
				if state == "failed" {
					require.Error(t, getErr)
				} else {
					require.NoError(t, getErr)
				}
			}
			require.NoError(t, e.Start(ctx, nil))
			require.Equal(t, original.GetRunID(), e.Handle.GetRunID(), "must never replace the original closed execution")
			if state == "running" {
				require.NoError(t, c.SignalWorkflow(ctx, original.GetID(), original.GetRunID(), "finish", nil))
			}
			getErr := e.Handle.Get(ctx, nil)
			if state == "failed" {
				var failure *temporal.ApplicationError
				require.ErrorAs(t, getErr, &failure)
				require.Equal(t, "TargetFailure", failure.Type())
				require.NoError(t, e.Start(ctx, nil))
				require.Error(t, e.Handle.Get(ctx, nil))
			} else {
				require.NoError(t, getErr)
			}
		})
	}
	t.Run("different_input_converter", func(t *testing.T) {
		binaryClient, err := client.DialContext(ctx, client.Options{
			HostPort:  server.FrontendHostPort(),
			Namespace: "default",
			DataConverter: converter.NewCompositeDataConverter(
				converter.NewProtoPayloadConverter(),
				converter.NewJSONPayloadConverter(),
			),
		})
		require.NoError(t, err)
		defer binaryClient.Close()
		lost := &responseLossClient{Client: binaryClient}
		e := newExecutor(lost)
		require.Error(t, e.Start(ctx, nil))
		require.NoError(t, e.Start(ctx, nil))
		require.Equal(t, lost.original.GetRunID(), e.Handle.GetRunID())
		require.NoError(t, c.SignalWorkflow(ctx, e.Handle.GetID(), e.Handle.GetRunID(), "finish", nil))
		require.NoError(t, e.Handle.Get(ctx, nil))
	})

	t.Run("foreign_reuse", func(t *testing.T) {
		lost := &responseLossClient{Client: c}
		e := newExecutor(lost)
		require.Error(t, e.Start(ctx, nil))
		original := lost.original
		require.NoError(t, c.SignalWorkflow(ctx, original.GetID(), original.GetRunID(), "finish", nil))
		require.NoError(t, original.Get(ctx, nil))
		opts := e.WorkflowOptions
		opts.Memo = map[string]interface{}{startIntentMemo: uuid.NewString()}
		opts.WorkflowIDReusePolicy = enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE
		foreign, err := c.ExecuteWorkflow(ctx, opts, "kitchenSink", &WorkflowInput{ExpectedSignalCount: 99})
		require.NoError(t, err)
		require.Error(t, e.Start(ctx, nil))
		require.Nil(t, e.Handle)
		require.NoError(t, c.SignalWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "finish", nil))
		require.NoError(t, foreign.Get(ctx, nil))
	})
	t.Run("foreign_existing", func(t *testing.T) {
		e := newExecutor(c)
		foreign, err := c.ExecuteWorkflow(ctx, e.WorkflowOptions, "kitchenSink", e.WorkflowInput)
		require.NoError(t, err)
		require.Error(t, e.Start(ctx, nil))
		require.Nil(t, e.Handle)
		require.NoError(t, c.SignalWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "finish", nil))
		require.NoError(t, foreign.Get(ctx, nil))
	})
	t.Run("canceled", func(t *testing.T) {
		e := newExecutor(c)
		canceled, stop := context.WithCancel(ctx)
		stop()
		require.Error(t, e.Start(canceled, nil))
		require.Nil(t, e.Handle)
	})
	action := func() *WithStartClientAction {
		return &WithStartClientAction{
			Variant: &WithStartClientAction_DoUpdate{
				DoUpdate: &DoUpdate{
					WithStart: true,
					Variant: &DoUpdate_DoActions{
						DoActions: &DoActionsUpdate{
							Variant: &DoActionsUpdate_DoActions{},
						},
					},
				},
			},
		}
	}
	for _, failure := range []string{"none", "update_unavailable"} {
		t.Run("uws_already_started_"+failure, func(t *testing.T) {
			observed := &responseLossClient{Client: c, failUpdate: failure == "update_unavailable"}
			e := newExecutor(observed)
			a := action()
			e.freezeStart(a)
			original, err := c.ExecuteWorkflow(ctx, e.WorkflowOptions, "kitchenSink", e.WorkflowInput)
			require.NoError(t, err)
			if failure == "update_unavailable" {
				require.Error(t, e.Start(ctx, a))
			}
			require.NoError(t, e.Start(ctx, a))
			require.Equal(t, original.GetRunID(), e.Handle.GetRunID())
			require.NoError(t, e.Start(ctx, a))
			value, err := c.QueryWorkflow(ctx, original.GetID(), original.GetRunID(), "updates")
			require.NoError(t, err)
			var updates int
			require.NoError(t, value.Get(&updates))
			require.Equal(t, 1, updates)
			require.NoError(t, c.SignalWorkflow(ctx, original.GetID(), original.GetRunID(), "finish", nil))
			require.NoError(t, original.Get(ctx, nil))
		})
	}

	t.Run("uws_confirmed_update_outcome", func(t *testing.T) {
		lost := &responseLossClient{Client: c, loseOutcome: true}
		e := newExecutor(lost)
		a := action()
		require.Error(t, e.Start(ctx, a))
		require.NoError(t, e.Start(ctx, a))
		require.Equal(t, 1, lost.multis)
		require.NotEmpty(t, a.GetDoUpdate().GetUpdateId())
		require.Equal(t, lost.updateID, a.GetDoUpdate().GetUpdateId())
		require.NoError(t, c.SignalWorkflow(ctx, e.Handle.GetID(), e.Handle.GetRunID(), "finish", nil))
		require.NoError(t, e.Handle.Get(ctx, nil))
	})
	for _, failure := range []string{"before_send", "lost_reply"} {
		t.Run("uws_retry_"+failure, func(t *testing.T) {
			lost := &responseLossClient{
				Client:    c,
				failMulti: failure == "before_send",
				loseMulti: failure == "lost_reply",
			}
			e := newExecutor(lost)
			a := action()
			require.Error(t, e.Start(ctx, a))
			updateID := lost.updateID
			require.NotEmpty(t, updateID)
			require.NoError(t, e.Start(ctx, a))
			require.Equal(t, 2, lost.multis)
			require.Equal(t, updateID, lost.updateID)
			value, err := c.QueryWorkflow(ctx, e.Handle.GetID(), e.Handle.GetRunID(), "updates")
			require.NoError(t, err)
			var updates int
			require.NoError(t, value.Get(&updates))
			require.Equal(t, 1, updates, "retry must not execute the update twice")
			canceled, stop := context.WithCancel(ctx)
			stop()
			require.ErrorIs(t, e.Start(canceled, a), context.Canceled)
			require.NoError(t, c.SignalWorkflow(ctx, e.Handle.GetID(), e.Handle.GetRunID(), "finish", nil))
			require.NoError(t, e.Handle.Get(ctx, nil))
		})
	}

	t.Run("uws_closed_target_failure_is_not_replaced", func(t *testing.T) {
		observed := &responseLossClient{Client: c}
		e := newExecutor(observed)
		e.WorkflowInput.ExpectedSignalCount = -1
		a := action()
		require.NoError(t, e.Start(ctx, a))
		original := e.Handle.GetRunID()
		require.NoError(t, c.SignalWorkflow(ctx, e.Handle.GetID(), original, "finish", nil))
		require.Error(t, e.Handle.Get(ctx, nil))
		require.NoError(t, e.Start(ctx, a))
		require.Equal(t, original, e.Handle.GetRunID())
		require.Equal(t, 1, observed.multis)
		require.Error(t, e.Handle.Get(ctx, nil))
	})
	t.Run("uws_failed_update_outcome_is_preserved", func(t *testing.T) {
		observed := &responseLossClient{Client: c}
		e := newExecutor(observed)
		e.WorkflowInput.ExpectedSignalCount = 42
		a := action()
		require.Error(t, e.Start(ctx, a))
		require.Error(t, e.Start(ctx, a))
		require.Equal(t, 1, observed.multis)
		require.NoError(t, c.SignalWorkflow(ctx, e.WorkflowOptions.ID, "", "finish", nil))
		require.NoError(t, c.GetWorkflow(ctx, e.WorkflowOptions.ID, "").Get(ctx, nil))
	})
	t.Run("uws_foreign_closed_failed_is_not_replaced", func(t *testing.T) {
		e := newExecutor(c)
		e.WorkflowInput.ExpectedSignalCount = -1
		foreign, err := c.ExecuteWorkflow(ctx, e.WorkflowOptions, "kitchenSink", e.WorkflowInput)
		require.NoError(t, err)
		require.NoError(t, c.SignalWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "finish", nil))
		require.Error(t, foreign.Get(ctx, nil))
		require.Error(t, e.Start(ctx, action()))
		require.Nil(t, e.Handle)
	})
	t.Run("uws_wire_reply_loss_and_external_reuse", func(t *testing.T) {
		var lost atomic.Bool
		var mu sync.Mutex
		requests := map[string]bool{}
		foreignRuns := make(chan client.WorkflowRun, 1)
		intercept := func(callCtx context.Context, method string, req, reply interface{}, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			if !strings.HasSuffix(method, "/ExecuteMultiOperation") {
				return invoke(callCtx, method, req, reply, conn, opts...)
			}
			request := req.(*workflowservice.ExecuteMultiOperationRequest)
			start := request.GetOperations()[0].GetStartWorkflow()
			mu.Lock()
			requests[start.GetRequestId()] = true
			mu.Unlock()
			err := invoke(callCtx, method, req, reply, conn, opts...)
			if err == nil && !lost.Swap(true) {
				originalRun := reply.(*workflowservice.ExecuteMultiOperationResponse).GetResponses()[0].GetStartWorkflow().GetRunId()
				if err := c.SignalWorkflow(ctx, start.GetWorkflowId(), originalRun, "finish", nil); err != nil {
					return err
				}
				if err := c.GetWorkflow(ctx, start.GetWorkflowId(), originalRun).Get(ctx, nil); err != nil {
					return err
				}
				foreign, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
					ID:                    start.GetWorkflowId(),
					TaskQueue:             queue,
					WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
				}, "kitchenSink", &WorkflowInput{ExpectedSignalCount: 99})
				if err != nil {
					return err
				}
				foreignRuns <- foreign
				return status.Error(codes.DeadlineExceeded, "lost real multi-operation reply after external ID reuse")
			}
			return err
		}
		wire, err := client.Dial(client.Options{
			HostPort:  server.FrontendHostPort(),
			Namespace: "default",
			ConnectionOptions: client.ConnectionOptions{
				DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(intercept)},
			},
		})
		require.NoError(t, err)
		defer wire.Close()
		e := newExecutor(wire)
		a := action()
		firstErr := e.Start(ctx, a)
		secondErr := e.Start(ctx, a)
		require.True(t, lost.Load(), "real successful multi-operation response must have been discarded")
		var foreign client.WorkflowRun
		select {
		case foreign = <-foreignRuns:
		case <-time.After(time.Second):
			t.Fatal("external reuse was not established")
		}
		value, err := c.QueryWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "updates")
		require.NoError(t, err)
		var updates int
		require.NoError(t, value.Get(&updates))
		require.Zero(t, updates, "recovery must not update a foreign replacement run")
		mu.Lock()
		requestCount := len(requests)
		mu.Unlock()
		require.Equal(t, 2, requestCount, "each outer attempt sends one multi-operation request")
		require.Error(t, firstErr)
		require.Error(t, secondErr)
		require.NoError(t, c.SignalWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "finish", nil))
		require.NoError(t, foreign.Get(ctx, nil))
	})

	t.Run("uws_foreign_is_not_updated", func(t *testing.T) {
		e := newExecutor(c)
		foreign, err := c.ExecuteWorkflow(ctx, e.WorkflowOptions, "kitchenSink", e.WorkflowInput)
		require.NoError(t, err)
		err = e.Start(ctx, action())
		require.Error(t, err)
		require.Nil(t, e.Handle)
		require.NoError(t, c.SignalWorkflow(ctx, foreign.GetID(), foreign.GetRunID(), "finish", nil))
		require.NoError(t, foreign.Get(ctx, nil))
	})
}
