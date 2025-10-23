package scheduledworkflows

import (
	"context"

	"go.temporal.io/sdk/activity"
)

type SchedulerActivities struct{}

// DeleteScheduleActivity deletes a schedule by ID
func (a *SchedulerActivities) DeleteScheduleActivity(ctx context.Context, scheduleID string) error {
	return activity.GetClient(ctx).ScheduleClient().GetHandle(ctx, scheduleID).Delete(ctx)
}
