package kitchensink

import (
	"context"
	"errors"
	"fmt"

	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
)

// CreateFiniteSchedule creates a schedule with three timed actions and one immediate action.
func (ca *ClientActivities) CreateFiniteSchedule(ctx context.Context, input ks.FiniteScheduleInput) error {
	queue := activity.GetInfo(ctx).TaskQueue
	options, err := ks.FiniteScheduleOptions(input, queue)
	if err != nil {
		return temporal.NewNonRetryableApplicationError("invalid finite schedule input", "InvalidFiniteScheduleInput", err)
	}
	schedules := ca.Client.ScheduleClient()
	_, err = schedules.Create(ctx, options)
	if err == nil {
		return nil
	}
	if !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return fmt.Errorf("creating finite schedule %s: %w", input.ScheduleID, err)
	}
	existing, err := schedules.GetHandle(ctx, input.ScheduleID).Describe(ctx)
	if err != nil {
		return fmt.Errorf("reconciling finite schedule %s: %w", input.ScheduleID, err)
	}
	if !ks.MatchesFiniteSchedule(existing, input.ScheduleID, queue) {
		return temporal.NewNonRetryableApplicationError("existing finite schedule differs from requested identity or configuration", "FiniteScheduleCollision", nil)
	}
	return nil
}
