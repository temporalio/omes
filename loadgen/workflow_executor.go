package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
)

// WorkflowExecutor is a very dumb executor that does nothing but run the specified workflow.
// Scenarios using it are expected to implement all their driver logic as a workflow.
type WorkflowExecutor struct {
	WorkflowType       string
	WorkflowArgCreator func(info ScenarioInfo) []interface{}
}

func (w WorkflowExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	// TODO: ??
	//executorTimeout := info.ScenarioOptionDuration(scenarios.ExecutorTimeoutFlag, 1*time.Hour)
	executorTimeout := 1 * time.Hour
	workflowOpts := client.StartWorkflowOptions{
		ID:                       fmt.Sprintf("%s-driver", info.TaskQueue()),
		TaskQueue:                info.TaskQueue(), // TODO: Should be special tq for driver?
		WorkflowExecutionTimeout: executorTimeout,
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
