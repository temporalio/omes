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
) error {

	countTicker := time.NewTicker(3 * time.Second)
	defer countTicker.Stop()

	printTicker := time.NewTicker(30 * time.Second)
	defer printTicker.Stop()

	var lastVisibilityCount int64

	check := func() error {
		visibilityCount, err := info.Client.CountWorkflow(ctx, request)
		if err != nil {
			return fmt.Errorf("failed to count workflows in visibility: %w", err)
		}
		lastVisibilityCount = visibilityCount.Count
		return nil
	}

	// Initial check
	if err := check(); err != nil {
		return err
	}
	if lastVisibilityCount >= int64(minCount) {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			// Context ended (deadline or cancellation). Return success only if min reached.
			if lastVisibilityCount >= int64(minCount) {
				return nil
			}
			return fmt.Errorf("expected at least %d workflows in visibility, got %d (context done)",
				minCount, lastVisibilityCount)
		case <-printTicker.C:
			info.Logger.Infof("current visibility count: %d (expected at least: %d)",
				lastVisibilityCount, minCount)
		case <-countTicker.C:
			if err := check(); err != nil {
				return err
			}
			if lastVisibilityCount >= int64(minCount) {
				return nil
			}
		}
	}
}

// GetNonCompletedWorkflows queries and returns an error for each non-completed workflow.
// Returns a list of errors (one per non-completed workflow) with workflow details, or a query error if the list fails.
func GetNonCompletedWorkflows(ctx context.Context, info ScenarioInfo, searchAttribute, runID string, limit int32) []error {
	nonCompletedQuery := fmt.Sprintf(
		"%s='%s' AND ExecutionStatus != 'Completed' AND ExecutionStatus != 'ContinuedAsNew'",
		searchAttribute,
		runID,
	)

	resp, err := info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     nonCompletedQuery,
		PageSize:  limit,
	})

	if err != nil {
		return []error{fmt.Errorf("failed to list non-completed workflows: %w", err)}
	}

	if len(resp.Executions) == 0 {
		return nil
	}

	var workflowErrors []error
	for _, exec := range resp.Executions {
		workflowErrors = append(workflowErrors, fmt.Errorf(
			"non-completed workflow: WorkflowID=%s, RunID=%s, Status=%s",
			exec.Execution.WorkflowId,
			exec.Execution.RunId,
			exec.Status.String()))
	}
	return workflowErrors
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
