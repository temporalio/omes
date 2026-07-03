package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration runs one bounded 64-KB, 25,000-write resource activity.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &TestInput{
					WorkflowInput: &WorkflowInput{
						InitialActions: ListActionSet(
							GenericActivity("matched_resource", DefaultRemoteActivity),
							NewEmptyReturnResultAction(),
						),
					},
				},
			}
		},
	})
}
