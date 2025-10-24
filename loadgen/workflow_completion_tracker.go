package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
)

const OmesRunIDSearchAttribute = "OmesRunID"

// WorkflowCompletionChecker allows verifying the workflow completion count after a scenario completed.
// Call Init before the scenario is started; then call Verify after the scenario completed.
type WorkflowCompletionChecker struct {
	// ExpectedWorkflowCount is an optional function to calculate the expected number of workflows
	// from the ExecutorState. If nil, defaults to using state.CompletedIterations.
	ExpectedWorkflowCount func(ExecutorState) int
	// Timeout is the maximum time to wait for workflow completion verification in visibility.
	// If zero, defaults to 3 minutes.
	Timeout time.Duration
}

func (wct WorkflowCompletionChecker) Init(ctx context.Context, info ScenarioInfo) error {
	if info.Configuration.DoNotRegisterSearchAttributes {
		return nil
	}

	if err := InitSearchAttribute(ctx, info, OmesRunIDSearchAttribute); err != nil {
		return fmt.Errorf("failed to register search attribute %s: %w",
			OmesRunIDSearchAttribute, err)
	}
	return nil
}

func (wct WorkflowCompletionChecker) Verify(ctx context.Context, info ScenarioInfo, state ExecutorState) error {
	// Calculate expected workflow count
	expectedCount := state.CompletedIterations
	if wct.ExpectedWorkflowCount != nil {
		expectedCount = wct.ExpectedWorkflowCount(state)
	}

	// Check that we have completions to verify
	if expectedCount == 0 {
		return fmt.Errorf("no workflows completed")
	}

	query := fmt.Sprintf(
		"%s='%s' AND ExecutionStatus = 'Completed'",
		OmesRunIDSearchAttribute,
		info.RunID,
	)

	timeout := wct.Timeout
	if timeout == 0 {
		timeout = 3 * time.Minute
	}

	verifyCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return MinVisibilityCountEventually(
		verifyCtx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query:     query,
		},
		expectedCount,
		timeout,
	)
}
