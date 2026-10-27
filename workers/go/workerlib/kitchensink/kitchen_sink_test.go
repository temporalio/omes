package kitchensink

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// A Nexus signal that completes its target must survive a retried Nexus task: the retry
// has to be deduplicated instead of failing with NOT_FOUND on the closed workflow.
func TestNexusSignalRetryAfterTargetCompleted(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()

	temporalPath, err := exec.LookPath("temporal")
	if err != nil {
		t.Skipf("temporal CLI not available: %v", err)
	}
	devServer, err := testsuite.StartDevServer(ctx, testsuite.DevServerOptions{
		ExistingPath:  temporalPath,
		ClientOptions: &client.Options{Namespace: "default"},
		LogLevel:      "error",
		Stdout:        io.Discard,
		Stderr:        io.Discard,
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = devServer.Stop() })
	c := devServer.Client()

	taskQueue := "nexus-signal-retry-" + uuid.NewString()
	endpoint := "nexus-signal-retry-" + uuid.NewString()[:8]
	_, err = c.OperatorService().CreateNexusEndpoint(ctx, &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexuspb.EndpointSpec{
			Name: endpoint,
			Target: &nexuspb.EndpointTarget{
				Variant: &nexuspb.EndpointTarget_Worker_{Worker: &nexuspb.EndpointTarget_Worker{
					Namespace: "default",
					TaskQueue: taskQueue,
				}},
			},
		},
	})
	require.NoError(t, err)

	// Run the real handler, but fail the first attempt after it succeeded, as if the
	// response was lost. The server then retries with the same request ID.
	var mu sync.Mutex
	attempts := map[string]int{}
	flakyOp := temporalnexus.MustNewTemporalOperation(
		temporalnexus.TemporalOperationOptions[*kitchensink.NexusOperationRequest, converter.RawValue]{
			Name: kitchensink.KitchenSinkNexusOperationName,
			Start: func(
				ctx context.Context,
				nc temporalnexus.NexusClient,
				input *kitchensink.NexusOperationRequest,
				opts temporalnexus.StartTemporalOperationOptions,
			) (temporalnexus.TemporalOperationResult[converter.RawValue], error) {
				res, err := startNexusOperation(ctx, nc, input, opts)
				mu.Lock()
				attempts[opts.RequestID]++
				first := attempts[opts.RequestID] == 1
				mu.Unlock()
				if err == nil && first {
					return res, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeUnavailable, "injected failure after signal")
				}
				return res, err
			},
		})
	service := nexus.NewService(KitchenSinkServiceName)
	require.NoError(t, service.Register(flakyOp))

	w := worker.New(c, taskQueue, worker.Options{})
	w.RegisterWorkflowWithOptions(KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	w.RegisterWorkflowWithOptions(nexusSignalCallerWorkflow, workflow.RegisterOptions{Name: "nexusSignalCaller"})
	w.RegisterNexusService(service)
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)

	// The target waits for signals; the Nexus signal below completes it.
	targetID := "nexus-signal-target-" + uuid.NewString()
	target, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: targetID, TaskQueue: taskQueue},
		"kitchenSink", &kitchensink.WorkflowInput{})
	require.NoError(t, err)

	caller, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{TaskQueue: taskQueue},
		"nexusSignalCaller", endpoint, &kitchensink.NexusOperationRequest{
			Action: &kitchensink.NexusOperationRequest_WorkflowAction{WorkflowAction: &kitchensink.NexusWorkflowAction{
				WorkflowId: targetID,
				Action: &kitchensink.NexusWorkflowAction_Signal{Signal: &kitchensink.DoSignal{
					Variant: &kitchensink.DoSignal_DoSignalActions_{DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActionsInMain{
							DoActionsInMain: kitchensink.SingleActionSet(kitchensink.NewEmptyReturnResultAction()),
						},
					}},
				}},
			}},
		})
	require.NoError(t, err)
	require.NoError(t, caller.Get(ctx, nil))
	require.NoError(t, target.Get(ctx, nil))

	mu.Lock()
	var retried bool
	for _, n := range attempts {
		retried = retried || n > 1
	}
	mu.Unlock()
	require.True(t, retried, "the injected failure should have caused a retry")

	var signals int
	iter := c.GetWorkflowHistory(ctx, targetID, target.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for iter.HasNext() {
		event, err := iter.Next()
		require.NoError(t, err)
		if event.GetEventType() == enumspb.EVENT_TYPE_WORKFLOW_EXECUTION_SIGNALED {
			signals++
		}
	}
	require.Equal(t, 1, signals)
}

func nexusSignalCallerWorkflow(ctx workflow.Context, endpoint string, input *kitchensink.NexusOperationRequest) error {
	nc := workflow.NewNexusClient(endpoint, KitchenSinkServiceName)
	return nc.ExecuteOperation(ctx, kitchensink.KitchenSinkNexusOperationName, input, workflow.NexusOperationOptions{
		ScheduleToCloseTimeout: 20 * time.Second,
	}).Get(ctx, nil)
}
