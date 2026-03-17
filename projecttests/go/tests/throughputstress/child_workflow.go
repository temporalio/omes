package throughputstress

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ChildWorkflow runs 3 remote payload activities and returns.
func ChildWorkflow(ctx workflow.Context, payload []byte) error {
	opts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	remoteCtx := workflow.WithActivityOptions(ctx, opts)

	for i := 0; i < 3; i++ {
		var result []byte
		if err := workflow.ExecuteActivity(remoteCtx, PayloadActivity, payload).Get(remoteCtx, &result); err != nil {
			return fmt.Errorf("child payload activity %d failed: %w", i, err)
		}
	}
	return nil
}
