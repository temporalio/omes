package scenarios

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"

	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "SAA for COGS: standalone activity with payload, no workflow.",
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: executeSAA,
			}
		},
	})
}

func executeSAA(ctx context.Context, run *loadgen.Run) error {
	inputData := make([]byte, 256)
	handle, err := run.Client.ExecuteActivity(ctx, client.StartActivityOptions{
		ID:                     fmt.Sprintf("a-%s-%s-%d", run.RunID, run.ExecutionID, run.Iteration),
		TaskQueue:              run.TaskQueue(),
		ScheduleToCloseTimeout: 60 * time.Second,
		RetryPolicy:            &temporal.RetryPolicy{MaximumAttempts: 1},
	}, "payload", inputData, int32(256))
	if err != nil {
		return fmt.Errorf("failed to start standalone activity: %w", err)
	}
	var result []byte
	return handle.Get(ctx, &result)
}
