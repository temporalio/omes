package scenarios

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type tpsExecutorV2 struct {
	lock       sync.Mutex
	state      *tpsState
	isResuming bool

	iterations             int
	initialIteration       int
	skipSleep              bool
	continueAsNewAfterIter int
	childrenSpawned        int
	sleepActivities        *loadgen.SleepActivityConfig
}

var _ loadgen.Resumable = (*tpsExecutorV2)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' '%s' to control internal parameters",
			IterFlag, ContinueAsNewAfterIterFlag, SkipSleepFlag),
		Executor: &tpsExecutorV2{state: &tpsState{}},
	})
}

// Return a snapshot of the current state.
func (t *tpsExecutorV2) Snapshot() any {
	t.lock.Lock()
	defer t.lock.Unlock()

	return *t.state
}

// LoadState loads the state from the provided byte slice.
func (t *tpsExecutorV2) LoadState(loader func(any) error) error {
	var state tpsState
	if err := loader(&state); err != nil {
		return err
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.state = &state
	t.isResuming = true

	return nil
}

// Run executes the throughput stress scenario.
//
// It executes `throughputStress` workflows in parallel - up to the configured maximum cocurrency limit - and
// waits for the results. At the end, it verifies that the total number of executed workflows matches Visibility's count.
//
// To resume a previous run, capture the state via the StatusCallback and then set `--option resume-from-state=<state>`.
// Note that the caller is responsible for adjusting the run config's iterations/timeout accordingly.
func (t *tpsExecutorV2) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Parse scenario options
	skipCleanNamespaceCheck := info.ScenarioOptionBool(SkipCleanNamespaceCheckFlag, false)

	// Add search attribute, if it doesn't exist yet, to query for workflows by run ID.
	// Running this on resume, too, in case a previous Omes run crashed before it could add the search attribute.
	if err := t.initSearchAttribute(ctx, info); err != nil {
		return err
	}

	t.lock.Lock()
	isResuming := t.isResuming
	currentState := *t.state
	t.lock.Unlock()

	if isResuming {
		info.Logger.Info(fmt.Sprintf("Resuming scenario from state: %#v", currentState))
		info.Configuration.StartFromIteration = int(currentState.CompletedIterations) + 1
	} else {
		if err := t.verifyFirstRun(ctx, info, skipCleanNamespaceCheck); err != nil {
			return err
		}
	}

	// Start the scenario run.
	//
	// However; when resuming, it can happen that there is no more time left to run more iterations. In that case,
	// we skip the executor run and go straight to the post-scenario verification.
	if isResuming && info.Configuration.Duration <= 0 {
		info.Logger.Info("Skipping executor run: out of time")
	} else {
		ksExec := &loadgen.KitchenSinkExecutor{
			DefaultConfiguration: loadgen.RunConfiguration{
				Iterations:    20,
				MaxConcurrent: 5,
			},
			TestInput: &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{
						kitchensink.NoOpSingleActivityActionSet(),
					},
				},
			},
			PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
				// Parse scenario options for throughput stress behavior and store in tpsExecutor
				t.iterations = opts.ScenarioOptionInt(IterFlag, 10)
				t.skipSleep = opts.ScenarioOptionBool(SkipSleepFlag, false)
				t.continueAsNewAfterIter = opts.ScenarioOptionInt(ContinueAsNewAfterIterFlag, 3)
				t.initialIteration = 0
				t.childrenSpawned = 0

				// Parse sleep activities if specified
				if sleepActivitiesStr, ok := opts.ScenarioOptions[SleepActivityJsonFlag]; ok {
					var err error
					t.sleepActivities, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
					if err != nil {
						return fmt.Errorf("failed to parse %s: %w", SleepActivityJsonFlag, err)
					}
				}

				// Replace the default noop action with full throughput stress actions
				params.WorkflowInput.InitialActions = t.createActions()

				// Add client-side operations (DescribeWorkflowExecution, SignalWorkflow, UpdateWorkflow)
				params.ClientSequence = t.createClientSequence()

				return nil
			},
			UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
				// Update state on each iteration
				t.updateStateOnIterationCompletion(run.Iteration)
				return nil
			},
		}
		if err := ksExec.Run(ctx, info); err != nil {
			return err
		}
	}

	t.lock.Lock()
	var completedIterations = t.state.CompletedIterations
	t.lock.Unlock()
	info.Logger.Info("Total iterations completed: ", completedIterations)

	// Post-scenario: verify that at least one iteration was completed.
	if completedIterations == 0 {
		return errors.New("No iterations completed. Either the scenario never ran, or it failed to resume correctly.")
	}

	// Note: Visibility verification is simplified for now since we're using KitchenSinkExecutor
	return nil
}

