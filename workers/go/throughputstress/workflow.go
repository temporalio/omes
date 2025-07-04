package throughputstress

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
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
	AQuery              = "myquery"
)

var activityStub = Activities{}

// ThroughputStressWorkflow is meant to mimic the throughputstress scenario from bench-go of days
// past, but in a less-opaque way. We do not ask why it is the way it is, it is not our place to
// question the inscrutable ways of the old code.
func ThroughputStressWorkflow(ctx workflow.Context, params *throughputstress.WorkflowParams) (*throughputstress.WorkflowOutput, error) {
	// Establish handlers
	if err := workflow.SetQueryHandler(ctx, AQuery, func() (string, error) {
		return "hi", nil
	}); err != nil {
		return nil, err
	}

	if err := workflow.SetUpdateHandler(ctx, UpdateSleep, func(ctx workflow.Context) error {
		return workflow.Sleep(ctx, 1*time.Second)
	}); err != nil {
		return nil, err
	}

	if err := workflow.SetUpdateHandler(ctx, UpdateActivity, func(ctx workflow.Context) error {
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
		return workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
	}); err != nil {
		return nil, err
	}

	if err := workflow.SetUpdateHandler(ctx, UpdateLocalActivity, func(ctx workflow.Context) error {
		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts(ctx))
		return workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
	}); err != nil {
		return nil, err
	}

	signalChan := workflow.GetSignalChannel(ctx, ASignal)
	workflow.Go(ctx, func(ctx workflow.Context) {
		for {
			selector := workflow.NewSelector(ctx)
			selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, nil)
			})
			selector.Select(ctx)
		}
	})

	for i := params.InitialIteration; i < params.Iterations; i++ {
		// Repeat the steps as defined by the ancient ritual
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
		if err := workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil); err != nil {
			return nil, err
		}

		if err := workflow.ExecuteActivity(actCtx, activityStub.SelfQuery, AQuery).Get(ctx, nil); err != nil {
			return nil, err
		}

		if err := workflow.ExecuteActivity(actCtx, activityStub.SelfDescribe).Get(ctx, nil); err != nil {
			return nil, err
		}

		localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts(ctx))
		if err := workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil); err != nil {
			return nil, err
		}
		if err := workflow.ExecuteLocalActivity(
			localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil); err != nil {
			return nil, err
		}

		// Do some stuff in parallel
		if err := workflowutils.RunConcurrently(ctx,
			func(ctx workflow.Context) error {
				// Make sure we pass through the search attribute that correlates us to a scenario
				// run to the child.
				attrs := workflow.GetInfo(ctx).SearchAttributes.IndexedFields
				childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
					WorkflowExecutionTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout,
					WorkflowID:               fmt.Sprintf("%s/child-%d", workflow.GetInfo(ctx).WorkflowExecution.ID, params.ChildrenSpawned),
					SearchAttributes: map[string]interface{}{
						scenarios.ThroughputStressScenarioIdSearchAttribute: attrs[scenarios.ThroughputStressScenarioIdSearchAttribute],
					},
				})
				child := workflow.ExecuteChildWorkflow(childCtx, ThroughputStressChild)
				params.ChildrenSpawned++
				return child.Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
				return workflow.ExecuteActivity(
					actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
				return workflow.ExecuteActivity(
					actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts(ctx))
				return workflow.ExecuteLocalActivity(
					localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts(ctx))
				return workflow.ExecuteLocalActivity(
					localActCtx, activityStub.Payload, MakePayloadInput(0, 256)).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				// This self-signal activity didn't exist in the original bench-go workflow, but
				// it was signaled semi-routinely externally. This is slightly more obvious and
				// introduces receiving signals in the workflow.
				localActCtx := workflow.WithLocalActivityOptions(ctx, defaultLocalActivityOpts(ctx))
				return workflow.ExecuteLocalActivity(localActCtx, activityStub.SelfSignal, ASignal).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				// This activity simulates custom traffic patterns by generating activities that sleep
				// for a specified duration, with optional priority and fairness keys.
				if params.SleepActivities == nil {
					return nil
				}
				sleepInputs := params.SleepActivities.Sample()

				var sleepFuncs []func(workflow.Context) error
				for _, actInput := range sleepInputs {
					sleepFuncs = append(sleepFuncs, func(ctx workflow.Context) error {
						opts := defaultActivityOpts(ctx)
						opts.Priority.PriorityKey = int(actInput.GetPriorityKey())
						if actInput.FairnessKey != "" {
							// TODO(fairness): hack until there is a fairness key in the SDK
							opts.ActivityID = fmt.Sprintf("x-temporal-internal-fairness-key[%s:%f]-%s",
								actInput.GetFairnessKey(), actInput.GetFairnessWeight(), uuid.New().String())
						}
						actCtx := workflow.WithActivityOptions(ctx, opts)
						return workflow.ExecuteActivity(actCtx, activityStub.Sleep, actInput).Get(ctx, nil)
					})
				}
				return workflowutils.RunConcurrently(ctx, sleepFuncs...)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
				if params.SkipSleep {
					return nil
				}
				return workflow.ExecuteActivity(actCtx, activityStub.SelfUpdate, UpdateSleep).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
				return workflow.ExecuteActivity(actCtx, activityStub.SelfUpdate, UpdateActivity).Get(ctx, nil)
			},
			func(ctx workflow.Context) error {
				actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
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
		); err != nil {
			return nil, err
		}

		// Wait for all handlers to finish
		if err := workflow.Await(ctx, func() bool {
			return workflow.AllHandlersFinished(ctx)
		}); err != nil {
			return nil, err
		}

		// Possibly continue as new
		if (i+1)%params.ContinueAsNewAfterIterCount == 0 {
			if params.InitialIteration == i {
				// If this is the first iteration on this workflow run, don't ContinueAsNew since the workflow would never complete.
				workflow.GetLogger(ctx).Info("skipping ContinueAsNew; will ContinueAsNew next iteration")
				continue
			}
			params.InitialIteration = i + 1 // plus one to start at the *next* iteration
			return nil, workflow.NewContinueAsNewError(ctx, "throughputStress", params)
		}
	}

	return &throughputstress.WorkflowOutput{
		ChildrenSpawned: params.ChildrenSpawned,
	}, nil
}

func nexusClient(ctx workflow.Context, params *throughputstress.WorkflowParams) workflow.NexusClient {
	if params.NexusEndpoint == "" {
		workflow.GetLogger(ctx).Debug("not running nexus tests, set nexusEndpoint in options to enable these tests")
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
		actCtx := workflow.WithActivityOptions(ctx, defaultActivityOpts(ctx))
		err := workflow.ExecuteActivity(
			actCtx, activityStub.Payload, MakePayloadInput(256, 256)).Get(ctx, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

func defaultActivityOpts(ctx workflow.Context) workflow.ActivityOptions {
	return workflow.ActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{},
	}
}

func defaultLocalActivityOpts(ctx workflow.Context) workflow.LocalActivityOptions {
	return workflow.LocalActivityOptions{
		StartToCloseTimeout: 1 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval: 10 * time.Millisecond,
			MaximumAttempts: 10,
		},
	}
}
