package kitchensink

import (
	"context"
	"errors"
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/api/failure/v1"
	"go.temporal.io/api/notificationservice/v1"
	"go.temporal.io/sdk/temporalnexus"
)

// Worker callbacks are NexusHandler-variant completion callbacks (commonpb.Callback_NexusHandler).
// When a standalone Nexus operation (SANO) with one attached completes, the server delivers a
// notificationservice.OnCompleteRequest carrying the outcome to a Nexus operation on a worker's task
// queue, addressed by service and operation name rather than through an endpoint.
//
// The client side (scenarios/nexus_worker_callbacks.go) and the Go worker's handler share this
// contract. The client puts the outcome it expects in the callback's source context, and the handler
// checks what the server delivered against it. A mismatch fails the handler, which the server records
// as a failed callback, so the client can verify delivery with DescribeNexusOperationExecution alone.
const (
	// WorkerCallbackServiceName is the Nexus service hosting the worker callback handler.
	WorkerCallbackServiceName = "omes-worker-callbacks"
	// WorkerCallbackOperationName is the operation the server invokes as the worker callback.
	WorkerCallbackOperationName = "on-complete"
)

// WorkerCallbackExpectation is the outcome a worker callback is expected to deliver.
type WorkerCallbackExpectation string

const (
	// WorkerCallbackExpectSuccess expects a successful result equal to the source context's EchoValue.
	WorkerCallbackExpectSuccess WorkerCallbackExpectation = "success"
	// WorkerCallbackExpectTimeout expects a failure caused by the operation timing out.
	WorkerCallbackExpectTimeout WorkerCallbackExpectation = "timeout"
)

// WorkerCallbackSourceContext is attached to a worker callback as its source context.
type WorkerCallbackSourceContext struct {
	// OperationID identifies the SANO the callback is attached to, for diagnostics.
	OperationID string `json:"operationId"`
	// Expect is the outcome the callback must deliver.
	Expect WorkerCallbackExpectation `json:"expect"`
	// EchoValue is the result expected when Expect is WorkerCallbackExpectSuccess.
	EchoValue string `json:"echoValue,omitempty"`
}

// ValidateWorkerCallback checks that a worker callback delivered the outcome its source context
// expects. It returns the decoded source context, which is zero when it couldn't be decoded.
func ValidateWorkerCallback(req *notificationservice.OnCompleteRequest) (WorkerCallbackSourceContext, error) {
	var sc WorkerCallbackSourceContext
	if req.GetSourceContext() == nil {
		return sc, errors.New("worker callback has no source context")
	}
	if err := jsonPayloadConverter.FromPayload(req.GetSourceContext(), &sc); err != nil {
		return WorkerCallbackSourceContext{}, fmt.Errorf("failed to decode worker callback source context: %w", err)
	}

	switch sc.Expect {
	case WorkerCallbackExpectSuccess:
		if f := req.GetFailure(); f != nil {
			return sc, fmt.Errorf("operation %q: worker callback delivered a failure, expected success: %s",
				sc.OperationID, f.GetMessage())
		}
		if req.GetSuccess() == nil {
			return sc, fmt.Errorf("operation %q: worker callback delivered no result", sc.OperationID)
		}
		var got string
		if err := jsonPayloadConverter.FromPayload(req.GetSuccess(), &got); err != nil {
			return sc, fmt.Errorf("operation %q: failed to decode worker callback result: %w", sc.OperationID, err)
		}
		if got != sc.EchoValue {
			return sc, fmt.Errorf("operation %q: worker callback result mismatch: got %q, want %q",
				sc.OperationID, got, sc.EchoValue)
		}
		return sc, nil

	case WorkerCallbackExpectTimeout:
		f := req.GetFailure()
		if f == nil {
			return sc, fmt.Errorf("operation %q: worker callback delivered success, expected a timeout failure", sc.OperationID)
		}
		if !hasTimeoutFailure(f) {
			return sc, fmt.Errorf("operation %q: worker callback delivered a failure without a timeout cause: %s",
				sc.OperationID, f.GetMessage())
		}
		return sc, nil

	default:
		return sc, fmt.Errorf("operation %q: unknown worker callback expectation %q", sc.OperationID, sc.Expect)
	}
}

// hasTimeoutFailure reports whether f or any failure in its cause chain is a timeout. The server may
// wrap an operation's timeout in another failure before delivering it.
func hasTimeoutFailure(f *failure.Failure) bool {
	for ; f != nil; f = f.GetCause() {
		if f.GetTimeoutFailureInfo() != nil {
			return true
		}
	}
	return false
}

// WorkerCallbackOperation is the Nexus operation the server invokes as a worker callback. Workers
// register it in a service named WorkerCallbackServiceName.
//
// It checks the delivered outcome with ValidateWorkerCallback. A mismatch is answered with a
// non-retryable handler error, which the server records as a failed callback carrying this error's
// message; the client reads that from DescribeNexusOperationExecution.
var WorkerCallbackOperation = nexus.NewSyncOperation(
	WorkerCallbackOperationName,
	func(ctx context.Context, req *notificationservice.OnCompleteRequest, _ nexus.StartOperationOptions) (*notificationservice.OnCompleteResponse, error) {
		sc, err := ValidateWorkerCallback(req)
		if err != nil {
			temporalnexus.GetLogger(ctx).Warn("Worker callback validation failed.",
				"operationID", sc.OperationID, "error", err)
			return nil, nexus.NewHandlerErrorf(nexus.HandlerErrorTypeBadRequest, "%v", err)
		}
		return &notificationservice.OnCompleteResponse{}, nil
	},
)
