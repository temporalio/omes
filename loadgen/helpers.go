package loadgen

import (
	"context"
	"fmt"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"time"
)

func VisibilityCountIsEventually(
	ctx context.Context,
	client client.Client,
	request *workflowservice.CountWorkflowExecutionsRequest,
	expectedCount int,
	waitAtMost time.Duration,
) error {
	deadline := time.Now().Add(waitAtMost)
	for {
		visibilityCount, err := client.CountWorkflow(ctx, request)
		if err != nil {
			return err
		}
		if visibilityCount.Count == int64(expectedCount) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("expected %d workflows in visibility, got %d after waiting %v",
				expectedCount, visibilityCount.Count, waitAtMost)
		}
		time.Sleep(5 * time.Second)
	}
}
