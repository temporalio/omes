package kitchensink

import (
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestNexusStartWorkflowRequest(t *testing.T) {
	t.Parallel()

	request := NexusStartWorkflowRequest("workflow-id", &NexusWorkflowStartOptions{TaskQueue: "task-queue"})

	require.True(t, proto.Equal(&NexusOperationRequest{
		Action: &NexusOperationRequest_WorkflowAction{WorkflowAction: &NexusWorkflowAction{
			WorkflowId:   "workflow-id",
			StartOptions: &NexusWorkflowStartOptions{TaskQueue: "task-queue"},
			Action:       &NexusWorkflowAction_Start{Start: &emptypb.Empty{}},
		}},
	}, request))
}

func TestNexusSignalWorkflowRequest(t *testing.T) {
	t.Parallel()

	request := NexusSignalWorkflowRequest(
		"workflow-id",
		"run-id",
		&DoSignal{WithStart: true},
		&NexusWorkflowStartOptions{TaskQueue: "task-queue"},
	)

	require.True(t, proto.Equal(&NexusOperationRequest{
		Action: &NexusOperationRequest_WorkflowAction{WorkflowAction: &NexusWorkflowAction{
			WorkflowId:   "workflow-id",
			RunId:        "run-id",
			StartOptions: &NexusWorkflowStartOptions{TaskQueue: "task-queue"},
			Action:       &NexusWorkflowAction_Signal{Signal: &DoSignal{WithStart: true}},
		}},
	}, request))
}

func TestNexusUpdateWorkflowRequest(t *testing.T) {
	t.Parallel()

	request := NexusUpdateWorkflowRequest("workflow-id", "run-id", &DoUpdate{})

	require.True(t, proto.Equal(&NexusOperationRequest{
		Action: &NexusOperationRequest_WorkflowAction{WorkflowAction: &NexusWorkflowAction{
			WorkflowId: "workflow-id",
			RunId:      "run-id",
			Action:     &NexusWorkflowAction_Update{Update: &DoUpdate{}},
		}},
	}, request))
}
