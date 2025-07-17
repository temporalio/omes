package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
)

// VisibilityCountIsEventually ensures that some visibility query count matches the provided
// expected number within the provided time limit.
func VisibilityCountIsEventually(
	ctx context.Context,
	info ScenarioInfo,
	request *workflowservice.CountWorkflowExecutionsRequest,
	expectedCount int,
	waitAtMost time.Duration,
) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, waitAtMost)
	defer cancel()

	countTicker := time.NewTicker(3 * time.Second)
	defer countTicker.Stop()

	printTicker := time.NewTicker(30 * time.Second)
	defer printTicker.Stop()

	var lastVisibilityCount int64
	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("expected %d workflows in visibility, got %d after waiting %v",
				expectedCount, lastVisibilityCount, waitAtMost)

		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected: %d)\n", lastVisibilityCount, expectedCount)

		case <-countTicker.C:
			visibilityCount, err := info.Client.CountWorkflow(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to count workflows in visibility: %w", err)
			}
			lastVisibilityCount = visibilityCount.Count
			if lastVisibilityCount == int64(expectedCount) {
				return nil
			}
		}
	}
}
