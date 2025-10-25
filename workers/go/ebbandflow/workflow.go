package ebbandflow

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen/ebbandflow"
	"github.com/temporalio/omes/workers/go/workflowutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var activityStub = Activities{}

// EbbAndFlowTrackWorkflow executes activities and returns their schedule-to-start times with fairness data
func EbbAndFlowTrackWorkflow(ctx workflow.Context, params *ebbandflow.WorkflowParams) (*ebbandflow.WorkflowOutput, error) {
	rng := rand.New(rand.NewSource(workflow.Now(ctx).UnixNano()))
	activities := params.SleepActivities.Sample(rng)

	if len(activities) == 0 {
		return &ebbandflow.WorkflowOutput{Timings: []ebbandflow.ActivityTiming{}}, nil
	}

	var results []ebbandflow.ActivityTiming
	var resultsMutex sync.Mutex

	var activityFuncs []func(workflow.Context) error
	for _, activity := range activities {
		activityFuncs = append(activityFuncs, func(ctx workflow.Context) error {
			// Set up activity options
			opts := workflow.ActivityOptions{
				StartToCloseTimeout: 1 * time.Minute,
				RetryPolicy:         &temporal.RetryPolicy{},
			}

			// Set priority, if specified
			if activity.Priority != nil {
				opts.Priority.PriorityKey = int(activity.Priority.PriorityKey)
			}

			// Set fairness, if specified
			fairnessKey := activity.GetFairnessKey()
			fairnessWeight := activity.GetFairnessWeight()
			if fairnessKey != "" {
				opts.Priority.FairnessKey = fairnessKey
				opts.Priority.FairnessWeight = fairnessWeight
			}

			// Execute activity
			var activityResult ActivityExecutionResult
			actCtx := workflow.WithActivityOptions(ctx, opts)
			err := workflow.ExecuteActivity(actCtx, activityStub.MeasureLatencyActivity, activity).Get(ctx, &activityResult)
			if err != nil {
				workflow.GetLogger(ctx).Error("Activity execution failed", "error", err)
				return err
			}

			// Calculate schedule-to-start time using accurate activity timing
			scheduleToStartMS := activityResult.ActualStartTime.Sub(activityResult.ScheduledTime)

			result := ebbandflow.ActivityTiming{
				ScheduleToStart: scheduleToStartMS,
			}

			// Thread-safe append to results
			resultsMutex.Lock()
			results = append(results, result)
			resultsMutex.Unlock()

			return nil
		})
	}

	err := workflowutils.RunConcurrently(ctx, activityFuncs...)
	if err != nil {
		workflow.GetLogger(ctx).Error("Failed to execute activities concurrently", "error", err)
	}

	// Check if all activities failed
	if len(results) == 0 {
		return nil, fmt.Errorf("failed to start any of the %d activities", len(activities))
	}

	return &ebbandflow.WorkflowOutput{Timings: results}, nil
}
