package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a noop activity.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							kitchensink.NoOpSingleActivityActionSet(),
						},
					},
				},
			}
		},
	})
}
