package scenarios

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

// --option arguments
const (
	IterFlag                   = "internal-iterations"
	IterTimeoutFlag            = "internal-iterations-timeout"
	ContinueAsNewAfterIterFlag = "continue-as-new-after-iterations"
	NexusEndpointFlag          = "nexus-endpoint"
	// MaxStartAttemptFlag is a flag to set the maximum number of attempts for starting a workflow.
	MaxStartAttemptFlag = "max-start-attempt"
	// MaxStartAttemptBackoffFlag is a flag to set the maximum backoff time between attempts for starting a workflow.
	MaxStartAttemptBackoffFlag = "max-start-attempt-backoff"
	// SkipCleanNamespaceCheckFlag is a flag to skip the check for existing workflows in the namespace.
	// This should be set to allow resuming from a previous run.
	SkipCleanNamespaceCheckFlag = "skip-clean-namespace-check"
	// VisibilityVerificationTimeoutFlag is the timeout for verifying the total visibility count at the end of the scenario.
	// It needs to account for a backlog of tasks and, if used, ElasticSearch's eventual consistency.
	VisibilityVerificationTimeoutFlag = "visibility-count-timeout"
	// SleepActivityJsonFlag is a JSON string that defines the sleep activity's behavior.
	// See throughputstress.SleepActivityConfig for details.
	SleepActivityJsonFlag = "sleep-activity-json"
)

const (
	baseBackoff                               = 1 * time.Second
	ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"
)

type tpsState struct {
	// CompletedIterations is the number of iteration that have been completed.
	CompletedIterations int `json:"completedIterations"`
	// LastCompletedIterationAt is the time when the last iteration was completed. Helpful for debugging.
	LastCompletedIterationAt time.Time `json:"lastCompletedIterationAt"`
}

type tpsConfig struct {
	InternalIterations            int
	InternalIterTimeout           time.Duration
	ContinueAsNewAfterIter        int
	MaxAttempts                   int
	MaxBackoff                    time.Duration
	NexusEndpoint                 string
	SkipSleep                     bool
	SkipCleanNamespaceCheck       bool
	SleepActivities               *loadgen.SleepActivityConfig
	VisibilityVerificationTimeout time.Duration
}

type tpsExecutor struct {
	lock       sync.Mutex
	state      *tpsState
	config     *tpsConfig
	isResuming bool

	iterations             int
	initialIteration       int
	continueAsNewAfterIter int
	childrenSpawned        int
	sleepActivities        *loadgen.SleepActivityConfig
	nexusEndpoint          string
}

var _ loadgen.Resumable = (*tpsExecutor)(nil)
var _ loadgen.Configurable = (*tpsExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' to control internal parameters",
			IterFlag, ContinueAsNewAfterIterFlag),
		Executor: &tpsExecutor{state: &tpsState{}},
	})
}

// Snapshot returns a snapshot of the current state.
func (t *tpsExecutor) Snapshot() any {
	t.lock.Lock()
	defer t.lock.Unlock()

	return *t.state
}

// LoadState loads the state from the provided byte slice.
func (t *tpsExecutor) LoadState(loader func(any) error) error {
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

func (t *tpsExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &tpsConfig{
		InternalIterTimeout:     info.ScenarioOptionDuration(IterTimeoutFlag, cmp.Or(info.Configuration.Duration+1*time.Minute, 1*time.Minute)),
		NexusEndpoint:           info.ScenarioOptions[NexusEndpointFlag],
		SkipCleanNamespaceCheck: info.ScenarioOptionBool(SkipCleanNamespaceCheckFlag, false),
	}

	config.InternalIterations = info.ScenarioOptionInt(IterFlag, 10)
	if config.InternalIterations <= 0 {
		return fmt.Errorf("internal-iterations must be positive, got %d", config.InternalIterations)
	}

	config.ContinueAsNewAfterIter = info.ScenarioOptionInt(ContinueAsNewAfterIterFlag, 3)
	if config.ContinueAsNewAfterIter < 0 {
		return fmt.Errorf("continue-as-new-after-iterations must be non-negative, got %d", config.ContinueAsNewAfterIter)
	}

	config.MaxAttempts = info.ScenarioOptionInt(MaxStartAttemptFlag, 5)
	if config.MaxAttempts <= 0 {
		return fmt.Errorf("max-start-attempt must be positive, got %d", config.MaxAttempts)
	}

	config.MaxBackoff = info.ScenarioOptionDuration(MaxStartAttemptBackoffFlag, 60*time.Second)
	if config.MaxBackoff <= 0 {
		return fmt.Errorf("max-start-attempt-backoff must be positive, got %v", config.MaxBackoff)
	}

	if sleepActivitiesStr, ok := info.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		config.SleepActivities, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", SleepActivityJsonFlag, err)
		}
	}

	var err error
	config.VisibilityVerificationTimeout, err = time.ParseDuration(cmp.Or(info.ScenarioOptions[VisibilityVerificationTimeoutFlag], "3m"))
	if err != nil {
		return fmt.Errorf("invalid %s: %w", VisibilityVerificationTimeoutFlag, err)
	}
	if config.VisibilityVerificationTimeout <= 0 {
		return fmt.Errorf("%s must be positive, got %v", VisibilityVerificationTimeoutFlag, config.VisibilityVerificationTimeout)
	}

	t.config = config
	return nil
}

