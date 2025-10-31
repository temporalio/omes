package loadgen

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
)

const OmesExecutionIDSearchAttribute = "OmesExecutionID"

// WorkflowCompletionChecker allows verifying the workflow completion count after a scenario completed.
type WorkflowCompletionChecker struct {
	// expectedWorkflowCount is an optional function to calculate the expected number of workflows
	// from the ExecutorState. If nil, defaults to using state.CompletedIterations.
	expectedWorkflowCount func(ExecutorState) int
	// timeout is the maximum time to wait for workflow completion verification in visibility.
	timeout time.Duration
	// info is the scenario information stored during initialization.
	info ScenarioInfo
}

// SetExpectedWorkflowCount sets a custom function to calculate the expected number of workflows.
// If not set, defaults to using state.CompletedIterations.
func (wct *WorkflowCompletionChecker) SetExpectedWorkflowCount(fn func(ExecutorState) int) {
	wct.expectedWorkflowCount = fn
}

// GetExpectedWorkflowCount returns the expected workflow count for the given state.
// If a custom function was set via SetExpectedWorkflowCount, it uses that.
// Otherwise, defaults to state.CompletedIterations.
func (wct *WorkflowCompletionChecker) GetExpectedWorkflowCount(state ExecutorState) int {
	if wct.expectedWorkflowCount != nil {
		return wct.expectedWorkflowCount(state)
	}
	return state.CompletedIterations
}

// NewWorkflowCompletionChecker creates a new checker with the given timeout.
// If timeout is zero, it uses a default of 30 seconds.
// Call this before the scenario is started to initialize and register search attributes.
func NewWorkflowCompletionChecker(ctx context.Context, info ScenarioInfo, timeout time.Duration) (*WorkflowCompletionChecker, error) {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	checker := &WorkflowCompletionChecker{
		timeout: timeout,
	}

	if err := checker.init(ctx, info); err != nil {
		return nil, err
	}

	return checker, nil
}

func (wct *WorkflowCompletionChecker) init(ctx context.Context, info ScenarioInfo) error {
	// Store the scenario info for later use
	wct.info = info

	if info.Configuration.DoNotRegisterSearchAttributes {
		return nil
	}

	if err := InitSearchAttribute(ctx, info, OmesExecutionIDSearchAttribute); err != nil {
		return fmt.Errorf("failed to register search attribute %s: %w",
			OmesExecutionIDSearchAttribute, err)
	}
	return nil
}

// Verify checks that the expected number of workflows have completed.
func (wct *WorkflowCompletionChecker) Verify(ctx context.Context, state ExecutorState) error {
	var allErrors []error

	// Calculate expected workflow count
	expectedCount := state.CompletedIterations
	if wct.expectedWorkflowCount != nil {
		expectedCount = wct.expectedWorkflowCount(state)
	}

	// Check that we have completions to verify
	if expectedCount == 0 {
		return fmt.Errorf("no workflows completed")
	}

	query := fmt.Sprintf(
		"%s='%s' AND ExecutionStatus = 'Completed'",
		OmesExecutionIDSearchAttribute,
		wct.info.OmesRunID(),
	)

	verifyCtx, cancel := context.WithTimeout(ctx, wct.timeout)
	defer cancel()

	err := MinVisibilityCountEventually(
		verifyCtx,
		wct.info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: wct.info.Namespace,
			Query:     query,
		},
		expectedCount,
		wct.timeout,
	)

	if err != nil {
		allErrors = append(allErrors, err)
	}

	// If verification failed, query for non-completed workflows to aid debugging
	if err != nil {
		workflowDetails, listErr := GetNonCompletedWorkflows(
			ctx,
			wct.info,
			OmesExecutionIDSearchAttribute,
			wct.info.OmesRunID(),
			10,
		)

		if listErr != nil {
			allErrors = append(allErrors, fmt.Errorf("failed to list non-completed workflows: %w", listErr))
		} else if workflowDetails != "" {
			allErrors = append(allErrors, fmt.Errorf("non-completed workflows found:%s", workflowDetails))
		}
	}

	return errors.Join(allErrors...)
}

// VerifyNoRunningWorkflows waits until there are no running workflows on the task queue for the given run ID.
// This is useful for scenarios that want to ensure all started workflows have completed.
func (wct *WorkflowCompletionChecker) VerifyNoRunningWorkflows(ctx context.Context) error {
	query := fmt.Sprintf("TaskQueue = %q and ExecutionStatus = 'Running'",
		TaskQueueForRun(wct.info.RunID))

	verifyCtx, cancel := context.WithTimeout(ctx, wct.timeout)
	defer cancel()

	return MinVisibilityCountEventually(
		verifyCtx,
		wct.info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: wct.info.Namespace,
			Query:     query,
		},
		0,
		wct.timeout,
	)
}
