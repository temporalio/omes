package scenarios

import (
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"math"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a noop activity.",
		Executor: loadgen.KitchenSinkExecutor{
			DefaultConfiguration: loadgen.RunConfiguration{
				Iterations:    1,
				MaxConcurrent: 1,
			},
			TestInput: &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{
						{Concurrent: true,
							Actions: []*kitchensink.Action{
								kitchensink.ResourceConsumingActivity(uint64(math.Pow(10, 9)), 1_000_000, 5),
								kitchensink.ResourceConsumingActivity(uint64(math.Pow(10, 9)), 1_000_000, 5),
								kitchensink.ResourceConsumingActivity(uint64(math.Pow(10, 9)), 1_000_000, 5),
								kitchensink.ResourceConsumingActivity(uint64(math.Pow(10, 9)), 1_000_000, 5),
								kitchensink.ResourceConsumingActivity(uint64(math.Pow(10, 9)), 1_000_000, 5),
							}},
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