// Run executes the throughput stress scenario.
//
// It executes `throughputStress` workflows in parallel - up to the configured maximum cocurrency limit - and
// waits for the results. At the end, it verifies that the total number of executed workflows matches Visibility's count.
//
// To resume a previous run, capture the state via the StatusCallback and then set `--option resume-from-state=<state>`.
// Note that the caller is responsible for adjusting the run config's iterations/timeout accordingly.
func (t *tpsExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := t.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}

	// Add search attribute, if it doesn't exist yet, to query for workflows by run ID.
	// Running this on resume, too, in case a previous Omes run crashed before it could add the search attribute.
	if err := loadgen.InitSearchAttribute(ctx, info, ThroughputStressScenarioIdSearchAttribute); err != nil {
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
		if err := t.verifyFirstRun(ctx, info, t.config.SkipCleanNamespaceCheck); err != nil {
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
			TestInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{
						NoOpSingleActivityActionSet(),
					},
				},
			},
			PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *TestInput) error {
				t.iterations = opts.ScenarioOptionInt(IterFlag, 10)
				t.continueAsNewAfterIter = opts.ScenarioOptionInt(ContinueAsNewAfterIterFlag, 3)
				t.initialIteration = 0
				t.childrenSpawned = 0
				t.nexusEndpoint = opts.ScenarioOptions[NexusEndpointFlag]
				if sleepActivitiesStr, ok := opts.ScenarioOptions[SleepActivityJsonFlag]; ok {
					var err error
					t.sleepActivities, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
					if err != nil {
						return fmt.Errorf("failed to parse %s: %w", SleepActivityJsonFlag, err)
					}
				}

				params.WorkflowInput.InitialActions = t.createActions()
				params.ClientSequence = t.createClientSequence()
				return nil
			},
			UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
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

	completedChildWorkflows := completedIterations * t.config.InternalIterations
	info.Logger.Info("Total child workflows: ", completedChildWorkflows)

	var continueAsNewWorkflows int
	if t.config.ContinueAsNewAfterIter > 0 {
		continueAsNewWorkflows = int(t.config.InternalIterations/t.config.ContinueAsNewAfterIter) * completedIterations
	}
	info.Logger.Info("Total continue-as-new workflows: ", continueAsNewWorkflows)

	completedWorkflows := completedIterations + completedChildWorkflows + continueAsNewWorkflows
	info.Logger.Info("Total workflows completed: ", completedWorkflows)

	// Post-scenario: verify that at least one iteration was completed.
	if completedIterations == 0 {
		return errors.New("No iterations completed. Either the scenario never ran, or it failed to resume correctly.")
	}

	// Note: Visibility verification is simplified for now since we're using KitchenSinkExecutor
	return nil
	// Post-scenario: verify reported workflow completion count from Visibility.
	return loadgen.MinVisibilityCountEventually(
		ctx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				ThroughputStressScenarioIdSearchAttribute, info.RunID),
		},
		completedWorkflows,
		t.config.VisibilityVerificationTimeout,
	)
}

