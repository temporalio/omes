package singleactivityworkflow

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

func SingleActivityWorkflow(ctx workflow.Context, input []byte, outputSize int32, failForAttempts int32) ([]byte, error) {
	activityName := "payload"
	var args []any
	if failForAttempts > 0 {
		activityName = "payloadWithRetries"
		args = []any{input, outputSize, failForAttempts}
	} else {
		args = []any{input, outputSize}
	}
	var output []byte
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts:    failForAttempts + 1,
			InitialInterval:    1 * time.Millisecond,
			BackoffCoefficient: 1.0,
		},
	}), activityName, args...).Get(ctx, &output)
	if err != nil {
		return nil, err
	}
	return output, nil
}
