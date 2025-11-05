package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/workflowservice/v1"
)

const OmesExecutionIDSearchAttribute = "OmesExecutionID"

// WorkflowCompletionVerifier allows verifying the workflow completion count after a scenario completed.
type WorkflowCompletionVerifier struct {
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
func (wct *WorkflowCompletionVerifier) SetExpectedWorkflowCount(fn func(ExecutorState) int) {
	wct.expectedWorkflowCount = fn
}

// NewWorkflowCompletionChecker creates a new checker with the given timeout.
// If timeout is zero, it uses a default of 30 seconds.
// Call this before the scenario is started to initialize and register search attributes.
func NewWorkflowCompletionChecker(ctx context.Context, info ScenarioInfo, timeout time.Duration) (*WorkflowCompletionVerifier, error) {
	if timeout == 0 {
		timeout = 3 * time.Minute // TODO: set back to 30s
	}

	checker := &WorkflowCompletionVerifier{
		timeout: timeout,
	}

	if err := checker.init(ctx, info); err != nil {
		return nil, err
	}

	return checker, nil
}

func (wct *WorkflowCompletionVerifier) init(ctx context.Context, info ScenarioInfo) error {
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

// VerifyRun implements the Verifier interface.
// It checks that the expected number of workflows have completed using the provided state.
func (wct *WorkflowCompletionVerifier) VerifyRun(ctx context.Context, info ScenarioInfo, state ExecutorState) []error {
	return wct.Verify(ctx, state)
}

// Verify checks that the expected number of workflows have completed.
func (wct *WorkflowCompletionVerifier) Verify(ctx context.Context, state ExecutorState) []error {
	var allErrors []error

	// Calculate expected workflow count
	expectedCount := state.CompletedIterations
	if wct.expectedWorkflowCount != nil {
		expectedCount = wct.expectedWorkflowCount(state)
	}

	// (1) Verify that we have completions at all.
	if expectedCount == 0 {
		allErrors = append(allErrors, fmt.Errorf("no workflows completed"))
	} else {
		// (2) Verify that all completed workflows have indeed completed.
		verifyCtx, cancel := context.WithTimeout(ctx, wct.timeout)
		defer cancel()

		query := fmt.Sprintf(
			"%s='%s' AND ExecutionStatus = 'Completed'",
			OmesExecutionIDSearchAttribute,
			wct.info.ExecutionID,
		)

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
	}

	// (3) Verify that all started workflows have completed.
	nonCompletedErrs := GetNonCompletedWorkflows(
		ctx,
		wct.info,
		OmesExecutionIDSearchAttribute,
		wct.info.ExecutionID,
		10,
	)
	allErrors = append(allErrors, nonCompletedErrs...)

	return allErrors
}

// TODO: remove this
// VerifyNoRunningWorkflows waits until there are no running workflows on the task queue for the given run ID.
// This is useful for scenarios that want to ensure all started workflows have completed.
func (wct *WorkflowCompletionVerifier) VerifyNoRunningWorkflows(ctx context.Context) error {
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
