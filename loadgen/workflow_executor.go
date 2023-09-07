package loadgen

import (
	"context"
	"fmt"
	"go.temporal.io/sdk/client"
)

// WorkflowExecutor is a very dumb executor that does nothing but run the specified workflow.
// Scenarios using it are expected to implement all their driver logic as a workflow.
type WorkflowExecutor struct {
	WorkflowType       string
	WorkflowArgCreator func(info ScenarioInfo) []interface{}
	StartOptsModifier  func(info ScenarioInfo, opts *client.StartWorkflowOptions)
}

func (w WorkflowExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	workflowOpts := client.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-driver", info.TaskQueue()),
		TaskQueue: info.TaskQueue(),
	}
	if w.StartOptsModifier != nil {
		w.StartOptsModifier(info, &workflowOpts)
	}
	run, err := info.Client.ExecuteWorkflow(
		ctx,
		workflowOpts,
		w.WorkflowType,
		w.WorkflowArgCreator(info)...,
	)
	if err != nil {
		return err
	}
	err = run.Get(ctx, nil)
	if err != nil {
		return err
	}
	return nil
}
