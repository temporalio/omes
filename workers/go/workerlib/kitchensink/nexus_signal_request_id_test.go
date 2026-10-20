package kitchensink

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
)

func TestNexusSignalRequestIDInterceptor(t *testing.T) {
	for _, tc := range []struct {
		name        string
		request     interface{ GetRequestId() string }
		operationID string
		want        string
	}{
		{name: "signal", request: &workflowservice.SignalWorkflowExecutionRequest{RequestId: "generated"}, operationID: "operation", want: "operation"},
		{name: "signal with start", request: &workflowservice.SignalWithStartWorkflowExecutionRequest{RequestId: "generated"}, operationID: "operation", want: "operation"},
		{name: "ordinary signal", request: &workflowservice.SignalWorkflowExecutionRequest{RequestId: "generated"}, want: "generated"},
		{name: "unrelated request", request: &workflowservice.RequestCancelWorkflowExecutionRequest{RequestId: "generated"}, operationID: "operation", want: "generated"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.operationID != "" {
				ctx = context.WithValue(ctx, nexusSignalRequestIDKey{}, tc.operationID)
			}
			transportErr := errors.New("transport failure")
			called := false
			err := NexusSignalRequestIDInterceptor(ctx, "method", tc.request, nil, nil,
				func(_ context.Context, _ string, request, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
					called = true
					require.Equal(t, tc.want, request.(interface{ GetRequestId() string }).GetRequestId())
					return transportErr
				})
			require.True(t, called)
			require.ErrorIs(t, err, transportErr)
		})
	}
}
