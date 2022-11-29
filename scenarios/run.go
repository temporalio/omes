package scenarios

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Run represents a scenario run
type Run struct {
	// Used for Workflow ID and task queue name generation
	ID       string
	Scenario *Scenario
	Client   client.Client
	// Each call to the execute method gets a distinct `ItertionInTest`
	IterationInTest uint32
	Logger          *zap.SugaredLogger
}

func (r *Run) WorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                TaskQueueForRunID(r.Scenario, r.ID),
		ID:                                       fmt.Sprintf("w-%s-%d", r.ID, r.IterationInTest),
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}

// ExecuteVoidWorkflow starts a workflow and waits for its completion ignoring its result
func (r *Run) ExecuteVoidWorkflow(ctx context.Context, name string, args ...interface{}) error {
	// TODO: ctx deadline might be too short if scenario is run with the Duration option.
	// Set different duration here.
	opts := r.WorkflowOptions()
	r.Logger.Debugf("Executing workflow with options: %v", opts)
	execution, err := r.Client.ExecuteWorkflow(ctx, opts, name, args...)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, nil); err != nil {
		return fmt.Errorf("error executing workflow (ID: %s, run ID: %s): %v", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
