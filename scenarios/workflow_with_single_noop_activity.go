package scenarios

import (
	"context"

	"github.com/temporalio/omes/omes"
	"github.com/temporalio/omes/omes/kitchensink"
)

func Execute(ctx context.Context, run *omes.Run) error {
	return run.ExecuteKitchenSinkWorkflow(ctx, &kitchensink.WorkflowParams{
		Actions: []*kitchensink.Action{{ExecuteActivity: &kitchensink.ExecuteActivityAction{Name: "noop"}}},
	})
}

func init() {
	omes.MustRegisterScenario(&omes.Scenario{
		Executor: &omes.SharedIterationsExecutor{
			Execute:     Execute,
			Concurrency: 5,
			Iterations:  10,
		},
	})
}
