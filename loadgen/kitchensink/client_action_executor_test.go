package kitchensink

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNexusStartWorkflowRequest(t *testing.T) {
	t.Parallel()

	request := NexusStartWorkflowRequest("workflow-id", &NexusWorkflowStartOptions{TaskQueue: "task-queue"})

	require.Equal(t, "workflow-id", request.GetWorkflowAction().GetWorkflowId())
	require.Equal(t, "task-queue", request.GetWorkflowAction().GetStartOptions().GetTaskQueue())
	require.NotNil(t, request.GetWorkflowAction().GetStart())
}
