package throughputstress

import (
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/throughputstress"
	"github.com/temporalio/omes/scenarios"
	"github.com/temporalio/omes/workers/go/workflowutils"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	UpdateSleep         = "sleep"
	UpdateActivity      = "activity"
	UpdateLocalActivity = "localActivity"
	ASignal             = "someSignal"
)

var activityStub = Activities{}

// ThroughputStressWorkflow is meant to mimic the throughputstress scenario from bench-go of days
// past, but in a less-opaque way. We do not ask why it is the way it is, it is not our place to
// question the inscrutable ways of the old code.
func ThroughputStressWorkflow(ctx workflow.Context, params *throughputstress.WorkflowParams) (throughputstress.WorkflowOutput, error) {
	output := throughputstress.WorkflowOutput{
		ChildrenSpawned: params.ChildrenSpawned,
		TimesContinued:  params.TimesContinued,
	}
	// Establish handlers
	err := workflow.SetQueryHandler(ctx, "myquery", func() (string, error) {
		return "hi", nil
	})
	if err != nil {
		return output, err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateSleep, func(ctx workflow.Context) error {
		return workflow.Sleep(ctx, 1*time.Second)
	})
	if err != nil {
		return output, err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateActivity, func(ctx workflow.Context) error {
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
		return workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
	})
	if err != nil {
		return output, err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateLocalActivity, func(ctx workflow.Context) error {
		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
		return workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
	})
	if err != nil {
		return output, err
	}
	signchan := workflow.GetSignalChannel(ctx, ASignal)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			signchan.Receive(ctx, nil)
		}
	})

	for i := params.InitialIteration; i < params.Iterations; i++ {
		// Repeat the steps as defined by the ancient ritual
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
		err = workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
		if err != nil {
			return output, err
		}

		err = workflow.ExecuteActivity(actCtx, activityStub.SelfQuery, "myquery").Get(ctx, nil)
		if err != nil {
			return output, err
		}

		err = workflow.ExecuteActivity(actCtx, activityStub.SelfDescribe).Get(ctx, nil)
		if err != nil {
			return output, err
		}

		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
		err = workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
		if err != nil {
			return output, err
		}
		err = workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
		if err != nil {
			return output, err
		}

		// Do some stuff in parallel
		err = workflowutils.RunConcurrently(ctx,
			func(ctx workflow.Context) error {
				// Make sure we pass through the search attribute that correlates us to a scenario
				// run to the child.
				attrs := workflow.GetInfo(ctx).SearchAttributes.IndexedFields
				childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
					SearchAttributes: map[string]interface{}{
						scenarios.ThroughputStressScenarioIdSearchAttribute: attrs[scenarios.ThroughputStressScenarioIdSearchAttribute],
					},
				})
				child := workflow.ExecuteChildWorkflow(childCtx, ThroughputStressChild)
				output.ChildrenSpawned++
				return child.Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(
					actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(
					actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(
					localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(
					localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				// This self-signal activity didn't exist in the original bench-go workflow, but
				// it was signaled semi-routinely externally. This is slightly more obvious and
				// introduces receiving signals in the workflow.
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(localActCtx, activityStub.SelfSignal, ASignal).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				if params.SkipSleep {
					return nil
				}
				return workflow.ExecuteActivity(actCtx, activityStub.SelfUpdate, UpdateSleep).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(actCtx, activityStub.SelfUpdate, UpdateActivity).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(actCtx, activityStub.SelfUpdate, UpdateLocalActivity).Get(ctx, nil)
			},
		)
		if err != nil {
			return output, err
		}
		// Possibly continue as new
		if params.ContinueAsNewAfterEventCount > 0 && workflow.GetInfo(ctx).GetCurrentHistoryLength() >= params.ContinueAsNewAfterEventCount {
			if i == 0 {
				return output, fmt.Errorf("trying to continue as new on first iteration, workflow will never end")
			}
			params.InitialIteration = i
			params.TimesContinued++
			params.ChildrenSpawned = output.ChildrenSpawned
			return output, workflow.NewContinueAsNewError(ctx, "throughputStress", params)
		}
	}

	return output, nil
}

func ThroughputStressChild(ctx workflow.Context) error {
	for i := 0; i < 3; i++ {
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
		err := workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func defaultActivityOpts() workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 5,
		},
	}
}

func defaultLocalActivityOpts() workflow.LocalActivityOptions {
	return workflow.LocalActivityOptions{
		StartToCloseTimeout: 3 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 10 * time.Millisecond,
			MaximumAttempts: 10,
		},
	}
}
