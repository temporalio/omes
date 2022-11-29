package scenarios

import (
	"context"

	"github.com/temporalio/omes/shared"
)

func Execute(ctx context.Context, run *Run) error {
	return run.ExecuteVoidWorkflow(ctx, "kitchenSink", shared.KitchenSinkWorkflowParams{
		Actions: []*shared.KitchenSinkAction{{ExecuteActivity: &shared.ExecuteActivityAction{Name: "noop"}}},
	})
}

func init() {
	registerScenario(Scenario{
		Name:        "WorkflowWithSingleNoopActivity",
		Execute:     Execute,
		Concurrency: 5,
		Iterations:  10,
	})
}
