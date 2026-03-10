package ebbandflow

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type workflowActivityInput struct {
	SleepDurationNanos int64   `json:"sleepDurationNanos"`
	PriorityKey        int32   `json:"priorityKey,omitempty"`
	FairnessKey        string  `json:"fairnessKey,omitempty"`
	FairnessWeight     float32 `json:"fairnessWeight,omitempty"`
}

type workflowInput struct {
	Activities []workflowActivityInput `json:"activities"`
}

type workflowOutput struct {
	Timings []activityTiming `json:"timings"`
}

type activityTiming struct {
	ScheduleToStart time.Duration `json:"d"`
}

type activityExecutionResult struct {
	ScheduledTime   time.Time `json:"scheduledTime"`
	ActualStartTime time.Time `json:"actualStartTime"`
}

func Workflow(ctx workflow.Context, params workflowInput) (workflowOutput, error) {
	if len(params.Activities) == 0 {
		return workflowOutput{Timings: []activityTiming{}}, nil
	}

	futures := make([]workflow.Future, 0, len(params.Activities))
	for _, activity := range params.Activities {
		opts := workflow.ActivityOptions{
			StartToCloseTimeout: 1 * time.Minute,
			RetryPolicy:         &temporal.RetryPolicy{},
		}
		opts.Priority.PriorityKey = int(activity.PriorityKey)
		if activity.FairnessKey != "" {
			opts.Priority.FairnessKey = activity.FairnessKey
			weight := activity.FairnessWeight
			if weight <= 0 {
				weight = 1.0
			}
			opts.Priority.FairnessWeight = weight
		}

		actCtx := workflow.WithActivityOptions(ctx, opts)
		future := workflow.ExecuteActivity(actCtx, SleepActivity, activity.SleepDurationNanos)
		futures = append(futures, future)
	}

	timings := make([]activityTiming, 0, len(futures))
	for _, future := range futures {
		var result activityExecutionResult
		if err := future.Get(ctx, &result); err != nil {
			return workflowOutput{}, err
		}
		timings = append(timings, activityTiming{
			ScheduleToStart: result.ActualStartTime.Sub(result.ScheduledTime),
		})
	}
	return workflowOutput{Timings: timings}, nil
}

func SleepActivity(ctx context.Context, sleepDurationNanos int64) (activityExecutionResult, error) {
	if sleepDurationNanos < 0 {
		return activityExecutionResult{}, fmt.Errorf("sleep duration cannot be negative")
	}
	if sleepDurationNanos > 0 {
		time.Sleep(time.Duration(sleepDurationNanos))
	}
	info := activity.GetInfo(ctx)
	return activityExecutionResult{
		ScheduledTime:   info.ScheduledTime,
		ActualStartTime: info.StartedTime,
	}, nil
}
