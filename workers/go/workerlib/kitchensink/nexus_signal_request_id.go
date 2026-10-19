package kitchensink

import (
	"context"

	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

type nexusSignalRequestIDKey struct{}

// NexusSignalRequestIDInterceptor applies the kitchen-sink handler's explicit
// request ID to its one signal per operation. Install it on the client used
// to run KitchenSinkNexusOperation.
func NexusSignalRequestIDInterceptor(ctx context.Context, method string, request, response any, connection *grpc.ClientConn, invoke grpc.UnaryInvoker, options ...grpc.CallOption) error {
	if id, _ := ctx.Value(nexusSignalRequestIDKey{}).(string); id != "" {
		switch request := request.(type) {
		case *workflowservice.SignalWorkflowExecutionRequest:
			request.RequestId = id
		case *workflowservice.SignalWithStartWorkflowExecutionRequest:
			request.RequestId = id
		}
	}
	return invoke(ctx, method, request, response, connection, options...)
}
