package throughputstress

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ChildWorkflow runs 3 payload activities and returns.
func ChildWorkflow(ctx workflow.Context, payload []byte) error {
	opts := workflow.LocalActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	localCtx := workflow.WithLocalActivityOptions(ctx, opts)

	for i := 0; i < 3; i++ {
		var result []byte
		if err := workflow.ExecuteLocalActivity(localCtx, PayloadActivity, payload).Get(localCtx, &result); err != nil {
			return fmt.Errorf("child payload activity %d failed: %w", i, err)
		}
	}
	return nil
}
