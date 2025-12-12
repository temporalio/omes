package schedulerstress

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

// NoopScheduledWorkflow accepts a byte payload and completes immediately.
// This workflow is designed to test scheduler throughput with minimal workflow execution overhead.
func NoopScheduledWorkflow(ctx workflow.Context, payload []byte) error {
	return nil
}

// SleepScheduledWorkflow accepts a byte payload and sleeps for 1 second.
// This workflow is designed to test scheduler behavior with longer-running workflows.
func SleepScheduledWorkflow(ctx workflow.Context, payload []byte) error {
	return workflow.Sleep(ctx, 1*time.Second)
}
