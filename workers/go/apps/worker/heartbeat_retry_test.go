package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers/go/harness"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
)

func TestHeartbeatTimeoutRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	for _, delayHeartbeat := range []bool{false, true} {
		name := "healthy retry completes"
		if delayHeartbeat {
			name = "healthy retry past heartbeat deadline fails"
		}
		t.Run(name, func(t *testing.T) {
			var heartbeats atomic.Int32
			c, err := client.Dial(client.Options{
				HostPort: address,
				ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{
					grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
						if _, ok := req.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok {
							heartbeats.Add(1)
							if delayHeartbeat {
								<-ctx.Done()
								return ctx.Err()
							}
						}
						return invoke(ctx, method, req, reply, cc, opts...)
					}),
				}},
			})
			require.NoError(t, err)
			defer c.Close()
			queue := "heartbeat-retry-" + uuid.NewString()
			w := buildWorker(c, harness.WorkerContext{TaskQueue: queue})
			require.NoError(t, w.Start())
			defer w.Stop()
			ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
			defer cancel()
			run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: queue, TaskQueue: queue},
				"kitchenSink", &ks.WorkflowInput{InitialActions: ks.ListActionSet(
					ks.HeartbeatActivity(1, 0, 2*time.Second, 10*time.Second, time.Second, 2, 500*time.Millisecond, 1),
					ks.NewEmptyReturnResultAction(),
				)})
			require.NoError(t, err)
			err = run.Get(ctx, nil)
			if delayHeartbeat {
				var timeout *temporal.TimeoutError
				require.ErrorAs(t, err, &timeout)
				require.Equal(t, enums.TIMEOUT_TYPE_HEARTBEAT, timeout.TimeoutType())
			} else {
				require.NoError(t, err)
			}
			require.Positive(t, heartbeats.Load())
			history := c.GetWorkflowHistory(ctx, run.GetID(), run.GetRunID(), false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
			var attempt int32
			var terminal enums.EventType
			for history.HasNext() {
				event, err := history.Next()
				require.NoError(t, err)
				if started := event.GetActivityTaskStartedEventAttributes(); started != nil {
					attempt = started.Attempt
				}
				terminal = event.EventType
			}
			require.EqualValues(t, 2, attempt)
			wantTerminal := enums.EVENT_TYPE_WORKFLOW_EXECUTION_COMPLETED
			if delayHeartbeat {
				wantTerminal = enums.EVENT_TYPE_WORKFLOW_EXECUTION_FAILED
			}
			require.Equal(t, wantTerminal, terminal)
		})
	}
}
