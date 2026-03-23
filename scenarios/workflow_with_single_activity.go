package scenarios

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
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
						startWorkflowOptions(r),
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

func startWorkflowOptions(r *loadgen.Run) client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue: loadgen.TaskQueueForRun(r.RunID),
		ID: fmt.Sprintf(
			"w-%s-%s-%d",
			r.RunID,
			r.ExecutionID,
			r.Iteration,
		),
		WorkflowExecutionErrorWhenAlreadyStarted: !r.Configuration.IgnoreAlreadyStarted,
	}
}
