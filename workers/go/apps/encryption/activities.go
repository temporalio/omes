package encryption

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// Registered activity names.
const (
	echoActivityName             = "EncryptionEchoActivity"
	failActivityName             = "EncryptionFailActivity"
	cancelActivityName           = "EncryptionCancelActivity"
	heartbeatTimeoutActivityName = "EncryptionHeartbeatTimeoutActivity"
)

// Error types, so a failure names where it came from.
const (
	activityFailureErrorType     = "EncryptionActivityFailure"
	localActivityFailureType     = "EncryptionLocalActivityFailure"
	workflowFailureErrorType     = "EncryptionWorkflowFailure"
	nestedFailureErrorType       = "EncryptionNestedFailure"
	retryAttemptFailureErrorType = "EncryptionRetryAttemptFailure"
	updateRejectedErrorType      = "EncryptionUpdateRejected"
)

// heartbeatFlushDelay gives the SDK time to actually ship a heartbeat to the
// server before the activity returns; heartbeats are sent asynchronously, so an
// activity that heartbeats and immediately completes may never send one.
const heartbeatFlushDelay = 100 * time.Millisecond

// EchoActivity heartbeats and returns a result: the activity input, header,
// heartbeat details, and completion result paths in one activity.
func EchoActivity(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	activity.RecordHeartbeat(ctx, newHeartbeatDetails(input, 1))
	if err := sleepOrCancel(ctx, heartbeatFlushDelay); err != nil {
		return ActivityResult{}, err
	}
	return ActivityResult{Step: input.Step, Echoed: input.Message, Filler: input.Filler}, nil
}

// FailActivity heartbeats and then fails with details, and with a cause chain
// that carries details of its own so the failure nests.
func FailActivity(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	activity.RecordHeartbeat(ctx, newHeartbeatDetails(input, 1))
	if err := sleepOrCancel(ctx, heartbeatFlushDelay); err != nil {
		return ActivityResult{}, err
	}
	return ActivityResult{}, temporal.NewNonRetryableApplicationError(
		"activity failed on purpose",
		activityFailureErrorType,
		newFailureChain(input.Step, input.Message, input.Filler, input.FailureDepth),
		newFailureDetails(input),
	)
}

// CancelActivity heartbeats until it is canceled, then reports cancellation with
// details. The caller must set WaitForCancellation so the canceled details make it
// into history.
func CancelActivity(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	for attempt := int32(1); ; attempt++ {
		activity.RecordHeartbeat(ctx, newHeartbeatDetails(input, attempt))
		select {
		case <-ctx.Done():
			return ActivityResult{}, temporal.NewCanceledError(newFailureDetails(input))
		case <-time.After(heartbeatFlushDelay):
		}
	}
}

// HeartbeatTimeoutActivity heartbeats once and then stops, so the server times it
// out on its heartbeat timeout. This is the only path that puts heartbeat details
// into history, as TimeoutFailureInfo.LastHeartbeatDetails.
func HeartbeatTimeoutActivity(ctx context.Context, input ActivityInput) (ActivityResult, error) {
	activity.RecordHeartbeat(ctx, newHeartbeatDetails(input, 1))
	if err := sleepOrCancel(ctx, heartbeatTimeoutActivitySleep); err != nil {
		return ActivityResult{}, err
	}
	return ActivityResult{Step: input.Step, Echoed: input.Message, Filler: input.Filler}, nil
}

// FailingLocalActivity fails so the LocalActivity marker it records carries both
// details and a Failure.
func FailingLocalActivity(_ context.Context, input ActivityInput) (ActivityResult, error) {
	return ActivityResult{}, temporal.NewNonRetryableApplicationError(
		"local activity failed on purpose",
		localActivityFailureType,
		nil,
		newFailureDetails(input),
	)
}

func sleepOrCancel(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
}

func newHeartbeatDetails(input ActivityInput, attempt int32) HeartbeatDetails {
	return HeartbeatDetails{
		Step:    input.Step,
		Attempt: attempt,
		Message: input.Message,
		Filler:  input.Filler,
	}
}

func newFailureDetails(input ActivityInput) FailureDetails {
	return FailureDetails{
		Step:    input.Step,
		Message: input.Message,
		Filler:  input.Filler,
		Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
	}
}
