package devserver

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

// registerNamespace registers a namespace.
// No-op if it already exists. Retries transient errors.
func registerNamespace(ctx context.Context, frontendAddr, namespace string) error {
	conn, err := grpc.NewClient(frontendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial frontend: %w", err)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	wf := workflowservice.NewWorkflowServiceClient(conn)
	for {
		_, err := wf.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
			Namespace:                        namespace,
			WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour),
		})
		if err == nil || isAlreadyExists(err) {
			return nil
		}
		if !isRetryable(err) {
			return err
		}

		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-time.After(time.Second):
		}
	}
}

func isAlreadyExists(err error) bool {
	var already *serviceerror.NamespaceAlreadyExists
	if errors.As(err, &already) {
		return true
	}
	if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
		return true
	}
	return false
}

func isRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return true
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.ResourceExhausted, codes.Aborted, codes.Internal:
		return true
	default:
		return false
	}
}
