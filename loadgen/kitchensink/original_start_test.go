package kitchensink

import (
	"context"
	"errors"
	"os"
	"reflect"
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
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Optional by reflection so the same regression executes against the base implementation.
func enableOriginalStart(executor *ClientActionsExecutor) {
	if field := reflect.ValueOf(executor).Elem().FieldByName("RecoverOriginalStart"); field.IsValid() {
		field.SetBool(true)
	}
}

type responseLossClient struct {
	client.Client
	starts      int
	multis      int
	original    client.WorkflowRun
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
	h, err := c.Client.UpdateWithStartWorkflow(ctx, opts)
	if err != nil {
		return nil, err
	}
	if c.loseMulti {
		return nil, serviceerror.NewDeadlineExceeded("lost multi-operation reply")
	}
	if c.loseOutcome {
		return &outcomeLossHandle{WorkflowUpdateHandle: h}, nil
	}
	return h, nil
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
	if input.ExpectedSignalCount < 0 {
		return temporal.NewNonRetryableApplicationError("original target failure", "TargetFailure", nil)
	}
	return nil
}

func TestOriginalStartProtocol(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	ref, err := versions.Get("SERVER_VERSION")
	require.NoError(t, err)
	options := devserver.Options{Ref: ref, OutputDir: os.Getenv("OMES_START_TEST_SERVER_CACHE")}
	if source := os.Getenv("OMES_START_TEST_SERVER_SOURCE"); source != "" {
		options.Ref = ""
		options.SourceDir = source
	}
	server, err := devserver.Start(ctx, options)
	require.NoError(t, err)
	defer func() { require.NoError(t, server.Stop()) }()
	c, err := client.Dial(client.Options{HostPort: server.FrontendHostPort(), Namespace: "default"})
	require.NoError(t, err)
	defer c.Close()
	queue := "original-start-" + uuid.NewString()
	w := worker.New(c, queue, worker.Options{})
	w.RegisterWorkflowWithOptions(originalStartWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	require.NoError(t, w.Start())
	defer w.Stop()
	newExecutor := func(cl client.Client) *ClientActionsExecutor {
		e := &ClientActionsExecutor{Client: cl, Namespace: "default", WorkflowType: "kitchenSink", WorkflowInput: &WorkflowInput{}, WorkflowOptions: client.StartWorkflowOptions{ID: uuid.NewString(), TaskQueue: queue, WorkflowExecutionErrorWhenAlreadyStarted: true}}
		enableOriginalStart(e)
		return e
	}
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
	t.Run("foreign_reuse_and_mismatched_input", func(t *testing.T) {
		lost := &responseLossClient{Client: c}
		e := newExecutor(lost)
		require.Error(t, e.Start(ctx, nil))
		original := lost.original
		require.NoError(t, c.SignalWorkflow(ctx, original.GetID(), original.GetRunID(), "finish", nil))
		require.NoError(t, original.Get(ctx, nil))
		opts := e.WorkflowOptions
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
		return &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{Variant: &DoActionsUpdate_DoActions{}}}}}}
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
	t.Run("uws_unknown_acceptance_is_not_replayed", func(t *testing.T) {
		lost := &responseLossClient{Client: c, loseMulti: true}
		e := newExecutor(lost)
		a := action()
		require.Error(t, e.Start(ctx, a))
		require.Error(t, e.Start(ctx, a))
		require.Equal(t, 1, lost.multis)
		require.NotEmpty(t, a.GetDoUpdate().GetUpdateId())
		canceled, stop := context.WithCancel(ctx)
		stop()
		require.True(t, errors.Is(e.Start(canceled, a), context.Canceled))
		require.NoError(t, c.SignalWorkflow(ctx, e.WorkflowOptions.ID, "", "finish", nil))
		require.NoError(t, c.GetWorkflow(ctx, e.WorkflowOptions.ID, "").Get(ctx, nil))
	})

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
				foreign, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: start.GetWorkflowId(), TaskQueue: queue, WorkflowIDReusePolicy: enumspb.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE}, "kitchenSink", &WorkflowInput{ExpectedSignalCount: 99})
				if err != nil {
					return err
				}
				foreignRuns <- foreign
				return status.Error(codes.DeadlineExceeded, "lost real multi-operation reply after external ID reuse")
			}
			return err
		}
		wire, err := client.Dial(client.Options{HostPort: server.FrontendHostPort(), Namespace: "default", ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{grpc.WithChainUnaryInterceptor(intercept)}}})
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
		require.Equal(t, 1, requestCount, "outer recovery must not reconstruct a new multi-operation request")
		if firstErr != nil {
			require.Error(t, secondErr)
		}
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