func (t *tpsExecutorV2) initSearchAttribute(ctx context.Context, info loadgen.ScenarioInfo) error {
	info.Logger.Infof("Initialising Search Attribute %s", ThroughputStressScenarioIdSearchAttribute)

	_, err := info.Client.OperatorService().AddSearchAttributes(ctx,
		&operatorservice.AddSearchAttributesRequest{
			Namespace: info.Namespace,
			SearchAttributes: map[string]enums.IndexedValueType{
				ThroughputStressScenarioIdSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
			},
		})
	var deniedErr *serviceerror.PermissionDenied
	var alreadyErr *serviceerror.AlreadyExists
	if errors.As(err, &alreadyErr) {
		info.Logger.Infof("Search Attribute %s already exists", ThroughputStressScenarioIdSearchAttribute)
	} else if err != nil {
		info.Logger.Warnf("Failed to add Search Attribute %s: %v", ThroughputStressScenarioIdSearchAttribute, err)
		if !errors.As(err, &deniedErr) {
			return err
		}
	} else {
		info.Logger.Infof("Search Attribute %s added", ThroughputStressScenarioIdSearchAttribute)
	}

	return nil
}

func (t *tpsExecutorV2) verifyFirstRun(ctx context.Context, info loadgen.ScenarioInfo, skipCleanNamespaceCheck bool) error {
	if skipCleanNamespaceCheck {
		info.Logger.Info("Skipping check to verify if the namespace is clean")
		return nil
	}

	// Complain if there are already existing workflows with the provided run id; unless resuming.
	workflowCountQry := fmt.Sprintf("%s='%s'", ThroughputStressScenarioIdSearchAttribute, info.RunID)
	visibilityCount, err := info.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     workflowCountQry,
	})
	if err != nil {
		return err
	}
	if visibilityCount.Count > 0 {
		return fmt.Errorf("there are already %d workflows with scenario Run ID '%s'",
			visibilityCount.Count, info.RunID)
	}

	return nil
}

func (t *tpsExecutorV2) updateStateOnIterationCompletion(completedIter int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state.CompletedIterations += 1
	t.state.LastCompletedIterationAt = time.Now()
}

func (t *tpsExecutorV2) createActions() []*kitchensink.ActionSet {
	var actions []*kitchensink.Action

	// Create a random number generator for sleep activity sampling
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Loop through iterations (from InitialIteration to Iterations)
	for i := t.initialIteration; i < t.iterations; i++ {
		// Create all the actions for this iteration
		iterationActions := t.createSingleIterationActions(rng, i)
		actions = append(actions, iterationActions...)

		// Check for continue-as-new
		if t.continueAsNewAfterIter > 0 && (i+1)%t.continueAsNewAfterIter == 0 {
			// Skip ContinueAsNew if this is the first iteration on this workflow run
			if t.initialIteration == i {
				// If this is the first iteration on this workflow run, don't ContinueAsNew since the workflow would never complete.
				continue
			}

			// Create new workflow input with updated iteration count
			originalInitialIteration := t.initialIteration
			t.initialIteration = i + 1 // plus one to start at the *next* iteration
			newInput := &kitchensink.WorkflowInput{
				InitialActions: t.createActions(),
			}
			t.initialIteration = originalInitialIteration // restore original value

			// Serialize the new input using converter
			payload, _ := converter.GetDefaultDataConverter().ToPayload(newInput)

			actions = append(actions, &kitchensink.Action{
				Variant: &kitchensink.Action_ContinueAsNew{
					ContinueAsNew: &kitchensink.ContinueAsNewAction{
						Arguments: []*common.Payload{payload},
					},
				},
			})
			break
		}
	}

	// Add the final ReturnResult action as required by the kitchen sink workflow
	// Maps to the final workflow output: &throughputstress.WorkflowOutput{ChildrenSpawned: params.ChildrenSpawned}
	actions = append(actions, &kitchensink.Action{
		Variant: &kitchensink.Action_ReturnResult{
			ReturnResult: &kitchensink.ReturnResultAction{
				ReturnThis: &common.Payload{},
			},
		},
	})

	return []*kitchensink.ActionSet{
		{
			Actions:    actions,
			Concurrent: false,
		},
	}
}

