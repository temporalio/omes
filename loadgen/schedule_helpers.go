package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
)

// ScheduleIDForRun returns a schedule ID for a given run ID and index
func ScheduleIDForRun(runID string, index int) string {
	return fmt.Sprintf("schedule-%s-%d", runID, index)
}

// WaitForScheduleCompletion polls a schedule until it has completed all actions
func WaitForScheduleCompletion(ctx context.Context, client client.Client, scheduleID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for schedule %s to complete: %w", scheduleID, ctx.Err())
		case <-ticker.C:
			handle := client.ScheduleClient().GetHandle(ctx, scheduleID)
			desc, err := handle.Describe(ctx)
			if err != nil {
				return fmt.Errorf("failed to describe schedule %s: %w", scheduleID, err)
			}

			// Schedule is complete when RemainingActions is 0 and no workflows are running
			if desc.Schedule.State.RemainingActions == 0 && len(desc.Info.RunningWorkflows) == 0 {
				return nil
			}
		}
	}
}

// DeleteSchedulesForRun deletes all schedules for a given run ID
func DeleteSchedulesForRun(ctx context.Context, client client.Client, runID string, count int) error {
	for i := 0; i < count; i++ {
		scheduleID := ScheduleIDForRun(runID, i)
		handle := client.ScheduleClient().GetHandle(ctx, scheduleID)
		if err := handle.Delete(ctx); err != nil {
			return fmt.Errorf("failed to delete schedule %s: %w", scheduleID, err)
		}
	}
	return nil
}
