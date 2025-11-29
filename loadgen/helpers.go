package loadgen

import (
	"context"
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
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
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

// ExportWorkflowHistories exports workflow histories based on filter settings.
func ExportWorkflowHistories(ctx context.Context, info ScenarioInfo) error {
	if info.ExportOptions.ExportHistoriesDir == "" {
		return nil // Export disabled
	}

	filter := info.ExportOptions.ExportHistoriesFilter
	if filter == "" {
		filter = "all" // Default
	}

	// Create run-specific output directory
	outputDir := filepath.Join(
		info.ExportOptions.ExportHistoriesDir,
		fmt.Sprintf("histories-%s", info.ExecutionID),
	)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create histories directory: %w", err)
	}

	query := fmt.Sprintf("%s='%s'", OmesExecutionIDSearchAttribute, info.ExecutionID)
	query += buildStatusFilter(filter, info.Logger)

	resp, err := info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     query,
	})
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	if len(resp.Executions) == 0 {
		info.Logger.Info("No workflows found matching export filter")
		return nil
	}

	var exportErrors []string
	for _, execution := range resp.Executions {
		if err := exportSingleWorkflowHistory(ctx, info.Client, execution, outputDir); err != nil {
			exportErrors = append(exportErrors, fmt.Sprintf("%s: %v", execution.Execution.WorkflowId, err))
		}
	}

	info.Logger.Infof("Exported %d workflow histories to %s",
		len(resp.Executions)-len(exportErrors), outputDir)

	if len(exportErrors) > 0 {
		return fmt.Errorf("%d workflow history exports failed: %s", len(exportErrors), strings.Join(exportErrors, "\n "))
	}
	return nil
}

func buildStatusFilter(filter string, logger *zap.SugaredLogger) string {
	switch filter {
	case "failed":
		return " AND ExecutionStatus = 'Failed'"
	case "terminated":
		return " AND ExecutionStatus = 'Terminated'"
	case "failed,terminated", "terminated,failed":
		return " AND (ExecutionStatus = 'Failed' OR ExecutionStatus = 'Terminated')"
	case "all":
		return "" // No status filter
	default:
		logger.Errorf("Unexpected status filter: %s", filter)
		return ""
	}
}

func exportSingleWorkflowHistory(
	ctx context.Context,
	c client.Client,
	execution *workflow.WorkflowExecutionInfo,
	outputDir string,
) error {
	workflowID := execution.Execution.WorkflowId
	runID := execution.Execution.RunId

	historyIter := c.GetWorkflowHistory(ctx, workflowID, runID, false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var events []*history.HistoryEvent
	for historyIter.HasNext() {
		event, err := historyIter.Next()
		if err != nil {
			return fmt.Errorf("failed to read history event: %w", err)
		}
		events = append(events, event)
	}

	history := &history.History{Events: events}
	marshaler := protojson.MarshalOptions{
		Indent: "  ",
	}
	data, err := marshaler.Marshal(history)
	if err != nil {
		return fmt.Errorf("failed to marshal history: %w", err)
	}

	// Sanitize workflow ID for filename (i.e. for child workflows)
	safeWorkflowID := strings.ReplaceAll(workflowID, "/", "_")
	filename := filepath.Join(outputDir, fmt.Sprintf("%s-%s.json", safeWorkflowID, runID))

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %w", err)
	}

	return nil
}
