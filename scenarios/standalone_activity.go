package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a standalone activity. The activity takes in some bytes and returns some bytes. " +
			"It never retries or heartbeats.",
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, r *loadgen.Run) error {
					payloadSize := r.ScenarioOptionInt("payload-size", 0)
					handle, err := r.Client.ExecuteActivity(
						ctx,
						activityOptions(r),
						"payload",
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

func activityOptions(r *loadgen.Run) client.StartActivityOptions {
	return client.StartActivityOptions{
		ID: fmt.Sprintf(
			"a-%s-%s-%d",
			r.RunID,
			r.ExecutionID,
			r.Iteration,
		),
		TaskQueue:           r.TaskQueue(),
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}
}