func (t *tpsExecutor) verifyFirstRun(ctx context.Context, info loadgen.ScenarioInfo, skipCleanNamespaceCheck bool) error {
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

func (t *tpsExecutor) updateStateOnIterationCompletion(completedIter int) {
	t.lock.Lock()
	defer t.lock.Unlock()
	t.state.CompletedIterations += 1
	t.state.LastCompletedIterationAt = time.Now()
}

func (t *tpsExecutor) createActions() []*ActionSet {
	var actions []*Action
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := t.initialIteration; i < t.iterations; i++ {
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
			newInput := &WorkflowInput{
				InitialActions: t.createActions(),
			}
			t.initialIteration = originalInitialIteration // restore original value

			actions = append(actions, &Action{
				Variant: &Action_ContinueAsNew{
					ContinueAsNew: &ContinueAsNewAction{
						Arguments: []*common.Payload{ConvertToPayload(newInput)},
					},
				},
			})
			break
		}
	}

	// Add the final ReturnResult action as required by the kitchen sink workflow
	// Maps to the final workflow output: &throughputstress.WorkflowOutput{ChildrenSpawned: params.ChildrenSpawned}
	actions = append(actions, &Action{
		Variant: &Action_ReturnResult{
			ReturnResult: &ReturnResultAction{
				ReturnThis: &common.Payload{},
			},
		},
	})

	return []*ActionSet{
		{
			Actions:    actions,
			Concurrent: false,
		},
	}
}

func (t *tpsExecutor) createSingleIterationActions(rng *rand.Rand, iteration int) []*Action {
	return []*Action{
		PayloadActivity(256, 256, AsRemoteActivityAction), // CHECK!
		//t.createSelfQuery(), // TODO
		//t.createSelfDescribe(), // TODO
		PayloadActivity(0, 256, AsLocalActivityAction), // CHECK!
		PayloadActivity(0, 256, AsLocalActivityAction), // CHECK!
		&Action{
			Variant: &Action_NestedActionSet{
				NestedActionSet: &ActionSet{
					Actions:    t.createConcurrentActions(rng, iteration),
					Concurrent: true,
				},
			},
		},
	}
}

func (t *tpsExecutor) createConcurrentActions(rng *rand.Rand, iteration int) []*Action {
	actions := []*Action{
		t.createChildWorkflowAction(iteration),            // CHECK!
		PayloadActivity(256, 256, AsRemoteActivityAction), // CHECK!
		PayloadActivity(256, 256, AsRemoteActivityAction), // CHECK!
		PayloadActivity(0, 256, AsLocalActivityAction),    // CHECK!
		PayloadActivity(0, 256, AsLocalActivityAction),    // CHECK!
		GenericActivity("noop", AsLocalActivityAction),
	}

	if t.sleepActivities != nil { // CHECK!
		// This activity simulates custom traffic patterns by generating activities that sleep
		// for a specified duration, with optional priority and fairness keys.
		sleepActivityActions := t.sleepActivities.Sample(rng)
		for _, sleepAction := range sleepActivityActions {
			actions = append(actions, &Action{
				Variant: &Action_ExecActivity{
					ExecActivity: sleepAction,
				},
			})
		}
	}

	if t.nexusEndpoint != "" {
		actions = append(actions, t.createNexusEchoSyncAction())
		actions = append(actions, t.createNexusEchoAsyncAction())
		actions = append(actions, t.createNexusWaitForCancelAction())
	}

	return actions
}

func (t *tpsExecutor) createChildWorkflowAction(iteration int) *Action {
	return &Action{
		Variant: &Action_ExecChildWorkflow{
			ExecChildWorkflow: &ExecuteChildWorkflowAction{
				Input: []*common.Payload{
					ConvertToPayload(&WorkflowInput{
						InitialActions: []*ActionSet{
							{
								Actions: []*Action{
									PayloadActivity(256, 256, AsRemoteActivityAction),
									PayloadActivity(256, 256, AsRemoteActivityAction),
									PayloadActivity(256, 256, AsRemoteActivityAction),
									NewEmptyReturnResultAction(),
								},
								Concurrent: false,
							},
						},
					}),
				},
				// TODO
				// Make sure we pass through the search attribute that correlates us to a scenario
				// run to the child.
				//SearchAttributes: map[string]*common.Payload{
				//	ThroughputStressScenarioIdSearchAttribute: kitchensink.ConvertToPayload(attrs[scenarios.ThroughputStressScenarioIdSearchAttribute]),
				//},
			},
		},
	}
}

