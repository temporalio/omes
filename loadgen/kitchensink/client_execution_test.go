package kitchensink

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	workflowpb "go.temporal.io/api/workflow/v1"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
)

type executionClient struct {
	client.Client
	runs    []string
	updates []string
}

func (c *executionClient) SignalWorkflow(_ context.Context, _, runID, _ string, _ interface{}) error {
	c.runs = append(c.runs, runID)
	return nil
}

func (c *executionClient) QueryWorkflow(_ context.Context, _, runID, _ string, _ ...interface{}) (converter.EncodedValue, error) {
	c.runs = append(c.runs, runID)
	return nil, nil
}

func (c *executionClient) DescribeWorkflowExecution(_ context.Context, _, runID string) (*workflowservice.DescribeWorkflowExecutionResponse, error) {
	c.runs = append(c.runs, runID)
	return &workflowservice.DescribeWorkflowExecutionResponse{WorkflowExecutionInfo: &workflowpb.WorkflowExecutionInfo{}}, nil
}

func (c *executionClient) UpdateWorkflow(_ context.Context, opts client.UpdateWorkflowOptions) (client.WorkflowUpdateHandle, error) {
	c.runs = append(c.runs, opts.RunID)
	c.updates = append(c.updates, opts.UpdateID)
	return completedUpdate{}, nil
}

type completedUpdate struct{ client.WorkflowUpdateHandle }

func (completedUpdate) Get(context.Context, interface{}) error { return nil }

func selfUpdate() *ClientAction {
	return &ClientAction{Variant: &ClientAction_DoUpdate{DoUpdate: &DoUpdate{
		Variant: &DoUpdate_DoActions{DoActions: &DoActionsUpdate{}},
	}}}
}

func TestClientExecutionBindsAllActionsAndDeduplicatesUpdates(t *testing.T) {
	seq := ClientActions(
		&ClientAction{Variant: &ClientAction_DoSignal{DoSignal: &DoSignal{
			Variant: &DoSignal_Custom{Custom: &HandlerInvocation{Name: "signal"}},
		}}},
		&ClientAction{Variant: &ClientAction_DoQuery{DoQuery: &DoQuery{
			Variant: &DoQuery_Custom{Custom: &HandlerInvocation{Name: "query"}},
		}}},
		&ClientAction{Variant: &ClientAction_DoDescribe{DoDescribe: &DoDescribe{}}},
		selfUpdate(),
		&ClientAction{Variant: &ClientAction_NestedActions{NestedActions: &ClientActionSet{Actions: []*ClientAction{selfUpdate()}}}},
	)
	original := proto.Clone(seq)
	c := &executionClient{}
	for range 2 {
		e := &ClientActionsExecutor{Client: c, Execution: &ClientExecution{RunID: "original", UpdateIDScope: "original/activity-7"}}
		require.NoError(t, e.ExecuteClientSequence(t.Context(), seq))
	}
	require.Len(t, c.runs, 10)
	for _, run := range c.runs {
		require.Equal(t, "original", run)
	}
	require.NotEmpty(t, c.updates[0])
	require.NotEqual(t, c.updates[0], c.updates[1])
	require.Equal(t, c.updates[:2], c.updates[2:])
	require.True(t, proto.Equal(original, seq), "preparing a retry must not mutate the workflow payload")
}

func TestClientExecutionValidatesNestedSequenceBeforeAnyRPC(t *testing.T) {
	for _, variant := range []string{"update-with-start", "signal-with-start", "wait-rebind", "unknown"} {
		t.Run(variant, func(t *testing.T) {
			set := &ClientActionSet{}
			switch variant {
			case "update-with-start":
				a := selfUpdate()
				a.GetDoUpdate().WithStart = true
				set.Actions = []*ClientAction{a}
			case "signal-with-start":
				set.Actions = []*ClientAction{{Variant: &ClientAction_DoSignal{DoSignal: &DoSignal{WithStart: true}}}}
			case "wait-rebind":
				set.WaitForCurrentRunToFinishAtEnd = true
			case "unknown":
				set.Actions = []*ClientAction{nil}
			}
			seq := ClientActions(selfUpdate(), &ClientAction{Variant: &ClientAction_NestedActions{NestedActions: set}})
			c := &executionClient{}
			e := &ClientActionsExecutor{Client: c, Execution: &ClientExecution{RunID: "original", UpdateIDScope: "activity"}}
			require.Error(t, e.ExecuteClientSequence(t.Context(), seq))
			require.Empty(t, c.runs)
		})
	}
}

func TestClientExecutionLegacyRouting(t *testing.T) {
	c := &executionClient{}
	e := &ClientActionsExecutor{Client: c}
	require.NoError(t, e.ExecuteClientSequence(t.Context(), ClientActions(selfUpdate())))
	require.Equal(t, []string{""}, c.runs)
	require.Equal(t, []string{""}, c.updates)
}

func TestClientExecutionCannotStart(t *testing.T) {
	e := &ClientActionsExecutor{Execution: &ClientExecution{RunID: "original", UpdateIDScope: "activity"}}
	require.ErrorContains(t, e.Start(t.Context(), nil), "cannot start workflows")
}
