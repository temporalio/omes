package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration starts one workflow, sends one signal, waits for the signaled state, and completes.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &TestInput{
					ClientSequence: ClientActions(NewSignalActionsWithIDs(1)...),
					WorkflowInput: &WorkflowInput{
						ExpectedSignalCount: 1,
						InitialActions: ListActionSet(
							NewAwaitWorkflowStateAction("signal_0", "received"),
							NewEmptyReturnResultAction(),
						),
					},
				},
			}
		},
	})
}
