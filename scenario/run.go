package scenario

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/kitchensink"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Run represents a scenario run.
type Run struct {
	// Used for Workflow ID and task queue name generation.
	ID string
	// The scenario used for the run.
	Scenario *Scenario
	// Temporal client to use for executing workflows, etc.
	Client client.Client
	// Each call to the Execute method gets a distinct `IterationInTest`.
	IterationInTest int
	Logger          *zap.SugaredLogger
}

// WorkflowOptions returns a default set of options that can be used in any scenario.
func (r *Run) WorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                r.Scenario.TaskQueueForRunID(r.ID),
		ID:                                       fmt.Sprintf("w-%s-%d", r.ID, r.IterationInTest),
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}

// ExecuteKitchenSinkWorkflow starts the generic "kitchen sink" workflow and waits for its completion ignoring its result.
func (r *Run) ExecuteKitchenSinkWorkflow(ctx context.Context, params *kitchensink.WorkflowParams) error {
	// TODO: ctx deadline might be too short if scenario is run with the Duration option.
	// Set different duration here.
	opts := r.WorkflowOptions()
	r.Logger.Debugf("Executing workflow with options: %v", opts)
	execution, err := r.Client.ExecuteWorkflow(ctx, opts, "kitchenSink", params)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, nil); err != nil {
		return fmt.Errorf("error executing workflow (ID: %s, run ID: %s): %w", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
