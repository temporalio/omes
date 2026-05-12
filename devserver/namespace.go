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

// registerNamespace registers `namespace` on the running server. No-op if it
// already exists. Retries for up to 30s while the frontend warms up.
func registerNamespace(ctx context.Context, frontendAddr, namespace string) error {
	conn, err := grpc.NewClient(frontendAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("dial frontend: %w", err)
	}
	defer conn.Close()

	wf := workflowservice.NewWorkflowServiceClient(conn)
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := wf.RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
			Namespace:                        namespace,
			WorkflowExecutionRetentionPeriod: durationpb.New(24 * time.Hour),
		})
		if err == nil || isAlreadyExists(err) {
			return nil
		}
		if time.Now().After(deadline) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
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
