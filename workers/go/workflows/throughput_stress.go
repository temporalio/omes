package workflows

import (
	"github.com/temporalio/omes/loadgen/througput_stress"
	"github.com/temporalio/omes/workers/go/activities"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"time"
)

// Recreating:
// 	return createInfiniteRepetition(
//		createSequence(
//			createRemote(0, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes), 10),
//			createQuery(1),
//			createDescribe(1),
//			createLocal(0, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes)),
//			createLocal(0, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes)),
//			createParallel(
//				createChildWorkflow("simple1", 7*86400, false, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes), false),
//				createRemote(0.00, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes), 10),
//				createRemote(0.00, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes), 10),
//				createLocal(0, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes)),
//				createLocal(0, 0, zeroWait, getPayloadSizeGenerator(payloadSizeInBytes), getPayloadSizeGenerator(payloadSizeInBytes)),
//				createUpdate(),
//				createUpdate(),
//			),
//			// If you dig in here, you might wonder how this workflow ever completes or makes progress, since after each
//			// iteration it's going to wait on this long timer. The answer is that the bench driver signal-with-starts the
//			// configdriven workflows, and if the `Count` parameter is sufficiently high, then this workflow will get
//			// periodically signalled via that mechanism. To abort this loop cleanly, send the `AbortInfiniteLoopSignalName` signal
//			// - which the configdriven monitor does by default upon completion.
//			createSignalWait(signalWait),
//			createContinueAsNew(continueAsNewStepThreshold, continueAsNewDurationThresholdInSeconds, 1000000),
//			createSendStats(),
//		),
//	)

const (
	UpdateSleep         = "sleep"
	UpdateActivity      = "activity"
	UpdateLocalActivity = "localActivity"
)

// ThroughputStressWorkflow is meant to mimic the throughputstress scenario from bench-go of days
// past, but in a less-opaque way. We do not ask why it is the way it is, it is not our place to
// question the inscrutable ways of the old code.
func ThroughputStressWorkflow(ctx workflow.Context, params *througput_stress.WorkflowParams) (string, error) {
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

	// Repeat the steps as defined by the ancient ritual TODO: Loop
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
