package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a noop activity.",
		Executor: loadgen.KitchenSinkExecutor{
			WorkflowParams: kitchensink.NewWorkflowParams(kitchensink.NopActionExecuteActivity),
		},
	})
}
