package ebbandflow

import (
	"context"
	"time"

	"go.temporal.io/sdk/activity"

	"github.com/temporalio/omes/loadgen/kitchensink"
)

type Activities struct{}

type ActivityExecutionResult struct {
	ScheduledTime   time.Time `json:"scheduledTime"`
	ActualStartTime time.Time `json:"actualStartTime"`
}

func (a Activities) MeasureLatencyActivity(
	ctx context.Context,
	activityAction *kitchensink.ExecuteActivityAction,
) (ActivityExecutionResult, error) {
	if delay := activityAction.GetDelay(); delay != nil {
		time.Sleep(delay.AsDuration())
	}

	activityInfo := activity.GetInfo(ctx)
	return ActivityExecutionResult{
		ScheduledTime:   activityInfo.ScheduledTime,
		ActualStartTime: activityInfo.StartedTime,
	}, nil
}
