package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration follows a deterministic one-, two-, or three-activity path.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{},
				},
				UpdateWorkflowOptions: func(_ context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
					pathLength := run.Iteration%3 + 1
					actionSet := &kitchensink.ActionSet{}
					for i := 0; i < pathLength; i++ {
						actionSet.Actions = append(actionSet.Actions, kitchensink.NoOpSingleActivityActionSet().Actions[0])
					}
					actionSet.Actions = append(actionSet.Actions, kitchensink.NoOpSingleActivityActionSet().Actions[1])
					options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{actionSet}
					return nil
				},
			}
		},
	})
}
