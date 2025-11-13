package scenarios

import (
	"cmp"
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	IterFlag                   = "internal-iterations"
	IterTimeoutFlag            = "internal-iterations-timeout"
	ContinueAsNewAfterIterFlag = "continue-as-new-after-iterations"
	// SleepTimeFlag controls the duration used for internal timer-based sleep actions (e.g. signal/update timers).
	// Default is 1s.
	SleepTimeFlag     = "sleep-time"
	NexusEndpointFlag = "nexus-endpoint"
	// SkipCleanNamespaceCheckFlag is a flag to skip the check for existing workflows in the namespace.
	// This should be set to allow resuming from a previous run.
	SkipCleanNamespaceCheckFlag = "skip-clean-namespace-check"
	// VisibilityVerificationTimeoutFlag is the timeout for verifying the total visibility count at the end of the scenario.
	// It needs to account for a backlog of tasks and, if used, ElasticSearch's eventual consistency.
	VisibilityVerificationTimeoutFlag = "visibility-count-timeout"
	// SleepActivityJsonFlag is a JSON string that defines the sleep activity's behavior.
	// See throughputstress.SleepActivityConfig for details.
	SleepActivityJsonFlag = "sleep-activity-json"
	// MinThroughputPerHourFlag is the minimum workflow throughput required (workflows/hour).
	// Default is 0, meaning disabled. The scenario calculates actual throughput and compares.
	MinThroughputPerHourFlag = "min-throughput-per-hour"
	// DisableLocalActivitiesFlag converts all local activities to remote activities when set to true.
	// Default is false, meaning local activities will be used as designed.
	DisableLocalActivitiesFlag = "disable-local-activities"
)

type tpsState struct {
	ExecutorState any `json:"executorState"`
}

type tpsConfig struct {
	InternalIterations            int
	InternalIterTimeout           time.Duration
	ContinueAsNewAfterIter        int
	NexusEndpoint                 string
	SleepTime                     time.Duration
	SkipCleanNamespaceCheck       bool
	SleepActivities               *loadgen.SleepActivityConfig
	VisibilityVerificationTimeout time.Duration
	MinThroughputPerHour          float64
	ScenarioRunID                 string
	RngSeed                       int64
	DisableLocalActivities        bool
}

type tpsExecutor struct {
	executor   *loadgen.KitchenSinkExecutor
	verifier   *tpsVerifier
	lock       sync.Mutex
	state      *tpsState
	config     *tpsConfig
	isResuming bool
	runID      string
	rng        *rand.Rand
}

var _ loadgen.Resumable = (*tpsExecutor)(nil)
var _ loadgen.Configurable = (*tpsExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' to control internal parameters",
			IterFlag, ContinueAsNewAfterIterFlag),
		ExecutorFn: func() loadgen.Executor { return newThroughputStressExecutor() },
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			t := executor.(*tpsExecutor)
			if t.verifier == nil || t.executor == nil {
				return nil
			}
			state := t.executor.GetState()
			return t.verifier.VerifyRun(ctx, info, state)
		},
	})
}

func newThroughputStressExecutor() *tpsExecutor {
	return &tpsExecutor{state: &tpsState{}}
}