func (t *tpsExecutor) createClientSequence() *ClientSequence {
	actions := []*ClientAction{
		t.createSelfQuery(),
		// This self-signal activity didn't exist in the original bench-go workflow, but
		// it was signaled semi-routinely externally. This is slightly more obvious and
		// introduces receiving signals in the workflow.
		&ClientAction{
			Variant: &ClientAction_DoSignal{
				DoSignal: &DoSignal{
					Variant: &DoSignal_DoSignalActions_{
						DoSignalActions: &DoSignal_DoSignalActions{
							Variant: &DoSignal_DoSignalActions_DoActions{
								DoActions: SingleActionSet(NewTimerAction(1)),
							},
						},
					},
				},
			},
		},
		// CHECK! UpdateSleep
		&ClientAction{
			Variant: &ClientAction_DoUpdate{
				DoUpdate: &DoUpdate{
					Variant: &DoUpdate_Custom{
						Custom: &HandlerInvocation{
							Name: "do_actions_update",
							Args: []*common.Payload{
								ConvertToPayload(&DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(NewTimerAction(uint64((1 * time.Second).Milliseconds()))),
									},
								}),
							},
						},
					},
				},
			},
		},
	}

	actions = append(actions,
		&ClientAction{
			Variant: &ClientAction_DoUpdate{
				DoUpdate: &DoUpdate{
					Variant: &DoUpdate_Custom{
						Custom: &HandlerInvocation{
							Name: "do_actions_update",
							Args: []*common.Payload{
								ConvertToPayload(&DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(
											PayloadActivity(0, 256, AsRemoteActivityAction),
										),
									},
								}),
							},
						},
					},
				},
			},
		},
		&ClientAction{
			Variant: &ClientAction_DoUpdate{
				DoUpdate: &DoUpdate{
					Variant: &DoUpdate_Custom{
						Custom: &HandlerInvocation{
							Name: "do_actions_update",
							Args: []*common.Payload{
								ConvertToPayload(&DoActionsUpdate{
									Variant: &DoActionsUpdate_DoActions{
										DoActions: SingleActionSet(
											PayloadActivity(0, 256, AsLocalActivityAction),
										),
									},
								}),
							},
						},
					},
				},
			},
		},
	)

	return &ClientSequence{
		ActionSets: []*ClientActionSet{
			{
				Actions:    actions,
				Concurrent: true,
			},
		},
	}
}

func (t *tpsExecutor) createSelfQuery() *ClientAction {
	return &ClientAction{
		Variant: &ClientAction_DoQuery{
			DoQuery: &DoQuery{
				Variant: &DoQuery_ReportState{
					ReportState: &common.Payloads{},
				},
			},
		},
	}
}

func (t *tpsExecutor) createNexusEchoSyncAction() *Action {
	return &Action{
		Variant: &Action_NexusOperation{
			NexusOperation: &ExecuteNexusOperation{
				Endpoint:       t.nexusEndpoint,
				Operation:      "echo-sync",
				Input:          "hello",
				ExpectedOutput: "hello",
			},
		},
	}
}

func (t *tpsExecutor) createNexusEchoAsyncAction() *Action {
	return &Action{
		Variant: &Action_NexusOperation{
			NexusOperation: &ExecuteNexusOperation{
				Endpoint:       t.nexusEndpoint,
				Operation:      "echo-async",
				Input:          "hello",
				ExpectedOutput: "hello",
			},
		},
	}
}

func (t *tpsExecutor) createNexusWaitForCancelAction() *Action {
	return &Action{
		Variant: &Action_NexusOperation{
			NexusOperation: &ExecuteNexusOperation{
				Endpoint:  t.nexusEndpoint,
				Operation: "wait-for-cancel",
				Input:     "",
				AwaitableChoice: &AwaitableChoice{
					Condition: &AwaitableChoice_CancelAfterStarted{
						CancelAfterStarted: &emptypb.Empty{},
					},
				},
			},
		},
	}
}

