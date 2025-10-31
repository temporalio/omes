package loadgen

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
)

// InitSearchAttribute ensures that a search attribute is defined in the namespace.
// It creates the search attribute if it doesn't exist, or verifies it exists if it does.
// This implementation matches the throughput_stress initSearchAttribute behavior.
func InitSearchAttribute(
	ctx context.Context,
	info ScenarioInfo,
	attributeName string,
) error {
	_, err := info.Client.OperatorService().AddSearchAttributes(ctx,
		&operatorservice.AddSearchAttributesRequest{
			Namespace: info.Namespace,
			SearchAttributes: map[string]enums.IndexedValueType{
				attributeName: enums.INDEXED_VALUE_TYPE_KEYWORD,
			},
		})
	var deniedErr *serviceerror.PermissionDenied
	var alreadyErr *serviceerror.AlreadyExists
	if errors.As(err, &alreadyErr) {
		info.Logger.Infof("Search Attribute %q not added: already exists", attributeName)
	} else if err != nil {
		info.Logger.Warnf("Search Attribute %q not added: %v", attributeName, err)
		if !errors.As(err, &deniedErr) {
			return err
		}
	} else {
		info.Logger.Infof("Search Attribute %q added", attributeName)
	}

	return nil
}

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
	done := false

	check := func() error {
		visibilityCount, err := info.Client.CountWorkflow(timeoutCtx, request)
		if err != nil {
			return fmt.Errorf("failed to count workflows in visibility: %w", err)
		}
		lastVisibilityCount = visibilityCount.Count
		if lastVisibilityCount >= int64(minCount) {
			done = true
		}
		return nil
	}

	// Initial check before entering the loop.
	if err := check(); err != nil {
		return err
	}

	// Loop until we reach the desired count or timeout.
	for !done {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf(
				"expected at least %d workflows in visibility, got %d after waiting %v",
				minCount, lastVisibilityCount, waitAtMost,
			)

		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected at least: %d)\n",
				lastVisibilityCount, minCount)

		case <-countTicker.C:
			if err := check(); err != nil {
				return err
			}
		}
	}

	return nil
}

// GetNonCompletedWorkflows queries and returns details about non-completed workflows for debugging purposes.
// Returns a formatted string with up to the specified number of workflow details, or an error if the query fails.
func GetNonCompletedWorkflows(ctx context.Context, info ScenarioInfo, searchAttribute, runID string, limit int32) (string, error) {
	nonCompletedQuery := fmt.Sprintf(
		"%s='%s' AND ExecutionStatus != 'Completed'",
		searchAttribute,
		runID,
	)

	resp, err := info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     nonCompletedQuery,
		PageSize:  limit,
	})

	if err != nil {
		return "", fmt.Errorf("failed to list non-completed workflows: %w", err)
	}

	if len(resp.Executions) == 0 {
		return "", nil
	}

	var workflowDetails string
	for i, exec := range resp.Executions {
		workflowDetails += fmt.Sprintf("\n  %d. WorkflowID: %s, RunID: %s, Status: %s",
			i+1,
			exec.Execution.WorkflowId,
			exec.Execution.RunId,
			exec.Status.String())
	}
	return workflowDetails, nil
}

// VerifyNoFailedWorkflows verifies that there are no failed or terminated workflows for the given search attribute.
func VerifyNoFailedWorkflows(ctx context.Context, info ScenarioInfo, searchAttribute, runID string) error {
	var errors []string

	for _, status := range []enums.WorkflowExecutionStatus{
		enums.WORKFLOW_EXECUTION_STATUS_TERMINATED,
		enums.WORKFLOW_EXECUTION_STATUS_FAILED,
	} {
		statusQuery := fmt.Sprintf(
			"%s='%s' and ExecutionStatus = '%s'",
			searchAttribute, runID, status)
		visibilityCount, err := info.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query:     statusQuery,
		})
		if err != nil {
			errors = append(errors, fmt.Sprintf("failed to run query %q: %v", statusQuery, err))
			continue
		}
		if visibilityCount.Count > 0 {
			errors = append(errors, fmt.Sprintf("unexpected %d workflows with status %s", visibilityCount.Count, status))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("workflow verification failed: %s", strings.Join(errors, "; "))
	}
	return nil
}