// Snapshot returns a snapshot of the current state.
func (t *tpsExecutor) Snapshot() any {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.executor == nil {
		return *t.state
	}

	return tpsState{
		ExecutorState: t.executor.Snapshot(),
	}
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

// Configure initializes tpsConfig. Largely, it reads and validates throughput_stress scenario options
func (t *tpsExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &tpsConfig{
		InternalIterTimeout:     info.ScenarioOptionDuration(IterTimeoutFlag, cmp.Or(info.Configuration.Duration+1*time.Minute, 1*time.Minute)),
		NexusEndpoint:           info.ScenarioOptions[NexusEndpointFlag],
		SkipCleanNamespaceCheck: info.ScenarioOptionBool(SkipCleanNamespaceCheckFlag, false),
		MinThroughputPerHour:    info.ScenarioOptionFloat(MinThroughputPerHourFlag, 0),
		ScenarioRunID:           info.RunID,
	}

	// Generate random number generator seed based on the run ID.
	h := fnv.New64a()
	h.Write([]byte(info.RunID))
	config.RngSeed = int64(h.Sum64())

	config.SleepTime = info.ScenarioOptionDuration(SleepTimeFlag, 1*time.Second)
	if config.SleepTime <= 0 {
		return fmt.Errorf("%s must be positive, got %v", SleepTimeFlag, config.SleepTime)
	}

	config.InternalIterations = info.ScenarioOptionInt(IterFlag, 10)
	if config.InternalIterations <= 0 {
		return fmt.Errorf("internal-iterations must be positive, got %d", config.InternalIterations)
	}

	config.ContinueAsNewAfterIter = info.ScenarioOptionInt(ContinueAsNewAfterIterFlag, 3)
	if config.ContinueAsNewAfterIter < 0 {
		return fmt.Errorf("continue-as-new-after-iterations must be non-negative, got %d", config.ContinueAsNewAfterIter)
	}

	if sleepActivitiesStr, ok := info.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		config.SleepActivities, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr, true)
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

	config.DisableLocalActivities = info.ScenarioOptionBool(DisableLocalActivitiesFlag, false)

	t.config = config
	t.rng = rand.New(rand.NewSource(config.RngSeed))

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
	t.runID = info.RunID

	t.lock.Lock()
	isResuming := t.isResuming
	currentState := *t.state
	t.lock.Unlock()

	if isResuming {
		info.Logger.Info(fmt.Sprintf("Resuming scenario from state: %#v", currentState))
		if execState, ok := currentState.ExecutorState.(loadgen.ExecutorState); ok {
			info.Configuration.StartFromIteration = execState.CompletedIterations
		}
	} else {
		if err := t.verifyFirstRun(ctx, info, t.config.SkipCleanNamespaceCheck); err != nil {
			return err
		}
	}

	// Start the scenario run.
	//
	// NOTE: When resuming, it can happen that there are no more iterations/time left to run more iterations.
	// In that case, we skip the executor run and go straight to the post-scenario verification.
	if isResuming && info.Configuration.Duration <= 0 && info.Configuration.Iterations == 0 {
		info.Logger.Info("Skipping executor run: out of time")
	} else {
		t.executor = &loadgen.KitchenSinkExecutor{
			TestInput: &TestInput{
				WorkflowInput: &WorkflowInput{
					InitialActions: []*ActionSet{},
				},
			},
			UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
				// Start some workflows via Update-with-Start.
				if t.maybeWithStart(0.5) {
					options.Params.WithStartAction = &WithStartClientAction{
						Variant: &WithStartClientAction_DoUpdate{
							DoUpdate: &DoUpdate{
								Variant: &DoUpdate_DoActions{
									DoActions: &DoActionsUpdate{
										Variant: &DoActionsUpdate_DoActions{},
									},
								},
								WithStart: true,
							},
						},
					}
				}

				// Generate the actions for the workflow.
				//
				// NOTE: No client actions (e.g. Signal) are defined; however, client action activities are.
				// That means these client actions are sent from the activity worker instead of Omes.
				options.Params.WorkflowInput.InitialActions = t.createActions(run)

				return nil
			},
		}

		// Restore state if resuming
		if isResuming {
			if execState, ok := t.state.ExecutorState.(loadgen.ExecutorState); ok {
				t.executor.LoadState(func(v any) error {
					s := v.(*loadgen.ExecutorState)
					*s = execState
					return nil
				})
			}
		}

		timeout := info.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 30*time.Second)

		// Configure expected workflow count function based on scenario config
		expectedWorkflowCount := func(state loadgen.ExecutorState) int {
			completedIterations := state.CompletedIterations

			// Calculate continue-as-new workflows
			var continueAsNewWorkflows int
			if t.config.ContinueAsNewAfterIter > 0 {
				// Subtract 1 because the last iteration doesn't trigger a continue-as-new.
				continueAsNewPerIter := (t.config.InternalIterations - 1) / t.config.ContinueAsNewAfterIter
				continueAsNewWorkflows = continueAsNewPerIter * completedIterations
			}

			// Calculate child workflows
			completedChildWorkflows := completedIterations * t.config.InternalIterations

			// Total: parent + children + continue-as-new
			return completedIterations + completedChildWorkflows + continueAsNewWorkflows
		}

		// Initialize workflow completion checker
		completionVerifier, err := loadgen.NewWorkflowCompletionChecker(ctx, info, timeout)
		if err != nil {
			return fmt.Errorf("failed to initialize workflow completion checker: %w", err)
		}
		completionVerifier.SetExpectedWorkflowCount(expectedWorkflowCount)

		// Create verifier that combines workflow completion and throughput checking
		t.verifier = &tpsVerifier{
			completionVerifier: completionVerifier,
			config:             t.config,
		}

		if err := t.executor.Run(ctx, info); err != nil {
			return err
		}
	}

	t.lock.Lock()
	var completedIterations int
	if t.executor != nil {
		completedIterations = t.executor.GetState().CompletedIterations
	} else {
		// Executor was skipped, use state from previous run
		if execState, ok := t.state.ExecutorState.(loadgen.ExecutorState); ok {
			completedIterations = execState.CompletedIterations
		}
	}
	t.lock.Unlock()

	// Calculate completion metrics for logging.
	var continueAsNewPerIter int
	var continueAsNewWorkflows int
	if t.config.ContinueAsNewAfterIter > 0 {
		// Subtract 1 because the last iteration doesn't trigger a continue-as-new.
		continueAsNewPerIter = (t.config.InternalIterations - 1) / t.config.ContinueAsNewAfterIter
		continueAsNewWorkflows = continueAsNewPerIter * completedIterations
	}
	completedChildWorkflows := completedIterations * t.config.InternalIterations
	completedWorkflows := completedIterations + completedChildWorkflows + continueAsNewWorkflows

	// Log completion summary.
	info.Logger.Info(fmt.Sprintf(
		"[Scenario completion summary] Run ID: %s, Total iterations completed: %d, "+
			"Total child workflows: %d (%d per iteration), Total continue-as-new workflows: %d (%d per iteration), "+
			"Total workflows completed: %d",
		info.RunID, completedIterations, completedChildWorkflows, t.config.InternalIterations,
		continueAsNewWorkflows, continueAsNewPerIter, completedWorkflows))

	return nil
}

