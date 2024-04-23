package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"math"
)

func parallelResourcesActions(
	numConccurrent int,
	bytesToAlloc int,
	cpuYieldEveryNIters int,
	cpuYieldForMs int,
	runForSeconds int64) *kitchensink.ActionSet {
	actions := make([]*kitchensink.Action, numConccurrent)
	for i := 0; i < numConccurrent; i++ {
		actions[i] = kitchensink.ResourceConsumingActivity(
			uint64(bytesToAlloc),
			uint32(cpuYieldEveryNIters),
			uint32(cpuYieldForMs),
			runForSeconds)
	}
	return &kitchensink.ActionSet{
		Concurrent: true,
		Actions:    actions,
	}
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Used for testing slot provider performance. Runs activities that consume certain amounts of resources.",
		Executor: loadgen.KitchenSinkExecutor{
			DefaultConfiguration: loadgen.RunConfiguration{
				Iterations:    1,
				MaxConcurrent: 1,
			},
			TestInput: &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{
						parallelResourcesActions(6, int(math.Pow(10, 9)), 10_000_000, 5, 30),
						parallelResourcesActions(300, int(math.Pow(10, 8)), 1_000_000, 2, 5),
						{
							Actions: []*kitchensink.Action{

								{
									Variant: &kitchensink.Action_ReturnResult{
										ReturnResult: &kitchensink.ReturnResultAction{
											ReturnThis: &common.Payload{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	})
}
