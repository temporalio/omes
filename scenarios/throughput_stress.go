package scenarios

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughputstress"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

// --option arguments
const (
	IterFlag                   = "internal-iterations"
	IterTimeoutFlag            = "internal-iterations-timeout"
	ContinueAsNewAfterIterFlag = "continue-as-new-after-iterations"
	SkipSleepFlag              = "skip-sleep"
	NexusEndpointFlag          = "nexus-endpoint"
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
	ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"
)

type tpsState struct {
	// CompletedIterations is the number of iteration that have been completed.
	CompletedIterations int `json:"completedIterations"`
	// LastCompletedIterationAt is the time when the last iteration was completed. Helpful for debugging.
	LastCompletedIterationAt time.Time `json:"lastCompletedIterationAt"`
}

type tpsExecutor struct {
	lock       sync.Mutex
	state      *tpsState
	isResuming bool
}

var _ loadgen.Resumable = (*tpsExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' '%s' to control internal parameters",
			IterFlag, ContinueAsNewAfterIterFlag, SkipSleepFlag),
		Executor: &tpsExecutor{state: &tpsState{}},
	})
}

// Return a snapshot of the current state.
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

// Run executes the throughput stress scenario.
//
// It executes `throughputStress` workflows in parallel - up to the configured maximum cocurrency limit - and
// waits for the results. At the end, it verifies that the total number of executed workflows matches Visibility's count.
//
// To resume a previous run, capture the state via the StatusCallback and then set `--option resume-from-state=<state>`.
// Note that the caller is responsible for adjusting the run config's iterations/timeout accordingly.
func (t *tpsExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Parse scenario options
	internalIterations := info.ScenarioOptionInt(IterFlag, 10)
	// When setting a Duration, wait until the end of the entire run before timing out to avoid aborting in the middle of the run.
	// Also, add a buffer to account for the time it takes to wait for the last workflows to complete.
	internalIterTimeout := info.ScenarioOptionDuration(IterTimeoutFlag, cmp.Or(info.Configuration.Duration+1*time.Minute, 1*time.Minute))
	continueAsNewAfterIter := info.ScenarioOptionInt(ContinueAsNewAfterIterFlag, 3)
	nexusEndpoint := info.ScenarioOptions[NexusEndpointFlag] // disabled by default
	skipSleep := info.ScenarioOptionBool(SkipSleepFlag, false)
	skipCleanNamespaceCheck := info.ScenarioOptionBool(SkipCleanNamespaceCheckFlag, false)

	var sleepActivities *throughputstress.SleepActivityConfig
	if sleepActivitiesStr, ok := info.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		sleepActivities, err = throughputstress.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", SleepActivityJsonFlag, err)
		}
	}

	visibilityVerificationTimeout, err := time.ParseDuration(cmp.Or(info.ScenarioOptions[VisibilityVerificationTimeoutFlag], "3m"))
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", VisibilityVerificationTimeoutFlag, err)
	}

	// Add search attribute, if it doesn't exist yet, to query for workflows by run ID.
	// Running this on resume, too, in case a previous Omes run crashed before it could add the search attribute.
	if err = t.initSearchAttribute(ctx, info); err != nil {
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
		err = t.verifyFirstRun(ctx, info, skipCleanNamespaceCheck)
		if err != nil {
			return err
		}
	}

	// Start the scenario run.
	genericExec := &loadgen.GenericExecutor{
		DefaultConfiguration: loadgen.RunConfiguration{
			Iterations:    20,
			MaxConcurrent: 5,
		},
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			wfID := fmt.Sprintf("throughputStress/%s/iter-%d", run.RunID, run.Iteration)

			var result throughputstress.WorkflowOutput
			if err := run.ExecuteAnyWorkflow(ctx,
				client.StartWorkflowOptions{
					ID:                                       wfID,
					TaskQueue:                                run.TaskQueue(),
					WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
					WorkflowExecutionTimeout:                 internalIterTimeout,
					WorkflowExecutionErrorWhenAlreadyStarted: false, // allows resuming from an executor crash
					SearchAttributes: map[string]any{
						ThroughputStressScenarioIdSearchAttribute: run.ScenarioInfo.RunID,
					},
				},
				"throughputStress",
				&result,
				throughputstress.WorkflowParams{
					SkipSleep:                   skipSleep,
					Iterations:                  internalIterations,
					ContinueAsNewAfterIterCount: continueAsNewAfterIter,
					NexusEndpoint:               nexusEndpoint,
					SleepActivities:             sleepActivities,
				},
			); err != nil {
				return err
			}

			t.updateStateOnIterationCompletion(run.Iteration)

			return nil
		},
	}
	if err = genericExec.Run(ctx, info); err != nil {
		return err
	}

	t.lock.Lock()
	var completedIterations = t.state.CompletedIterations
	t.lock.Unlock()
	info.Logger.Info("Total iterations completed: ", completedIterations)

	completedChildWorkflows := completedIterations * internalIterations
	info.Logger.Info("Total child workflows: ", completedChildWorkflows)

	var continueAsNewWorkflows int
	if continueAsNewAfterIter > 0 {
		continueAsNewWorkflows = int(internalIterations/continueAsNewAfterIter) * completedIterations
	}
	info.Logger.Info("Total continue-as-new workflows: ", continueAsNewWorkflows)

	completedWorkflows := completedIterations + completedChildWorkflows + continueAsNewWorkflows
	info.Logger.Info("Total workflows completed: ", completedWorkflows)

	// Post-scenario, verify reported count from Visibility matches the expected count.
	return loadgen.VisibilityCountIsEventually(
		ctx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				ThroughputStressScenarioIdSearchAttribute, info.RunID),
		},
		completedWorkflows,
		visibilityVerificationTimeout,
	)
}

func (t *tpsExecutor) initSearchAttribute(ctx context.Context, info loadgen.ScenarioInfo) error {
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
