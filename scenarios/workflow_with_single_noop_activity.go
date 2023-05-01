package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func Execute(ctx context.Context, run *loadgen.Run) error {
	return run.ExecuteKitchenSinkWorkflow(ctx, &kitchensink.WorkflowParams{
		Actions: []*kitchensink.Action{{ExecuteActivity: &kitchensink.ExecuteActivityAction{Name: "noop"}}},
	})
}

func init() {
	loadgen.MustRegisterScenario(&loadgen.Scenario{
		Executor: &loadgen.SharedIterationsExecutor{
			Execute:     Execute,
			Concurrency: 5,
			Iterations:  10,
		},
	})
}
