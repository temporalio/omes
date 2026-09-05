package kitchensink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
)

func TestNexusWorkflowOptionsAlwaysSetExecutionTimeout(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name      string
		input     *ks.NexusWorkflowAction
		requestID string
		expected  string
	}{
		{name: "RequestID", input: &ks.NexusWorkflowAction{}, requestID: "request-id", expected: "request-id"},
		{name: "WorkflowID", input: &ks.NexusWorkflowAction{WorkflowId: "workflow-id"}, requestID: "request-id", expected: "workflow-id"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			options := nexusWorkflowOptions(testCase.input, testCase.requestID)

			require.Equal(t, testCase.expected, options.ID)
			require.Equal(t, 60*time.Minute, options.WorkflowExecutionTimeout)
		})
	}
}