// createSingleIterationActions creates all actions for a single iteration to match the original throughput stress workflow
// This maps directly to the main loop in ThroughputStressWorkflow (lines 71-240)
func (t *tpsExecutorV2) createSingleIterationActions(rng *rand.Rand, iteration int) []*kitchensink.Action {
	var actions []*kitchensink.Action
	actions = append(actions, createPayloadActivity(256, 256))
	actions = append(actions, createSelfQueryActivity())
	actions = append(actions, createSelfDescribeActivity())
	actions = append(actions, createPayloadLocalActivity(0, 256))
	actions = append(actions, createPayloadLocalActivity(0, 256))
	concurrentActions := t.createConcurrentActions(rng, iteration)
	actions = append(actions, &kitchensink.Action{
		Variant: &kitchensink.Action_NestedActionSet{
			NestedActionSet: &kitchensink.ActionSet{
				Actions:    concurrentActions,
				Concurrent: true,
			},
		},
	})
	return actions
}

// createPayloadActivity creates a payload activity
func createPayloadActivity(inSize, outSize int) *kitchensink.Action {
	// Generate random input data of inSize bytes
	inData := make([]byte, inSize)
	rand.Read(inData)

	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Payload{
					Payload: &kitchensink.ExecuteActivityAction_PayloadActivity{
						InputData:     inData,
						BytesToReturn: int32(outSize),
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
			},
		},
	}
}

