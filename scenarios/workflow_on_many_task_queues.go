package scenarios

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow on one of the task queues. " +
			"Workers must be started with --task-queue-suffix-index-end as one less than task queue count here. " +
			"Additional options: task-queue-count (required).",
		Executor: loadgen.KitchenSinkExecutor{
			WorkflowParams: kitchensink.NewWorkflowParams(kitchensink.NopActionExecuteActivity),
			PrepareWorkflowParams: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.WorkflowParams) error {
				// Require task queue count
				if opts.ScenarioOptionInt("task-queue-count", 0) == 0 {
					return fmt.Errorf("task-queue-count option required")
				}
				return nil
			},
			UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
				// Add suffix to the task queue based on modulus of iteration
				options.StartOptions.TaskQueue +=
					fmt.Sprintf("-%v", run.Iteration%run.ScenarioInfo.ScenarioOptionInt("task-queue-count", 0))
				return nil
			},
		},
	})
}
