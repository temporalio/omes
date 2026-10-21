package kitchensink

import (
	"context"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

type nexusSignalRequestIDKey struct{}

// NexusSignalRequestIDInterceptor sets the Nexus operation's request ID on the
// signal the kitchen-sink handler sends, so a redelivered operation is
// deduplicated by the server instead of signaling a target that may have
// already completed. Install it on the client used to run
// KitchenSinkNexusOperation.
func NexusSignalRequestIDInterceptor(ctx context.Context, method string, request, response any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
	if id, _ := ctx.Value(nexusSignalRequestIDKey{}).(string); id != "" {
		// Each signal request type has its own RequestId field.
		switch signal := request.(type) {
		case *workflowservice.SignalWorkflowExecutionRequest:
			signal.RequestId = id
		case *workflowservice.SignalWithStartWorkflowExecutionRequest:
			signal.RequestId = id
		}
	}
	return invoke(ctx, method, request, response, connection, options...)
}