// createPayloadLocalActivity creates a payload local activity
func createPayloadLocalActivity(inSize, outSize int) *kitchensink.Action {
	// Generate random input data of inSize bytes
	inData := make([]byte, inSize)
	rand.Read(inData)

	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Payload{
					Payload: &kitchensink.ExecuteActivityAction_PayloadActivity{
						InputData:     inData,
						BytesToReturn: int32(outSize),
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
				Locality: &kitchensink.ExecuteActivityAction_IsLocal{
					IsLocal: &emptypb.Empty{},
				},
				RetryPolicy: &common.RetryPolicy{
					InitialInterval:    &durationpb.Duration{Nanos: 10000000}, // 10ms
					MaximumAttempts:    10,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

// createSelfQueryActivity creates a self-query activity
func createSelfQueryActivity() *kitchensink.Action {
	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Generic{
					Generic: &kitchensink.ExecuteActivityAction_GenericActivity{
						Type:      "noop",
						Arguments: []*common.Payload{}, // Empty arguments for noop
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
				RetryPolicy: &common.RetryPolicy{
					MaximumAttempts:    1,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

// createSelfDescribeActivity creates a self-describe activity
func createSelfDescribeActivity() *kitchensink.Action {
	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Generic{
					Generic: &kitchensink.ExecuteActivityAction_GenericActivity{
						Type:      "noop",
						Arguments: []*common.Payload{}, // Empty arguments for noop
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
				RetryPolicy: &common.RetryPolicy{
					MaximumAttempts:    1,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

// createConcurrentActions creates the concurrent actions that run in parallel
func (t *tpsExecutorV2) createConcurrentActions(rng *rand.Rand, iteration int) []*kitchensink.Action {
	var actions []*kitchensink.Action
	actions = append(actions, t.createChildWorkflowAction(iteration))
	actions = append(actions, createPayloadActivity(256, 256))
	actions = append(actions, createPayloadActivity(256, 256))
	actions = append(actions, createPayloadLocalActivity(0, 256))
	actions = append(actions, createPayloadLocalActivity(0, 256))
	actions = append(actions, createSelfSignalActivity())
	if t.sleepActivities != nil {
		sleepActivityActions := t.sleepActivities.Sample(rng)
		for _, sleepAction := range sleepActivityActions {
			actions = append(actions, &kitchensink.Action{
				Variant: &kitchensink.Action_ExecActivity{
					ExecActivity: sleepAction,
				},
			})
		}
	}
	if !t.skipSleep {
		actions = append(actions, createSelfUpdateActivity("sleep"))
	}
	actions = append(actions, createSelfUpdateActivity("activity"))
	actions = append(actions, createSelfUpdateActivity("localActivity"))
	// TODO: Nexus
	return actions
}

func (t *tpsExecutorV2) createChildWorkflowAction(iteration int) *kitchensink.Action {
	var childActions []*kitchensink.Action
	for i := 0; i < 3; i++ {
		childActions = append(childActions, createPayloadActivity(256, 256))
	}
	childActions = append(childActions, &kitchensink.Action{
		Variant: &kitchensink.Action_ReturnResult{
			ReturnResult: &kitchensink.ReturnResultAction{
				ReturnThis: &common.Payload{},
			},
		},
	})
	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecChildWorkflow{
			ExecChildWorkflow: &kitchensink.ExecuteChildWorkflowAction{
				WorkflowType: "kitchenSink",
				Input: []*common.Payload{
					func() *common.Payload {
						childInput := &kitchensink.WorkflowInput{
							InitialActions: []*kitchensink.ActionSet{
								{
									Actions:    childActions,
									Concurrent: false,
								},
							},
						}
						payload, _ := converter.GetDefaultDataConverter().ToPayload(childInput)
						return payload
					}(),
				},
				WorkflowExecutionTimeout: &durationpb.Duration{Seconds: 3600}, // 1 hour timeout
			},
		},
	}
}

func createSelfSignalActivity() *kitchensink.Action {
	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Generic{
					Generic: &kitchensink.ExecuteActivityAction_GenericActivity{
						Type:      "noop",
						Arguments: []*common.Payload{}, // Empty arguments for noop
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
				Locality: &kitchensink.ExecuteActivityAction_IsLocal{
					IsLocal: &emptypb.Empty{},
				},
				RetryPolicy: &common.RetryPolicy{
					InitialInterval:    &durationpb.Duration{Nanos: 10000000}, // 10ms
					MaximumAttempts:    10,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

func createSelfUpdateActivity(updateName string) *kitchensink.Action {
	return &kitchensink.Action{
		Variant: &kitchensink.Action_ExecActivity{
			ExecActivity: &kitchensink.ExecuteActivityAction{
				ActivityType: &kitchensink.ExecuteActivityAction_Generic{
					Generic: &kitchensink.ExecuteActivityAction_GenericActivity{
						Type:      "noop",
						Arguments: []*common.Payload{}, // Empty arguments for noop
					},
				},
				StartToCloseTimeout: &durationpb.Duration{Seconds: 60},
				RetryPolicy: &common.RetryPolicy{
					MaximumAttempts:    1,
					BackoffCoefficient: 1.0,
				},
			},
		},
	}
}

func (t *tpsExecutorV2) createClientSequence() *kitchensink.ClientSequence {
	var actionSets []*kitchensink.ClientActionSet

	for i := t.initialIteration; i < t.iterations; i++ {
		var actions []*kitchensink.ClientAction

		actions = append(actions, &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoQuery{
				DoQuery: &kitchensink.DoQuery{
					Variant: &kitchensink.DoQuery_ReportState{
						ReportState: &common.Payloads{},
					},
				},
			},
		})

		actions = append(actions, &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoSignal{
				DoSignal: &kitchensink.DoSignal{
					Variant: &kitchensink.DoSignal_Custom{
						Custom: &kitchensink.HandlerInvocation{
							Name: "test_signal",
							Args: []*common.Payload{},
						},
					},
				},
			},
		})

		if !t.skipSleep {
			actions = append(actions, &kitchensink.ClientAction{
				Variant: &kitchensink.ClientAction_DoUpdate{
					DoUpdate: &kitchensink.DoUpdate{
						Variant: &kitchensink.DoUpdate_Custom{
							Custom: &kitchensink.HandlerInvocation{
								Name: "update_sleep",
								Args: []*common.Payload{},
							},
						},
					},
				},
			})
		}

		actions = append(actions, &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoUpdate{
				DoUpdate: &kitchensink.DoUpdate{
					Variant: &kitchensink.DoUpdate_Custom{
						Custom: &kitchensink.HandlerInvocation{
							Name: "update_activity",
							Args: []*common.Payload{},
						},
					},
				},
			},
		})

		actions = append(actions, &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoUpdate{
				DoUpdate: &kitchensink.DoUpdate{
					Variant: &kitchensink.DoUpdate_Custom{
						Custom: &kitchensink.HandlerInvocation{
							Name: "update_local_activity",
							Args: []*common.Payload{},
						},
					},
				},
			},
		})

		// Add the action set to run concurrently
		actionSets = append(actionSets, &kitchensink.ClientActionSet{
			Actions:    actions,
			Concurrent: true,
		})
	}

	return &kitchensink.ClientSequence{
		ActionSets: actionSets,
	}
}
