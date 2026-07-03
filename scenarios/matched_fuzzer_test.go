package scenarios

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.uber.org/zap"
)

func TestMatchedFuzzerUsesDeterministicOneToThreeActivityPaths(t *testing.T) {
	scenario := loadgen.GetScenario("matched_fuzzer")
	if scenario == nil {
		t.Fatal("matched_fuzzer is not registered")
	}
	executor, ok := scenario.ExecutorFn().(loadgen.KitchenSinkExecutor)
	if !ok || executor.UpdateWorkflowOptions == nil {
		t.Fatal("matched_fuzzer does not provide dynamic KitchenSink paths")
	}
	info := loadgen.ScenarioInfo{Logger: zap.NewNop().Sugar()}
	for iteration, want := range []int{1, 2, 3, 1} {
		params := proto.Clone(executor.TestInput).(*kitchensink.TestInput)
		options := loadgen.KitchenSinkWorkflowOptions{Params: params}
		if err := executor.UpdateWorkflowOptions(context.Background(), info.NewRun(iteration), &options); err != nil {
			t.Fatal(err)
		}
		got := 0
		for _, actionSet := range options.Params.WorkflowInput.InitialActions {
			for _, action := range actionSet.Actions {
				if action.GetExecActivity() != nil {
					got++
				}
			}
		}
		if got != want {
			t.Fatalf("iteration %d got %d activities, want %d", iteration, got, want)
		}
	}
}
