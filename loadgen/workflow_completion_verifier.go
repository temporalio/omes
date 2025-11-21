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
	checker := &WorkflowCompletionVerifier{}

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

	// Retry InitSearchAttribute until context deadline expires
	retryTicker := time.NewTicker(2 * time.Second)
	defer retryTicker.Stop()

	// Try immediately first
	var lastErr error
	if err := InitSearchAttribute(ctx, info, OmesExecutionIDSearchAttribute); err != nil {
		lastErr = err
		info.Logger.Warnf("failed to register search attribute %s, will retry: %v",
			OmesExecutionIDSearchAttribute, err)
	} else {
		return nil
	}

	// Retry loop until context deadline
	for {
		select {
		case <-ctx.Done():
			// Context ended (deadline or cancellation). Return last error.
			return fmt.Errorf("failed to register search attribute %s after retries: %w",
				OmesExecutionIDSearchAttribute, lastErr)
		case <-retryTicker.C:
			// Don't perform retry if context is already done
			if ctx.Err() != nil {
				return fmt.Errorf("failed to register search attribute %s after retries: %w",
					OmesExecutionIDSearchAttribute, lastErr)
			}
			if err := InitSearchAttribute(ctx, info, OmesExecutionIDSearchAttribute); err != nil {
				lastErr = err
				info.Logger.Warnf("failed to register search attribute %s, will retry: %v",
					OmesExecutionIDSearchAttribute, err)
			} else {
				info.Logger.Infof("successfully registered search attribute %s after retries",
					OmesExecutionIDSearchAttribute)
				return nil
			}
		}
	}
}

// VerifyRun implements the Verifier interface.
// It checks that the expected number of workflows have completed using the provided state.
func (wct *WorkflowCompletionVerifier) VerifyRun(ctx context.Context, info ScenarioInfo, state ExecutorState) []error {
	return wct.Verify(ctx, state)
}

// Verify checks that the expected number of workflows have completed.
// It retries all checks until the context deadline is reached.
func (wct *WorkflowCompletionVerifier) Verify(ctx context.Context, state ExecutorState) []error {
	// Calculate expected workflow count
	expectedCount := state.CompletedIterations
	if wct.expectedWorkflowCount != nil {
		expectedCount = wct.expectedWorkflowCount(state)
	}

	// (1) Verify that we have completions at all.
	if expectedCount == 0 {
		return []error{fmt.Errorf("no workflows completed")}
	}

	// Setup retry loop
	checkTicker := time.NewTicker(15 * time.Second)
	defer checkTicker.Stop()

	printTicker := time.NewTicker(30 * time.Second)
	defer printTicker.Stop()

	query := fmt.Sprintf(
		"%s='%s' AND ExecutionStatus = 'Completed'",
		OmesExecutionIDSearchAttribute,
		wct.info.ExecutionID,
	)

	wct.info.Logger.Infof("Visibility query for completed workflows - CLI command: temporal workflow count --namespace %s --query %q",
		wct.info.Namespace, query)

	var lastErrors []error

	// Function to perform all checks
	performChecks := func() []error {
		var allErrors []error

		// (2) Verify that all completed workflows have indeed completed.
		err := MinVisibilityCount(
			ctx,
			wct.info,
			&workflowservice.CountWorkflowExecutionsRequest{
				Namespace: wct.info.Namespace,
				Query:     query,
			},
			expectedCount,
		)
		if err != nil {
			allErrors = append(allErrors, err)
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

	// Initial check
	lastErrors = performChecks()
	if len(lastErrors) == 0 {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Context ended (deadline or cancellation). Return last errors.
			return lastErrors
		case <-printTicker.C:
			wct.info.Logger.Infof("verification still has error(s), retrying until deadline: %v", lastErrors)
		case <-checkTicker.C:
			// Don't perform checks if context is already done
			if ctx.Err() != nil {
				return lastErrors
			}
			lastErrors = performChecks()
			if len(lastErrors) == 0 {
				return nil
			}
		}
	}
}

// TODO: remove this
// VerifyNoRunningWorkflows waits until there are no running workflows on the task queue for the given run ID.
// This is useful for scenarios that want to ensure all started workflows have completed.
// It retries the check until the context deadline is reached.
func (wct *WorkflowCompletionVerifier) VerifyNoRunningWorkflows(ctx context.Context) error {
	query := fmt.Sprintf("TaskQueue = %q and ExecutionStatus = 'Running'",
		TaskQueueForRun(wct.info.RunID))

	wct.info.Logger.Infof("Visibility query for running workflows - CLI command: temporal workflow count --namespace %s --query %q",
		wct.info.Namespace, query)

	// Setup retry loop
	checkTicker := time.NewTicker(3 * time.Second)
	defer checkTicker.Stop()

	printTicker := time.NewTicker(30 * time.Second)
	defer printTicker.Stop()

	var lastError error

	// Function to perform check
	performCheck := func() error {
		return MinVisibilityCount(
			ctx,
			wct.info,
			&workflowservice.CountWorkflowExecutionsRequest{
				Namespace: wct.info.Namespace,
				Query:     query,
			},
			0,
		)
	}

	// Initial check
	lastError = performCheck()
	if lastError == nil {
		return nil
	}

	// Retry loop until context deadline
	for {
		select {
		case <-ctx.Done():
			// Context ended (deadline or cancellation). Return last error.
			return lastError
		case <-printTicker.C:
			wct.info.Logger.Infof("still waiting for running workflows to complete, retrying until deadline...")
		case <-checkTicker.C:
			// Don't perform check if context is already done
			if ctx.Err() != nil {
				return lastError
			}
			lastError = performCheck()
			if lastError == nil {
				return nil
			}
		}
	}
}
