package kitchensink

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestOriginalStartSpecRoundTrip(t *testing.T) {
	makeExecutor := func() *ClientActionsExecutor {
		return &ClientActionsExecutor{Namespace: "ns", WorkflowType: "kitchenSink", WorkflowInput: &WorkflowInput{ExpectedSignalCount: 7}, WorkflowOptions: client.StartWorkflowOptions{ID: "original", TaskQueue: "queue"}, RecoverOriginalStart: true}
	}
	first := makeExecutor()
	a := &WithStartClientAction{Variant: &WithStartClientAction_DoUpdate{DoUpdate: &DoUpdate{WithStart: true, Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}}}}}
	spec, err := first.PrepareOriginalStart(a)
	require.NoError(t, err)
	data, err := json.Marshal(spec)
	require.NoError(t, err)
	var persisted OriginalStartSpec
	require.NoError(t, json.Unmarshal(data, &persisted))
	restored := makeExecutor()
	require.NoError(t, restored.RestoreOriginalStart(persisted))
	require.True(t, proto.Equal(first.WorkflowInput, restored.WorkflowInput))
	again, err := restored.PrepareOriginalStart(nil)
	require.NoError(t, err)
	require.Equal(t, spec, again)
	foreign := makeExecutor()
	foreign.WorkflowOptions.TaskQueue = "foreign"
	require.Error(t, foreign.RestoreOriginalStart(spec))
	require.Nil(t, foreign.originalStart)
}
