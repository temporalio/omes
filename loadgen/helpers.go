package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
)

// MinVisibilityCountEventually checks that the given visibility query returns at least the expected
// number of workflows. It repeatedly queries until it either finds the expected count or times out.
func MinVisibilityCountEventually(
	ctx context.Context,
	info ScenarioInfo,
	request *workflowservice.CountWorkflowExecutionsRequest,
	minCount int,
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
			return fmt.Errorf("expected at least %d workflows in visibility, got %d after waiting %v",
				minCount, lastVisibilityCount, waitAtMost)

		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected at least: %d)\n", lastVisibilityCount, minCount)

		case <-countTicker.C:
			visibilityCount, err := info.Client.CountWorkflow(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to count workflows in visibility: %w", err)
			}
			lastVisibilityCount = visibilityCount.Count
			if lastVisibilityCount >= int64(minCount) {
				return nil
			}
		}
	}
}
