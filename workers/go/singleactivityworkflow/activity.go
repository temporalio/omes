package singleactivityworkflow

import (
	"context"
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// PayloadWithRetries behaves like "payload" but fails with a retryable error
// on attempts 1..failForAttempts, then succeeds on the next attempt.
func PayloadWithRetries(ctx context.Context, inputData []byte, bytesToReturn int32, failForAttempts int32) ([]byte, error) {
	if activity.GetInfo(ctx).Attempt <= failForAttempts {
		return nil, temporal.NewApplicationError(
			fmt.Sprintf("deliberate failure (attempt %d of %d)", activity.GetInfo(ctx).Attempt, failForAttempts),
			"RetryableError", nil,
		)
	}
	output := make([]byte, bytesToReturn)
	copy(output, inputData)
	return output, nil
}
