package workflows

import (
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/throughput_stress"
	"github.com/temporalio/omes/workers/go/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const (
	UpdateSleep         = "sleep"
	UpdateActivity      = "activity"
	UpdateLocalActivity = "localActivity"
	ASignal             = "someSignal"
)

// ThroughputStressWorkflow is meant to mimic the throughputstress scenario from bench-go of days
// past, but in a less-opaque way. We do not ask why it is the way it is, it is not our place to
// question the inscrutable ways of the old code.
func ThroughputStressWorkflow(ctx workflow.Context, params *throughput_stress.WorkflowParams) (string, error) {
	// Establish handlers
	err := workflow.SetQueryHandler(ctx, "myquery", func() (string, error) {
		return "hi", nil
	})
	if err != nil {
		return "", err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateSleep, func(ctx workflow.Context) error {
		return workflow.Sleep(ctx, 1*time.Second)
	})
	if err != nil {
		return "", err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateActivity, func(ctx workflow.Context) error {
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
		return workflow.ExecuteActivity(
			actCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
	})
	if err != nil {
		return "", err
	}
	err = workflow.SetUpdateHandler(ctx, UpdateLocalActivity, func(ctx workflow.Context) error {
		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
		return workflow.ExecuteLocalActivity(
			localActCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
	})
	if err != nil {
		return "", err
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
			actCtx, "Payload", activities.MakePayloadInput(256, 256)).Get(ctx, nil)
		if err != nil {
			return "", err
		}

		err = workflow.ExecuteActivity(actCtx, "SelfQuery", "myquery").Get(ctx, nil)
		if err != nil {
			return "", err
		}

		err = workflow.ExecuteActivity(actCtx, "SelfDescribe").Get(ctx, nil)
		if err != nil {
			return "", err
		}

		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
		err = workflow.ExecuteLocalActivity(
			localActCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
		if err != nil {
			return "", err
		}
		err = workflow.ExecuteLocalActivity(
			localActCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
		if err != nil {
			return "", err
		}

		// Do some stuff in parallel
		err = RunConcurrently(ctx,
			func(ctx workflow.Context) error {
				child := workflow.ExecuteChildWorkflow(ctx, "throughputStressChild")
				return child.Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(
					actCtx, "Payload", activities.MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(
					actCtx, "Payload", activities.MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(
					localActCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(
					localActCtx, "Payload", activities.MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				// This self-signal activity didn't exist in the original bench-go workflow, but
				// it was signaled semi-routinely externally. This is slightly more obvious and
				// introduces receiving signals in the workflow.
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts())
				return workflow.ExecuteLocalActivity(localActCtx, "SelfSignal", ASignal).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(actCtx, "SelfUpdate", UpdateSleep).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(actCtx, "SelfUpdate", UpdateActivity).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
				return workflow.ExecuteActivity(actCtx, "SelfUpdate", UpdateLocalActivity).Get(ctx, nil)
			},
		)
		if err != nil {
			return "", err
		}
		// Possibly continue as new
		if workflow.GetInfo(ctx).GetCurrentHistoryLength() >= params.ContinueAsNewAfterEventCount {
			if i == 0 {
				return "", fmt.Errorf("trying to continue as new on first iteration, workflow will never end")
			}
			params.InitialIteration = i
			return "", workflow.NewContinueAsNewError(ctx, "throughputStress", params)
		}
	}

	return "hi", nil
}

func ThroughputStressChild(ctx workflow.Context) error {
	for i := 0; i < 3; i++ {
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts())
		err := workflow.ExecuteActivity(
			actCtx, "Payload", activities.MakePayloadInput(256, 256)).Get(ctx, nil)
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
