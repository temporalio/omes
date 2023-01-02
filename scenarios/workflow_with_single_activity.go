package scenarios

import (
	"context"

	"github.com/temporalio/omes/kitchensink"
	"github.com/temporalio/omes/scenario"
)

func Execute(ctx context.Context, run *scenario.Run) error {
	return run.ExecuteKitchenSinkWorkflow(ctx, &kitchensink.WorkflowParams{
		Actions: []*kitchensink.Action{{ExecuteActivity: &kitchensink.ExecuteActivityAction{Name: "noop"}}},
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
