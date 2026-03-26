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
					failForAttempts := r.ScenarioOptionInt("fail-for-attempts", 0)
					activityName := "payload"
					var args []any
					if failForAttempts > 0 {
						activityName = "payloadWithRetries"
						args = []any{make([]byte, payloadSize), int32(payloadSize), int32(failForAttempts)}
					} else {
						args = []any{make([]byte, payloadSize), int32(payloadSize)}
					}
					handle, err := r.Client.ExecuteActivity(
						ctx,
						activityOptions(r, int32(failForAttempts)),
						activityName,
						args...,
					)
					if err != nil {
						return err
					}
					getCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
					defer cancel()
					return handle.Get(getCtx, nil)
				},
			}
		},
	})
}

func activityOptions(r *loadgen.Run, failForAttempts int32) client.StartActivityOptions {
	return client.StartActivityOptions{
		ID: fmt.Sprintf(
			"a-%s-%s-%d",
			r.RunID,
			r.ExecutionID,
			r.Iteration,
		),
		TaskQueue:           r.TaskQueue(),
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: 60 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    failForAttempts + 1,
			InitialInterval:    1 * time.Millisecond,
			BackoffCoefficient: 1.0,
		},
	}
}
