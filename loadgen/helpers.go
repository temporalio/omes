package loadgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
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

// WorkflowHistoryExport represents the exported workflow history data.
type WorkflowHistoryExport struct {
	WorkflowID string                        `json:"workflowId"`
	RunID      string                        `json:"runId"`
	Status     enums.WorkflowExecutionStatus `json:"status"`
	StartTime  time.Time                     `json:"startTime"`
	CloseTime  time.Time                     `json:"closeTime"`
	Events     []*history.HistoryEvent       `json:"events"`
}

// ExportFailedWorkflowHistories exports histories of failed/terminated workflows to JSON files.
func ExportFailedWorkflowHistories(ctx context.Context, info ScenarioInfo, searchAttribute, runID, outputDir string) error {
	// Create output directory for failed histories
	historiesDir := filepath.Join(outputDir, fmt.Sprintf("failed-histories-%s", runID))
	if err := os.MkdirAll(historiesDir, 0755); err != nil {
		return fmt.Errorf("failed to create histories directory: %w", err)
	}

	// Query for failed and terminated workflows
	statusQuery := fmt.Sprintf(
		"%s='%s' and (ExecutionStatus = 'Failed' or ExecutionStatus = 'Terminated')",
		searchAttribute, runID)

	// List failed/terminated workflows
	resp, err := info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     statusQuery,
	})
	if err != nil {
		return fmt.Errorf("failed to list failed workflows: %w", err)
	}

	var exportErrors []string
	totalExported := 0
	// Export each workflow history
	for _, execution := range resp.Executions {
		if err := exportSingleWorkflowHistory(ctx, info.Client, execution, historiesDir); err != nil {
			exportErrors = append(exportErrors, fmt.Sprintf("failed to export %s: %v", execution.Execution.WorkflowId, err))
		} else {
			totalExported++
		}
	}

	if totalExported > 0 {
		info.Logger.Infof("Exported %d failed workflow histories to %s", totalExported, historiesDir)
	} else {
		info.Logger.Info("No failed workflows found to export")
	}

	if len(exportErrors) > 0 {
		return fmt.Errorf("some exports failed: %s", strings.Join(exportErrors, "; "))
	}
	return nil
}

// exportSingleWorkflowHistory exports a single workflow's history to a JSON file.
func exportSingleWorkflowHistory(
	ctx context.Context,
	c client.Client,
	execution *workflow.WorkflowExecutionInfo,
	outputDir string,
) error {
	workflowID := execution.Execution.WorkflowId
	runID := execution.Execution.RunId

	// Fetch full workflow history
	historyIter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			return fmt.Errorf("failed to read history event: %w", err)
		}
		events = append(events, event)
	}

	// Create export structure
	export := WorkflowHistoryExport{
		WorkflowID: workflowID,
		RunID:      runID,
		Status:     execution.Status,
		StartTime:  execution.StartTime.AsTime(),
		CloseTime:  execution.CloseTime.AsTime(),
		Events:     events,
	}

	// Write to JSON file
	filename := filepath.Join(outputDir, fmt.Sprintf("workflow-%s-%s.json", workflowID, runID))
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}
