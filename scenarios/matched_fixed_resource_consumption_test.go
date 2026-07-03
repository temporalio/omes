package scenarios

import (
	"testing"

	"github.com/temporalio/omes/loadgen"
)

func TestMatchedFixedResourceConsumptionUsesOneMatchedActivity(t *testing.T) {
	scenario := loadgen.GetScenario("matched_fixed_resource_consumption")
	if scenario == nil {
		t.Fatal("matched_fixed_resource_consumption is not registered")
	}
	executor, ok := scenario.ExecutorFn().(loadgen.KitchenSinkExecutor)
	if !ok {
		t.Fatal("matched_fixed_resource_consumption does not use KitchenSinkExecutor")
	}
	actions := executor.TestInput.WorkflowInput.InitialActions
	if len(actions) != 1 || len(actions[0].Actions) != 2 {
		t.Fatalf("got action sets %#v, want one activity and one return", actions)
	}
	generic := actions[0].Actions[0].GetExecActivity().GetGeneric()
	if generic == nil || generic.Type != "matched_resource" {
		t.Fatalf("got generic activity %#v, want matched_resource", generic)
	}
	if actions[0].Actions[1].GetReturnResult() == nil {
		t.Fatal("matched resource workflow does not return a result")
	}
}
