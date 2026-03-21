package scenarios

import (
	"time"

	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "SAW baseline for COGS: single workflow executing one payload activity.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							saaCogsSAWActionSet(),
						},
					},
				},
			}
		},
	})
}

func saaCogsSAWActionSet() *kitchensink.ActionSet {
	return &kitchensink.ActionSet{
		Actions: []*kitchensink.Action{
			{
				Variant: &kitchensink.Action_ExecActivity{
					ExecActivity: &kitchensink.ExecuteActivityAction{
						ActivityType: &kitchensink.ExecuteActivityAction_Payload{
							Payload: &kitchensink.ExecuteActivityAction_PayloadActivity{
								BytesToReceive: 256,
								BytesToReturn:  256,
							},
						},
						ScheduleToCloseTimeout: durationpb.New(60 * time.Second),
						RetryPolicy: &common.RetryPolicy{
							MaximumAttempts: 1,
						},
					},
				},
			},
			{
				Variant: &kitchensink.Action_ReturnResult{
					ReturnResult: &kitchensink.ReturnResultAction{
						ReturnThis: &common.Payload{},
					},
				},
			},
		},
	}
}
