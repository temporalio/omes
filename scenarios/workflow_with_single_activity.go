package scenarios

import (
	"context"

	"github.com/temporalio/omes/scenario"
	"github.com/temporalio/omes/shared"
)

func Execute(ctx context.Context, run *scenario.Run) error {
	return run.ExecuteKitchenSinkWorkflow(ctx, &shared.KitchenSinkWorkflowParams{
		Actions: []*shared.KitchenSinkAction{{ExecuteActivity: &shared.ExecuteActivityAction{Name: "noop"}}},
	})
}

func init() {
	scenario.Register(&scenario.Scenario{
		Name:        "WorkflowWithSingleNoopActivity",
		Execute:     Execute,
		Concurrency: 5,
		Iterations:  10,
	})
}
