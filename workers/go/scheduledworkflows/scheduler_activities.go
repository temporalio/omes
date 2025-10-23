package scheduledworkflows

import (
	"context"
	"errors"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
)

type SchedulerActivities struct{}

// DeleteScheduleActivity deletes a schedule by ID
func (a *SchedulerActivities) DeleteScheduleActivity(ctx context.Context, scheduleID string) error {
	err := activity.GetClient(ctx).ScheduleClient().GetHandle(ctx, scheduleID).Delete(ctx)
	if err != nil {
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			// Return nil if schedule is not found (already deleted or never existed)
			return nil
		}
	}
	return err
}
