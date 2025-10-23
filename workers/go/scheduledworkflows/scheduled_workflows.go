package scheduledworkflows

import (
	"hash/fnv"
	"time"

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

type SchedulerOrchestrationInput struct {
	RunID                 string
	Iteration             int
	WaitTimeBeforeCleanup time.Duration
}

type SchedulerOrchestrationResult struct {
	SchedulesDeleted int
	TotalDuration    time.Duration
}

// Signals for orchestration workflow
const (
	ScheduleIDsSignal = "schedule-ids"
	DeleteSignal      = "delete-schedules"
)

type ScheduleIDsSignalData struct {
	ScheduleIDs []string
	StartTime   time.Time
}

func SchedulerOrchestrationWorkflow(ctx workflow.Context, input SchedulerOrchestrationInput) (SchedulerOrchestrationResult, error) {
	logger := workflow.GetLogger(ctx)
	startTime := workflow.Now(ctx)

	logger.Info("Starting scheduler orchestration workflow",
		"runID", input.RunID,
		"iteration", input.Iteration,
		"waitTimeBeforeCleanup", input.WaitTimeBeforeCleanup)

	var scheduleIDs []string
	var scheduleStartTime time.Time
	var deleteRequested bool

	// Set up signal handlers
	scheduleIDsChannel := workflow.GetSignalChannel(ctx, ScheduleIDsSignal)
	deleteChannel := workflow.GetSignalChannel(ctx, DeleteSignal)

	result := SchedulerOrchestrationResult{}

	// Wait for signals
	for !deleteRequested {
		selector := workflow.NewSelector(ctx)

		// Handle schedule IDs signal
		selector.AddReceive(scheduleIDsChannel, func(c workflow.ReceiveChannel, more bool) {
			var signalData ScheduleIDsSignalData
			c.Receive(ctx, &signalData)
			logger.Info("Received schedule IDs", "count", len(signalData.ScheduleIDs), "startTime", signalData.StartTime)
			scheduleIDs = append(scheduleIDs, signalData.ScheduleIDs...)
			// Store the start time from the first signal
			if scheduleStartTime.IsZero() {
				scheduleStartTime = signalData.StartTime
			}
		})

		// Handle delete signal
		selector.AddReceive(deleteChannel, func(c workflow.ReceiveChannel, more bool) {
			var signal interface{}
			c.Receive(ctx, &signal)
			logger.Info("Received delete signal, will cleanup after wait time", "totalSchedules", len(scheduleIDs))
			deleteRequested = true
		})

		selector.Select(ctx)
	}

	// Wait until start time + WaitTimeBeforeCleanup before deleting schedules
	if !scheduleStartTime.IsZero() && input.WaitTimeBeforeCleanup > 0 {
		cleanupTime := scheduleStartTime.Add(input.WaitTimeBeforeCleanup)
		currentTime := workflow.Now(ctx)
		if cleanupTime.After(currentTime) {
			waitDuration := cleanupTime.Sub(currentTime)
			logger.Info("Waiting before cleanup", "waitDuration", waitDuration, "cleanupTime", cleanupTime)
			workflow.Sleep(ctx, waitDuration)
		}
	}

	// Delete all schedules
	if len(scheduleIDs) > 0 {
		logger.Info("Deleting schedules", "count", len(scheduleIDs))
		deletedCount, err := deleteSchedules(ctx, scheduleIDs)
		if err != nil {
			logger.Warn("Some schedule deletions failed", "error", err)
		}
		result.SchedulesDeleted = deletedCount
	} else {
		logger.Info("No schedules to delete")
	}

	result.TotalDuration = workflow.Now(ctx).Sub(startTime)
	logger.Info("Scheduler orchestration completed",
		"totalDuration", result.TotalDuration,
		"schedulesDeleted", result.SchedulesDeleted)

	return result, nil
}

func deleteSchedules(ctx workflow.Context, scheduleIDs []string) (int, error) {
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

	// Start all activities concurrently
	futures := make([]workflow.Future, len(scheduleIDs))
	for i, scheduleID := range scheduleIDs {
		futures[i] = workflow.ExecuteActivity(activityCtx, schedulerActivityStub.DeleteScheduleActivity, scheduleID)
	}

	// Wait for all activities to complete and count successes
	deletedCount := 0
	for i, future := range futures {
		err := future.Get(activityCtx, nil)
		if err != nil {
			workflow.GetLogger(ctx).Warn("Delete schedule activity failed", "scheduleID", scheduleIDs[i], "error", err)
		} else {
			deletedCount++
		}
	}

	return deletedCount, nil
}
