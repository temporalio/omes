package scenarios

import (
	"math"
	"math/rand"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"
)

// This scenario is meant to be adjusted and run manually to evaluate the performance of different
// slot provider implementations

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

func parallelRandomDelays(
	numConccurrent int,
	minRuntime time.Duration,
	maxRuntime time.Duration) *kitchensink.ActionSet {
	actions := make([]*kitchensink.Action, numConccurrent)
	for i := 0; i < numConccurrent; i++ {
		randMs := rand.Float64()*float64(maxRuntime.Milliseconds()-minRuntime.Milliseconds()) + float64(minRuntime.Milliseconds())
		actions[i] = &kitchensink.Action{
			Variant: &kitchensink.Action_ExecActivity{
				ExecActivity: &kitchensink.ExecuteActivityAction{
					ActivityType: &kitchensink.ExecuteActivityAction_Delay{
						// Random duration
						Delay: durationpb.New(time.Duration(randMs) * time.Millisecond),
					},
					StartToCloseTimeout: &durationpb.Duration{Seconds: 30},
				},
			},
		}
	}
	return &kitchensink.ActionSet{
		Concurrent: true,
		Actions:    actions,
	}
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Used for testing slot provider performance. Runs activities that consume certain amounts of resources.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							// Add a short warm up so metrics can get emitted
							{
								Actions: []*kitchensink.Action{
									{
										Variant: &kitchensink.Action_ExecActivity{
											ExecActivity: &kitchensink.ExecuteActivityAction{
												ActivityType: &kitchensink.ExecuteActivityAction_Delay{
													Delay: durationpb.New(time.Second * 2),
												},
												StartToCloseTimeout: &durationpb.Duration{Seconds: 30},
											},
										},
									},
								},
							},

							// Essentially not bound. Hard to get node to actually use this memory.
							parallelResourcesActions(6, int(math.Pow(10, 9)), 10_000_000, 5, 30),

							// This section ends up being quite CPU bound.
							parallelResourcesActions(300, int(math.Pow(10, 8)), 1_000_000, 2, 5),

							// An IO bound scenario where the activities aren't doing much work and
							// mostly waiting
							parallelRandomDelays(1000, time.Millisecond*100, time.Second*5),

							// Add a short delay before finishing so metrics can get emitted
							{
								Actions: []*kitchensink.Action{
									{
										Variant: &kitchensink.Action_ExecActivity{
											ExecActivity: &kitchensink.ExecuteActivityAction{
												ActivityType: &kitchensink.ExecuteActivityAction_Delay{
													Delay: durationpb.New(time.Second * 5),
												},
												StartToCloseTimeout: &durationpb.Duration{Seconds: 30},
											},
										},
									},
								},
							},
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
			}
		},
	})
}
