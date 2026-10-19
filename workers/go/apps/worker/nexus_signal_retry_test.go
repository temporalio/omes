package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/devserver"
	"github.com/temporalio/omes/internal/versions"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers/go/harness"
	"github.com/temporalio/omes/workers/go/workerlib/kitchensink"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type signalResponseLoss struct {
	interceptor.ClientInterceptorBase
	client client.Client
	lose   bool
	wait   bool
	calls  atomic.Int32
	lost   atomic.Bool
	mu     sync.Mutex
	runs   []string
}

func (loss *signalResponseLoss) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	return &signalResponseLossOutbound{
		ClientOutboundInterceptorBase: interceptor.ClientOutboundInterceptorBase{Next: next},
		loss:                          loss,
	}
}

type signalResponseLossOutbound struct {
	interceptor.ClientOutboundInterceptorBase
	loss *signalResponseLoss
}

func (out *signalResponseLossOutbound) SignalWorkflow(ctx context.Context, input *interceptor.ClientSignalWorkflowInput) error {
	out.loss.calls.Add(1)
	if err := out.Next.SignalWorkflow(ctx, input); err != nil {
		return err
	}
	return out.loss.afterSignal(ctx, input.WorkflowID, input.RunID)
}

func (out *signalResponseLossOutbound) SignalWithStartWorkflow(ctx context.Context, input *interceptor.ClientSignalWithStartWorkflowInput) (client.WorkflowRun, error) {
	out.loss.calls.Add(1)
	run, err := out.Next.SignalWithStartWorkflow(ctx, input)
	if err != nil {
		return nil, err
	}
	out.loss.mu.Lock()
	out.loss.runs = append(out.loss.runs, run.GetRunID())
	out.loss.mu.Unlock()
	if err := out.loss.afterSignal(ctx, run.GetID(), run.GetRunID()); err != nil {
		return nil, err
	}
	return run, nil
}

func (loss *signalResponseLoss) afterSignal(ctx context.Context, workflowID, runID string) error {
	lose := loss.lose && loss.lost.CompareAndSwap(false, true)
	if !lose && !loss.wait {
		return nil
	}
	if err := loss.client.GetWorkflow(ctx, workflowID, runID).Get(ctx, nil); err != nil {
		return fmt.Errorf("waiting for signaled target completion: %w", err)
	}
	if lose {
		return serviceerror.NewUnavailable("injected loss of acknowledgement after signal applied")
	}
	return nil
}

func terminalNexusSignalCaller(ctx workflow.Context, endpoint, target string, withStart, repeat bool) error {
	request := &ks.NexusOperationRequest{Action: &ks.NexusOperationRequest_WorkflowAction{
		WorkflowAction: &ks.NexusWorkflowAction{
			WorkflowId:   target,
			StartOptions: &ks.NexusWorkflowStartOptions{TaskQueue: endpoint},
			Action: &ks.NexusWorkflowAction_Signal{Signal: &ks.DoSignal{
				WithStart: withStart,
				Variant: &ks.DoSignal_DoSignalActions_{DoSignalActions: &ks.DoSignal_DoSignalActions{
					Variant: &ks.DoSignal_DoSignalActions_DoActionsInMain{
						DoActionsInMain: ks.SingleActionSet(ks.NewEmptyReturnResultAction()),
					},
				}},
			}},
		},
	}}
	var result converter.RawValue
	c := workflow.NewNexusClient(endpoint, kitchensink.KitchenSinkServiceName)
	err := c.ExecuteOperation(
		ctx, ks.KitchenSinkNexusOperationName, request,
		workflow.NexusOperationOptions{ScheduleToCloseTimeout: 15 * time.Second},
	).Get(ctx, &result)
	if err != nil || !repeat {
		return err
	}
	return c.ExecuteOperation(ctx, ks.KitchenSinkNexusOperationName, request,
		workflow.NexusOperationOptions{ScheduleToCloseTimeout: 15 * time.Second}).Get(ctx, &result)
}

func startNexusSignalTestServer(t *testing.T) string {
	t.Helper()
	ref, err := versions.Get("SERVER_VERSION")
	require.NoError(t, err)
	server, err := devserver.Start(t.Context(), devserver.Options{
		Ref: ref,
		DynamicConfigValues: map[string]any{
			"history.enableChasm":                true,
			"history.enableCHASMCallbacks":       true,
			"history.enableCHASMSignalBacklinks": true,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, server.Stop()) })
	return server.FrontendHostPort()
}

func TestTerminalNexusSignalRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	c, err := client.Dial(client.Options{HostPort: address, Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	baseClient, err := buildClient(harness.ClientConfig{TargetHost: address, Namespace: "default"})
	require.NoError(t, err)
	t.Cleanup(baseClient.Close)

	for _, tc := range []struct {
		name      string
		lose      bool
		missing   bool
		withStart bool
		repeat    bool
	}{
		{name: "acknowledged signal"},
		{name: "missing target remains a failure", missing: true},
		{name: "lost acknowledgement retries safely", lose: true},
		{name: "distinct operation does not reuse acknowledgement", repeat: true},
		{name: "distinct signal with start can start another run", repeat: true, withStart: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 25*time.Second)
			defer cancel()
			queue := "terminal-signal-" + uuid.NewString()
			endpoint := queue
			target := queue + "/target"
			loss := &signalResponseLoss{client: c, lose: tc.lose, wait: tc.repeat}
			workerClient, err := client.NewClientFromExisting(baseClient, client.Options{
				Namespace:    "default",
				Interceptors: []interceptor.ClientInterceptor{loss},
			})
			require.NoError(t, err)
			t.Cleanup(workerClient.Close)
			service := nexus.NewService(kitchensink.KitchenSinkServiceName)
			require.NoError(t, service.Register(kitchensink.KitchenSinkNexusOperation))
			w := worker.New(workerClient, queue, worker.Options{})
			w.RegisterWorkflow(kitchensink.KitchenSinkWorkflow)
			w.RegisterWorkflow(terminalNexusSignalCaller)
			w.RegisterNexusService(service)
			require.NoError(t, w.Start())
			t.Cleanup(w.Stop)
			_, err = c.OperatorService().CreateNexusEndpoint(ctx, &operatorservice.CreateNexusEndpointRequest{
				Spec: &nexuspb.EndpointSpec{
					Name: endpoint,
					Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
						Worker: &nexuspb.EndpointTarget_Worker{Namespace: "default", TaskQueue: queue},
					}},
				},
			})
			require.NoError(t, err)
			if !tc.missing && !tc.withStart {
				_, err = c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
					ID: target, TaskQueue: queue, WorkflowExecutionTimeout: 20 * time.Second,
				}, kitchensink.KitchenSinkWorkflow, &ks.WorkflowInput{})
				require.NoError(t, err)
			}
			caller, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
				ID: queue + "/caller", TaskQueue: queue, WorkflowExecutionTimeout: 20 * time.Second,
			}, terminalNexusSignalCaller, endpoint, target, tc.withStart, tc.repeat)
			require.NoError(t, err)
			callErr := caller.Get(ctx, nil)
			loss.mu.Lock()
			runs := append([]string(nil), loss.runs...)
			loss.mu.Unlock()
			if tc.missing {
				var operationErr *temporal.NexusOperationError
				require.ErrorAs(t, callErr, &operationErr)
				require.Equal(t, "NOT_FOUND", operationErr.Failure.GetCause().GetNexusHandlerFailureInfo().GetType())
				require.False(t, loss.lost.Load())
				return
			}
			require.NoError(t, c.GetWorkflow(ctx, target, "").Get(ctx, nil))
			targetState, err := c.DescribeWorkflowExecution(ctx, target, "")
			require.NoError(t, err)
			require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, targetState.WorkflowExecutionInfo.Status)
			if tc.repeat {
				require.Equal(t, int32(2), loss.calls.Load())
				if tc.withStart {
					require.NoError(t, callErr)
					require.Len(t, runs, 2)
					require.NotEqual(t, runs[0], runs[1])
				} else {
					var operationErr *temporal.NexusOperationError
					require.ErrorAs(t, callErr, &operationErr)
					require.Equal(t, "NOT_FOUND", operationErr.Failure.GetCause().GetNexusHandlerFailureInfo().GetType())
				}
				return
			}
			if tc.lose {
				require.True(t, loss.lost.Load())
				require.GreaterOrEqual(t, loss.calls.Load(), int32(2))
			}
			require.NoError(t, callErr, "an applied terminal signal must survive a retried Nexus handler")
		})
	}
}
