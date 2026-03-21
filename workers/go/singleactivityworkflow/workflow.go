package singleactivityworkflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func SingleActivityWorkflow(ctx workflow.Context, input []byte, outputSize int32) ([]byte, error) {
	var output []byte
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}), "payload", input, outputSize).Get(ctx, &output)
	if err != nil {
		return nil, err
	}
	return output, nil
}
