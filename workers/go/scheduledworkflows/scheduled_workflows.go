package scheduledworkflows

import (
	"hash/fnv"
	"time"

	"github.com/temporalio/omes/scenarios"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// NoopScheduledWorkflow is a simple workflow that does nothing and completes immediately.
// This is used for testing schedule creation and management overhead without workflow execution overhead.
func NoopScheduledWorkflow(ctx workflow.Context, payload []byte) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("NoopScheduledWorkflow started", "payloadSize", len(payload))
	return nil
}

// SleepScheduledWorkflow is a workflow that sleeps for a configurable duration.
// This is used for testing overlap policies when scheduled workflows take time to complete.
// The sleep duration is randomly chosen between 5-10 seconds.
func SleepScheduledWorkflow(ctx workflow.Context, payload []byte) error {
	logger := workflow.GetLogger(ctx)

	// Generate sleep duration between 5-10 seconds using hash bucketing on workflow run ID
	runID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	sleepDuration := getSleepDuration(runID)

	logger.Info("SleepScheduledWorkflow started", "sleepDuration", sleepDuration, "payloadSize", len(payload))

	// Sleep for the calculated duration
	workflow.Sleep(ctx, sleepDuration)

	logger.Info("SleepScheduledWorkflow completed", "sleepDuration", sleepDuration)
	return nil
}

// getSleepDuration uses hash bucketing to get value between 5-10 seconds
func getSleepDuration(runID string) time.Duration {
	h := fnv.New32a()
	h.Write([]byte(runID))
	hash := h.Sum32()
	randomSeconds := 5 + int(hash%6) // 0-5 range, add to 5 for 5-10 range
	return time.Duration(randomSeconds) * time.Second
}

// Activities for scheduled workflows if needed

var schedulerActivityStub = &SchedulerActivities{}

func CleanUpSchedulesWorkflow(ctx workflow.Context, input scenarios.CleanUpScheduleInput) error {
	logger := workflow.GetLogger(ctx)

	logger.Debug("Starting cleanup workflow for schedule",
		"scheduleID", input.ScheduleID,
		"deleteAfter", input.DeleteAfter)

	// Wait for the specified duration
	if input.DeleteAfter > 0 {
		logger.Debug("Waiting before cleanup", "waitDuration", input.DeleteAfter)
		workflow.Sleep(ctx, input.DeleteAfter)
	}

	// Delete the schedule
	logger.Debug("Deleting schedule", "scheduleID", input.ScheduleID)

	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	activityCtx := workflow.WithActivityOptions(ctx, activityOptions)

	err := workflow.ExecuteActivity(activityCtx, schedulerActivityStub.DeleteScheduleActivity, input.ScheduleID).Get(ctx, nil)
	if err != nil {
		logger.Error("Failed to delete schedule", "scheduleID", input.ScheduleID, "error", err)
		return err
	}

	logger.Debug("Successfully deleted schedule", "scheduleID", input.ScheduleID)
	return nil
}
