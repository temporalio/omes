package throughputstress

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type ThroughputStressInput struct {
	InternalIterations     int           `json:"internal_iterations"`
	ContinueAsNewAfterIter int           `json:"continue_as_new_after_iter"`
	SleepDuration          time.Duration `json:"sleep_duration"`
	IncludeRetryScenarios  bool          `json:"include_retry_scenarios"`
	NexusEndpoint          string        `json:"nexus_endpoint"`
	PayloadSizeBytes       int           `json:"payload_size_bytes"`
	// Continue-as-new state
	CompletedIterations int `json:"completed_iterations"`
}

func ThroughputStressWorkflow(ctx workflow.Context, input ThroughputStressInput) error {
	logger := workflow.GetLogger(ctx)
	wfInfo := workflow.GetInfo(ctx)

	localOpts := workflow.LocalActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	localCtx := workflow.WithLocalActivityOptions(ctx, localOpts)

	remoteOpts := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
	}
	remoteCtx := workflow.WithActivityOptions(ctx, remoteOpts)

	// Set up query handler
	queryResult := "ok"
	err := workflow.SetQueryHandler(ctx, "status", func() (string, error) {
		return queryResult, nil
	})
	if err != nil {
		return err
	}

	// Set up update handler
	err = workflow.SetUpdateHandler(ctx, "update-with-timer", func(ctx workflow.Context) (string, error) {
		if err := workflow.Sleep(ctx, input.SleepDuration); err != nil {
			return "", err
		}
		return "update-timer-done", nil
	})
	if err != nil {
		return err
	}
	err = workflow.SetUpdateHandler(ctx, "update-with-payload", func(ctx workflow.Context, payload []byte) (string, error) {
		return fmt.Sprintf("received-%d-bytes", len(payload)), nil
	})
	if err != nil {
		return err
	}

	// Signal channel for self-signals
	signalCh := workflow.GetSignalChannel(ctx, "self-signal")

	payload := makePayload(input.PayloadSizeBytes)

	iterationsThisRun := input.InternalIterations
	if input.ContinueAsNewAfterIter > 0 {
		remaining := input.InternalIterations - input.CompletedIterations
		if remaining < iterationsThisRun {
			iterationsThisRun = remaining
		}
	}

	for i := 0; i < iterationsThisRun; i++ {
		currentIter := input.CompletedIterations + i
		logger.Info("Starting iteration", "iteration", currentIter)

		// === Sequential local activities ===
		for j := 0; j < 3; j++ {
			var result []byte
			if err := workflow.ExecuteLocalActivity(localCtx, PayloadActivity, payload).Get(localCtx, &result); err != nil {
				return fmt.Errorf("local activity %d failed at iter %d: %w", j, currentIter, err)
			}
		}

		// === Self-query via client activity ===
		if err := workflow.ExecuteActivity(remoteCtx, ClientQueryActivity, wfInfo.WorkflowExecution.ID, wfInfo.WorkflowExecution.RunID).Get(remoteCtx, nil); err != nil {
			return fmt.Errorf("self-query failed at iter %d: %w", currentIter, err)
		}

		// === Parallel section ===
		var futures []workflow.Future

		// Child workflow
		childOpts := workflow.ChildWorkflowOptions{
			WorkflowID: fmt.Sprintf("%s-child-%d", wfInfo.WorkflowExecution.ID, currentIter),
		}
		childCtx := workflow.WithChildOptions(ctx, childOpts)
		futures = append(futures, workflow.ExecuteChildWorkflow(childCtx, ChildWorkflow, payload))

		// Remote activities
		futures = append(futures,
			workflow.ExecuteActivity(remoteCtx, PayloadActivity, payload),
			workflow.ExecuteActivity(remoteCtx, PayloadActivity, payload),
		)

		// More local activities
		futures = append(futures,
			workflow.ExecuteLocalActivity(localCtx, PayloadActivity, payload),
			workflow.ExecuteLocalActivity(localCtx, PayloadActivity, payload),
		)

		// Self-query via client activity
		futures = append(futures,
			workflow.ExecuteActivity(remoteCtx, ClientQueryActivity, wfInfo.WorkflowExecution.ID, wfInfo.WorkflowExecution.RunID),
		)

		// Self-signal via client activity (signal + timer wait)
		futures = append(futures,
			workflow.ExecuteActivity(remoteCtx, ClientSignalActivity, wfInfo.WorkflowExecution.ID, "self-signal"),
		)

		// Self-update via client activities
		futures = append(futures,
			workflow.ExecuteActivity(remoteCtx, ClientUpdateActivity, wfInfo.WorkflowExecution.ID, "update-with-timer", nil),
			workflow.ExecuteActivity(remoteCtx, ClientUpdateActivity, wfInfo.WorkflowExecution.ID, "update-with-payload", payload),
		)

		// Wait for all parallel futures
		for fi, f := range futures {
			if err := f.Get(ctx, nil); err != nil {
				return fmt.Errorf("parallel future %d failed at iter %d: %w", fi, currentIter, err)
			}
		}

		// Consume the self-signal
		signalCh.Receive(ctx, nil)

		// === Optional retry scenarios ===
		if input.IncludeRetryScenarios {
			retryOpts := workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    100 * time.Millisecond,
					BackoffCoefficient: 1.0,
					MaximumAttempts:    3,
				},
			}
			retryCtx := workflow.WithActivityOptions(ctx, retryOpts)

			if err := workflow.ExecuteActivity(retryCtx, RetryableErrorActivity).Get(retryCtx, nil); err != nil {
				return fmt.Errorf("retryable error activity failed at iter %d: %w", currentIter, err)
			}

			timeoutOpts := workflow.ActivityOptions{
				StartToCloseTimeout: 2 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    100 * time.Millisecond,
					BackoffCoefficient: 1.0,
					MaximumAttempts:    3,
				},
			}
			timeoutCtx := workflow.WithActivityOptions(ctx, timeoutOpts)

			if err := workflow.ExecuteActivity(timeoutCtx, TimeoutActivity).Get(timeoutCtx, nil); err != nil {
				return fmt.Errorf("timeout activity failed at iter %d: %w", currentIter, err)
			}

			heartbeatOpts := workflow.ActivityOptions{
				StartToCloseTimeout: 30 * time.Second,
				HeartbeatTimeout:    5 * time.Second,
				RetryPolicy: &temporal.RetryPolicy{
					InitialInterval:    100 * time.Millisecond,
					BackoffCoefficient: 1.0,
					MaximumAttempts:    3,
				},
			}
			heartbeatCtx := workflow.WithActivityOptions(ctx, heartbeatOpts)

			if err := workflow.ExecuteActivity(heartbeatCtx, HeartbeatActivity).Get(heartbeatCtx, nil); err != nil {
				return fmt.Errorf("heartbeat activity failed at iter %d: %w", currentIter, err)
			}
		}

		// Check for continue-as-new
		if input.ContinueAsNewAfterIter > 0 && (i+1)%input.ContinueAsNewAfterIter == 0 && currentIter+1 < input.InternalIterations {
			newInput := input
			newInput.CompletedIterations = currentIter + 1
			return workflow.NewContinueAsNewError(ctx, ThroughputStressWorkflow, newInput)
		}
	}

	return nil
}

func makePayload(size int) []byte {
	if size <= 0 {
		size = 256
	}
	b := make([]byte, size)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}
