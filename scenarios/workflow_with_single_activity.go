package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a single-activity workflow. It takes in some bytes, passes them to an " +
			"activity, and returns the bytes returned by the activity. The activity never retries or heartbeats.",
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, r *loadgen.Run) error {
					payloadSize := r.ScenarioOptionInt("payload-size", 0)
					handle, err := r.Client.ExecuteWorkflow(
						ctx,
						r.DefaultStartWorkflowOptions(),
						"singleActivityWorkflow",
						make([]byte, payloadSize),
						int32(payloadSize),
					)
					if err != nil {
						return err
					}
					return handle.Get(ctx, nil)
				},
			}
		},
	})
}
