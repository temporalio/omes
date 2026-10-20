package loadgen

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type verificationCountClient struct {
	client.Client
	count func(*workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error)
}

func (c verificationCountClient) CountWorkflow(_ context.Context, req *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	return c.count(req)
}

func TestVerifyNoFailedWorkflowsClassifiesOutcomes(t *testing.T) {
	queryErr := serviceerror.NewUnavailable("visibility unavailable")
	for _, tc := range []struct {
		name       string
		failed     int64
		terminated int64
		queryErr   error
	}{
		{name: "healthy"},
		{name: "failed history", failed: 1},
		{name: "terminated histories", terminated: 2},
		{name: "query failure", queryErr: queryErr},
		{name: "failed history and query failure", failed: 1, queryErr: queryErr},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			info := ScenarioInfo{
				Namespace: "test-namespace",
				Client: verificationCountClient{count: func(req *workflowservice.CountWorkflowExecutionsRequest) (*workflowservice.CountWorkflowExecutionsResponse, error) {
					calls++
					require.Equal(t, "test-namespace", req.Namespace)
					require.Contains(t, req.Query, "OmesExecutionID='exec-42'")
					if strings.HasSuffix(req.Query, "'"+enums.WORKFLOW_EXECUTION_STATUS_TERMINATED.String()+"'") {
						return &workflowservice.CountWorkflowExecutionsResponse{Count: tc.terminated}, tc.queryErr
					}
					require.True(t, strings.HasSuffix(req.Query, "'"+enums.WORKFLOW_EXECUTION_STATUS_FAILED.String()+"'"))
					return &workflowservice.CountWorkflowExecutionsResponse{Count: tc.failed}, nil
				}},
			}
			err := VerifyNoFailedWorkflows(t.Context(), info, OmesExecutionIDSearchAttribute, "exec-42")
			require.Equal(t, 2, calls)
			if tc.failed+tc.terminated == 0 && tc.queryErr == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			if tc.failed+tc.terminated > 0 {
				require.ErrorIs(t, fmt.Errorf("scenario: %w", err), ErrUnexpectedWorkflowOutcomes)
				if tc.failed > 0 {
					require.ErrorContains(t, err, fmt.Sprintf("unexpected %d workflows with status Failed", tc.failed))
				} else {
					require.ErrorContains(t, err, fmt.Sprintf("unexpected %d workflows with status Terminated", tc.terminated))
				}
			} else {
				require.NotErrorIs(t, err, ErrUnexpectedWorkflowOutcomes)
			}
			if tc.queryErr != nil {
				require.ErrorIs(t, err, tc.queryErr)
			}
		})
	}
}
