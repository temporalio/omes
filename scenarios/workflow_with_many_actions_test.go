package scenarios

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.uber.org/zap"
)

func TestWorkflowWithManyActionsUsesUniqueChildIDsPerIteration(t *testing.T) {
	scenario := loadgen.GetScenario("workflow_with_many_actions")
	executor, ok := scenario.ExecutorFn().(loadgen.KitchenSinkExecutor)
	if !ok {
		t.Fatal("workflow_with_many_actions does not use KitchenSinkExecutor")
	}
	info := loadgen.ScenarioInfo{
		RunID:           "run-1",
		ExecutionID:     "execution-1",
		ScenarioOptions: map[string]string{"children-per-workflow": "2", "activities-per-workflow": "0"},
		Logger:          zap.NewNop().Sugar(),
	}
	if err := executor.PrepareTestInput(context.Background(), info, executor.TestInput); err != nil {
		t.Fatal(err)
	}
	if executor.UpdateWorkflowOptions == nil {
		t.Fatal("workflow_with_many_actions does not update child IDs per iteration")
	}
	params := proto.Clone(executor.TestInput).(*kitchensink.TestInput)
	options := loadgen.KitchenSinkWorkflowOptions{Params: params}
	if err := executor.UpdateWorkflowOptions(context.Background(), info.NewRun(7), &options); err != nil {
		t.Fatal(err)
	}

	var childIDs []string
	for _, actionSet := range options.Params.WorkflowInput.InitialActions {
		for _, action := range actionSet.Actions {
			if child := action.GetExecChildWorkflow(); child != nil {
				childIDs = append(childIDs, child.WorkflowId)
			}
		}
	}
	want := []string{"run-1-execution-1-7-child-wf-0", "run-1-execution-1-7-child-wf-1"}
	if len(childIDs) != len(want) {
		t.Fatalf("got child IDs %v, want %v", childIDs, want)
	}
	for index := range want {
		if childIDs[index] != want[index] {
			t.Fatalf("got child IDs %v, want %v", childIDs, want)
		}
	}
}
