package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/workers/go/harness"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc"
)

func payloadHeartbeatWorkflow(ctx workflow.Context, heartbeatTimeout time.Duration, local bool, bytesToReturn int32) ([]byte, error) {
	var result []byte
	if local {
		ctx = workflow.WithLocalActivityOptions(ctx, workflow.LocalActivityOptions{StartToCloseTimeout: 10 * time.Second})
		err := workflow.ExecuteLocalActivity(ctx, "payload", []byte("input"), bytesToReturn).Get(ctx, &result)
		return result, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		HeartbeatTimeout:    heartbeatTimeout,
	})
	err := workflow.ExecuteActivity(ctx, "payload", []byte("input"), bytesToReturn).Get(ctx, &result)
	return result, err
}

func TestPayloadInitialHeartbeat(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	for _, tc := range []struct {
		name       string
		timeout    time.Duration
		local      bool
		heartbeats int32
		bytes      int32
	}{
		{name: "remote opted in", timeout: 5 * time.Second, heartbeats: 1, bytes: 17},
		{name: "empty remote opted in", timeout: 5 * time.Second, heartbeats: 1},
		{name: "remote default unchanged", bytes: 17},
		{name: "local unchanged", local: true, bytes: 17},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var heartbeats atomic.Int32
			c, err := client.Dial(client.Options{
				HostPort: address,
				ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{
					grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
						err := invoke(ctx, method, req, reply, cc, opts...)
						if _, ok := req.(*workflowservice.RecordActivityTaskHeartbeatRequest); ok && err == nil {
							heartbeats.Add(1)
						}
						return err
					}),
				}},
			})
			require.NoError(t, err)
			defer c.Close()
			queue := "payload-heartbeat-" + uuid.NewString()
			w := buildWorker(c, harness.WorkerContext{TaskQueue: queue})
			w.RegisterWorkflow(payloadHeartbeatWorkflow)
			require.NoError(t, w.Start())
			defer w.Stop()
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: queue, TaskQueue: queue},
				payloadHeartbeatWorkflow, tc.timeout, tc.local, tc.bytes)
			require.NoError(t, err)
			var result []byte
			require.NoError(t, run.Get(ctx, &result))
			require.Len(t, result, int(tc.bytes))
			require.Equal(t, tc.heartbeats, heartbeats.Load(), "accepted heartbeat RPCs")
		})
	}
}