func (t *tpsExecutor) verifyFirstRun(ctx context.Context, info loadgen.ScenarioInfo, skipCleanNamespaceCheck bool) error {
	if skipCleanNamespaceCheck {
		info.Logger.Info("Skipping check to verify if the namespace is clean")
		return nil
	}

	// Complain if there are already existing workflows with the provided run id; unless resuming.
	workflowCountQry := fmt.Sprintf("%s='%s'", loadgen.OmesExecutionIDSearchAttribute, info.ExecutionID)
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

func (t *tpsExecutor) createActions(run *loadgen.Run) []*ActionSet {
	return []*ActionSet{
		{
			Actions:    t.createActionsChunk(run, 0, 0, t.config.InternalIterations),
			Concurrent: false,
		},
	}
}

// activityLocality returns the appropriate activity locality function based on the config.
// If DisableLocalActivities is true, all activities will be remote; otherwise, return the local activity function.
func (t *tpsExecutor) activityLocality() func(*ExecuteActivityAction) *Action {
	if t.config.DisableLocalActivities {
		return DefaultRemoteActivity
	}
	return DefaultLocalActivity
}

func (t *tpsExecutor) createActionsChunk(
	run *loadgen.Run,
	childCount int,
	continueAsNewCounter int,
	remainingInternalIters int,
) []*Action {
	if remainingInternalIters == 0 {
		return []*Action{}
	}

	var chunkActions []*Action
	itersPerChunk := cmp.Or(t.config.ContinueAsNewAfterIter, t.config.InternalIterations) // no CAN? all iters in one chunk
	isLastChunk := remainingInternalIters <= itersPerChunk
	itersPerChunk = min(itersPerChunk, remainingInternalIters) // cap chunk size to remaining iterations

	rng := rand.New(rand.NewSource(t.config.RngSeed + int64(run.Iteration)))

	// Create actions for the current chunk
	for i := 0; i < itersPerChunk; i++ {
		syncActions := []*Action{
			PayloadActivity(256, 256, t.activityLocality()),
			PayloadActivity(0, 256, t.activityLocality()),
			PayloadActivity(0, 256, t.activityLocality()),
			// TODO: use local activity: server error log "failed to set query completion state to succeeded
			ClientActivity(ClientActions(t.createSelfQuery()), DefaultRemoteActivity),
		}

		childCount++
		asyncActions := []*Action{
			t.createChildWorkflowAction(run, childCount),
			PayloadActivity(256, 256, DefaultRemoteActivity),
			PayloadActivity(256, 256, DefaultRemoteActivity),
			PayloadActivity(0, 256, t.activityLocality()),
			PayloadActivity(0, 256, t.activityLocality()),
			GenericActivity("noop", t.activityLocality()),
			ClientActivity(ClientActions(t.createSelfQuery()), DefaultRemoteActivity),
			ClientActivity(ClientActions(t.createSelfSignal()), t.activityLocality()),
			ClientActivity(ClientActions(t.createSelfUpdateWithTimer()), DefaultRemoteActivity),
			ClientActivity(ClientActions(t.createSelfUpdateWithPayload()), DefaultRemoteActivity),
			// TODO: use local activity: there is an 8s gap in the event history
			ClientActivity(ClientActions(t.createSelfUpdateWithPayloadAsLocal()), DefaultRemoteActivity),
		}

		// Add sleep activities, if configured.
		// It simulates custom traffic patterns by generating activities that sleep
		// for a specified duration, with optional priority and fairness keys.
		if t.config.SleepActivities != nil {
			sleepActivityActions := t.config.SleepActivities.Sample(rng)
			for _, sleepAction := range sleepActivityActions {
				asyncActions = append(asyncActions, &Action{
					Variant: &Action_ExecActivity{
						ExecActivity: sleepAction,
					},
				})
			}
		}

		// Add Nexus operations, if configured.
		if t.config.NexusEndpoint != "" {
			asyncActions = append(asyncActions, t.createNexusEchoSyncAction())
			asyncActions = append(asyncActions, t.createNexusEchoAsyncAction())
		}

		chunkActions = append(chunkActions, syncActions...)
		chunkActions = append(chunkActions, &Action{
			Variant: &Action_NestedActionSet{
				NestedActionSet: &ActionSet{
					Actions:    asyncActions,
					Concurrent: true,
				},
			},
		})
	}

	if isLastChunk {
		// No more iterations remain, add result action to complete workflow.
		chunkActions = append(chunkActions, &Action{
			Variant: &Action_ReturnResult{
				ReturnResult: &ReturnResultAction{
					ReturnThis: &common.Payload{},
				},
			},
		})
	} else {
		// More iterations remain, create nested ContinueAsNew with more actions.
		chunkActions = append(chunkActions, &Action{
			Variant: &Action_ContinueAsNew{
				ContinueAsNew: &ContinueAsNewAction{
					Arguments: []*common.Payload{
						ConvertToPayload(&WorkflowInput{
							InitialActions: []*ActionSet{
								{
									Actions: t.createActionsChunk(
										run,
										childCount,
										continueAsNewCounter+1,
										remainingInternalIters-itersPerChunk),
									Concurrent: false,
								},
							},
						}),
					},
				},
			},
		})
	}

	return chunkActions
}

func (t *tpsExecutor) createChildWorkflowAction(run *loadgen.Run, childID int) *Action {
	return &Action{
		Variant: &Action_ExecChildWorkflow{
			ExecChildWorkflow: &ExecuteChildWorkflowAction{
				Input: []*common.Payload{
					ConvertToPayload(&WorkflowInput{
						InitialActions: []*ActionSet{
							{
								Actions: []*Action{
									PayloadActivity(256, 256, DefaultRemoteActivity),
									PayloadActivity(256, 256, DefaultRemoteActivity),
									PayloadActivity(256, 256, DefaultRemoteActivity),
									NewEmptyReturnResultAction(),
								},
								Concurrent: false,
							},
						},
					}),
				},
				WorkflowId:            fmt.Sprintf("%s/child-%d", run.DefaultStartWorkflowOptions().ID, childID),
				WorkflowIdReusePolicy: enums.WorkflowIdReusePolicy(enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING),
				SearchAttributes: map[string]*common.Payload{
					loadgen.OmesExecutionIDSearchAttribute: &common.Payload{
						Metadata: map[string][]byte{"encoding": []byte("json/plain"), "type": []byte("Keyword")},
						Data:     []byte(fmt.Sprintf("%q", run.ExecutionID)), // quoted to be valid JSON string
					},
				},
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

func (t *tpsExecutor) createSelfSignal() *ClientAction {
	return &ClientAction{
		Variant: &ClientAction_DoSignal{
			DoSignal: &DoSignal{
				Variant: &DoSignal_DoSignalActions_{
					DoSignalActions: &DoSignal_DoSignalActions{
						Variant: &DoSignal_DoSignalActions_DoActions{
							DoActions: SingleActionSet(
								NewTimerAction(t.config.SleepTime),
							),
						},
					},
				},
			},
		},
	}
}

func (t *tpsExecutor) createSelfUpdateWithTimer() *ClientAction {
	return &ClientAction{
		Variant: &ClientAction_DoUpdate{
			DoUpdate: &DoUpdate{
				Variant: &DoUpdate_DoActions{
					DoActions: &DoActionsUpdate{
						Variant: &DoActionsUpdate_DoActions{
							DoActions: SingleActionSet(
								NewTimerAction(t.config.SleepTime),
							),
						},
					},
				},
				WithStart: t.maybeWithStart(0.5),
			},
		},
	}
}

func (t *tpsExecutor) createSelfUpdateWithPayload() *ClientAction {
	return &ClientAction{
		Variant: &ClientAction_DoUpdate{
			DoUpdate: &DoUpdate{
				Variant: &DoUpdate_DoActions{
					DoActions: &DoActionsUpdate{
						Variant: &DoActionsUpdate_DoActions{
							DoActions: SingleActionSet(
								PayloadActivity(0, 256, DefaultRemoteActivity),
							),
						},
					},
				},
				WithStart: t.maybeWithStart(0.5),
			},
		},
	}
}

func (t *tpsExecutor) createSelfUpdateWithPayloadAsLocal() *ClientAction {
	return &ClientAction{
		Variant: &ClientAction_DoUpdate{
			DoUpdate: &DoUpdate{
				Variant: &DoUpdate_DoActions{
					DoActions: &DoActionsUpdate{
						Variant: &DoActionsUpdate_DoActions{
							DoActions: SingleActionSet(
								PayloadActivity(0, 256, t.activityLocality()),
							),
						},
					},
				},
				WithStart: t.maybeWithStart(0.5),
			},
		},
	}
}

func (t *tpsExecutor) createNexusEchoSyncAction() *Action {
	return &Action{
		Variant: &Action_NexusOperation{
			NexusOperation: &ExecuteNexusOperation{
				Endpoint:       t.config.NexusEndpoint,
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
				Endpoint:       t.config.NexusEndpoint,
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
				Endpoint:  t.config.NexusEndpoint,
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

func (t *tpsExecutor) maybeWithStart(likelihood float64) bool {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.rng.Float64() <= likelihood
}

type tpsVerifier struct {
	completionVerifier *loadgen.WorkflowCompletionVerifier
	config             *tpsConfig
}

func (v *tpsVerifier) VerifyRun(ctx context.Context, info loadgen.ScenarioInfo, state loadgen.ExecutorState) []error {
	var errors []error

	// 1. Delegate to completion verifier
	errors = append(errors, v.completionVerifier.VerifyRun(ctx, info, state)...)

	// 2. Check throughput, if configured.
	if v.config.MinThroughputPerHour > 0 {
		// Recalculate expected workflow count for throughput check
		var continueAsNewWorkflows int
		if v.config.ContinueAsNewAfterIter > 0 {
			continueAsNewPerIter := (v.config.InternalIterations - 1) / v.config.ContinueAsNewAfterIter
			continueAsNewWorkflows = continueAsNewPerIter * state.CompletedIterations
		}
		completedChildWorkflows := state.CompletedIterations * v.config.InternalIterations
		completedWorkflows := state.CompletedIterations + completedChildWorkflows + continueAsNewWorkflows

		// Calculate duration from executor state
		var totalDuration time.Duration
		if !state.StartedAt.IsZero() && !state.LastCompletedAt.IsZero() {
			totalDuration = state.LastCompletedAt.Sub(state.StartedAt)
		}

		if totalDuration == 0 {
			errors = append(errors, fmt.Errorf("throughput check: no duration recorded (startedAt=%v, lastCompletedAt=%v)",
				state.StartedAt, state.LastCompletedAt))
		} else {
			actualThroughput := float64(completedWorkflows) / totalDuration.Hours()

			if actualThroughput < v.config.MinThroughputPerHour {
				expectedWorkflows := int(totalDuration.Hours() * v.config.MinThroughputPerHour)
				errors = append(errors, fmt.Errorf("throughput check: %.1f workflows/hour < %.1f required "+
					"(completed %d workflows, expected %d in %v)",
					actualThroughput,
					v.config.MinThroughputPerHour,
					completedWorkflows,
					expectedWorkflows,
					totalDuration.Round(time.Second)))
			}
		}
	}

	return errors
}
