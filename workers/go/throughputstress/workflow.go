package throughputstress

import (
	"errors"
	"fmt"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
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
			func(ctx workflow.Context) error {
				return runEchoNexusOperation(ctx, params, EchoSyncOperation)
			},
			func(ctx workflow.Context) error {
				return runEchoNexusOperation(ctx, params, EchoAsyncOperation)
			},
			func(ctx workflow.Context) error {
				client := nexusClient(ctx, params)
				if client == nil {
					return nil
				}
				opCtx, cancel := workflow.WithCancel(ctx)
				defer cancel()
				fut := client.ExecuteOperation(
					opCtx,
					WaitForCancelOperation,
					nil,
					workflow.NexusOperationOptions{},
				)
				// Wait for the operation to be started before canceling it.
				if err := fut.GetNexusOperationExecution().Get(ctx, nil); err != nil {
					workflow.GetLogger(ctx).Error("could not start nexus operation", "error", err)
					return err
				}
				cancel()
				err := fut.Get(ctx, nil)
				if err == nil {
					return fmt.Errorf("expected operation to fail but it succeeded")
				}
				var canceledErr *temporal.CanceledError
				if !errors.As(err, &canceledErr) {
					return fmt.Errorf("expected operation to fail as canceled, instead it failed with: %w", err)
				}
				return nil
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

func nexusClient(ctx workflow.Context, params *throughputstress.WorkflowParams) workflow.NexusClient {
	if params.NexusEndpoint == "" {
		workflow.GetLogger(ctx).Info("not running nexus tests, set nexusEndpoint in options to enable these tests")
		return nil
	}
	return workflow.NewNexusClient(params.NexusEndpoint, ThroughputStressServiceName)
}

func runEchoNexusOperation(ctx workflow.Context, params *throughputstress.WorkflowParams, operation nexus.OperationReference[string, string]) error {
	client := nexusClient(ctx, params)
	if client == nil {
		return nil
	}
	fut := client.ExecuteOperation(
		ctx,
		operation,
		"hello",
		workflow.NexusOperationOptions{},
	)
	var output string
	if err := fut.Get(ctx, &output); err != nil {
		return err
	}
	if output != "hello" {
		return fmt.Errorf(`expected "hello", got %q`, output)
	}
	return nil
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
