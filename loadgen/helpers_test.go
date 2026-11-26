package loadgen

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	sdkmocks "go.temporal.io/sdk/mocks"
	"go.uber.org/zap"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestExportWorkflowHistoriesPaginates(t *testing.T) {
	ctx := context.Background()
	outputDir := t.TempDir()
	temporalClient := &sdkmocks.Client{}
	info := ScenarioInfo{
		ExecutionID: "export-test",
		Namespace:   "test-namespace",
		Logger:      zap.NewNop().Sugar(),
		Client:      temporalClient,
		ExportOptions: ExportOptions{
			ExportHistoriesDir:    outputDir,
			ExportHistoriesFilter: "failed",
		},
	}

	query := OmesExecutionIDSearchAttribute + "='export-test' AND ExecutionStatus = 'Failed'"
	nextPageToken := []byte("next-page")
	for i, workflowID := range []string{"first-workflow", "second-workflow"} {
		request := &workflowservice.ListWorkflowExecutionsRequest{Namespace: info.Namespace, Query: query}
		response := &workflowservice.ListWorkflowExecutionsResponse{
			Executions: []*workflow.WorkflowExecutionInfo{
				{Execution: &common.WorkflowExecution{WorkflowId: workflowID, RunId: "run-id"}},
			},
		}
		if i == 0 {
			response.NextPageToken = nextPageToken
		} else {
			request.NextPageToken = nextPageToken
		}
		temporalClient.On("ListWorkflow", ctx, request).Return(response, nil).Once()
		iterator := &sdkmocks.HistoryEventIterator{}
		iterator.On("HasNext").Return(true).Once()
		iterator.On("Next").Return(&history.HistoryEvent{EventId: int64(i + 1)}, nil).Once()
		iterator.On("HasNext").Return(false).Once()
		temporalClient.On("GetWorkflowHistory", ctx, workflowID, "run-id", false, enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT).
			Return(iterator).Once()
	}

	require.NoError(t, ExportWorkflowHistories(ctx, info))
	for i, workflowID := range []string{"first-workflow", "second-workflow"} {
		data, err := os.ReadFile(filepath.Join(outputDir, "histories-export-test", workflowID+"@run-id.json"))
		require.NoError(t, err)
		var exported history.History
		require.NoError(t, protojson.Unmarshal(data, &exported))
		require.Len(t, exported.Events, 1)
		require.EqualValues(t, i+1, exported.Events[0].EventId)
	}
	temporalClient.AssertExpectations(t)
}
