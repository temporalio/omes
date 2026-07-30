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
			"Workers must be started with --task-queue-suffix-index-end as one less than task queue count here.",
		Options: func(o *loadgen.OptionSet) {
			o.Int("task-queue-count", 0, "Number of task queues to spread iterations across.")
			o.MarkRequired("task-queue-count")
		},
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							kitchensink.NoOpSingleActivityActionSet(),
						},
					},
				},
				UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
					// Add suffix to the task queue based on modulus of iteration
					options.StartOptions.TaskQueue +=
						fmt.Sprintf("-%v", run.Iteration%run.ScenarioInfo.OptionInt("task-queue-count"))
					return nil
				},
			}
		},
	})
}
