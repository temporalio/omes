package scenarios

import (
	"cmp"
	"context"
	"encoding/json"
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
	IterFlag    = "internal-iterations"
	IterTimeout = "internal-iterations-timeout"
	// IterResumeFromState is the iteration state to resume from. If set, it will skip the
	// run initialization and start the next iteration immediately where the previous run left off.
	// The state is emitted by the scenario's StatusCallback.
	IterResumeFromState = "resume-from-state"
	SkipSleepFlag       = "skip-sleep"
	CANEventFlag        = "continue-as-new-after-event-count"
	NexusEndpointFlag   = "nexus-endpoint"
	// VisibilityVerificationTimeout is the timeout for verifying the total visibility count at the end of the scenario.
	// It needs to account for a backlog of tasks and, if used, ElasticSearch's eventual consistency.
	VisibilityVerificationTimeout = "visibility-count-timeout"
	// SleepActivityPerPriorityJsonFlag is a JSON string that defines the sleep activity's priorities and sleep duration.
	// See throughputstress.SleepActivity for more details.
	SleepActivityPerPriorityJsonFlag = "sleep-activity-per-priority-json"
)

const (
	ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"
)

type tpsState struct {
	// CompletedIteration is the iteration that has been completed.
	CompletedIteration int `json:"completedIteration"`
	// WorkflowCount is the total number of workflows that have been completed so far.
	WorkflowCount int `json:"workflowCount"`
}

type tpsExecutor struct {
	lock             sync.Mutex
	state            *tpsState
	completedIterMap map[int]struct{}
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
	internalIterations := info.ScenarioOptionInt(IterFlag, 5)
	internalIterTimeout := info.ScenarioOptionDuration(IterTimeout, time.Minute)
	resumeFromState, resumingFromPreviousRun := info.ScenarioOptions[IterResumeFromState]
	continueAsNewCount := info.ScenarioOptionInt(CANEventFlag, 120)
	nexusEndpoint := info.ScenarioOptions[NexusEndpointFlag] // disabled by default
	skipSleep := info.ScenarioOptionBool(SkipSleepFlag, false)

	var sleepActivityPerPriority *throughputstress.SleepActivity
	if sleepActivitiesWithPriorityStr, ok := info.ScenarioOptions[SleepActivityPerPriorityJsonFlag]; ok {
		sleepActivityPerPriority = &throughputstress.SleepActivity{}
		err := json.Unmarshal([]byte(sleepActivitiesWithPriorityStr), sleepActivityPerPriority)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", SleepActivityPerPriorityJsonFlag, err)
		}
	}

	visibilityVerificationTimeout, err := time.ParseDuration(cmp.Or(info.ScenarioOptions[VisibilityVerificationTimeout], "3m"))
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", VisibilityVerificationTimeout, err)
	}
	timeout := time.Duration(1*internalIterations) * internalIterTimeout

	// Initialize the scenario run.
	var lastCompletedIter int
	if resumingFromPreviousRun {
		info.Logger.Info(fmt.Sprintf("Resuming scenario from state: %s", resumeFromState))
		err := json.Unmarshal([]byte(resumeFromState), &t.state)
		if err != nil {
			return fmt.Errorf("failed to unmarshal scenario's resume state: %w", err)
		}
		lastCompletedIter = int(t.state.CompletedIteration)
	} else {
		err = t.initFirstRun(ctx, info)
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
			curIteration := lastCompletedIter + run.Iteration
			wfID := fmt.Sprintf("throughputStress/%s/iter-%d", run.RunID, curIteration)

			var result throughputstress.WorkflowOutput
			err = run.ExecuteAnyWorkflow(ctx,
				client.StartWorkflowOptions{
					ID:                                       wfID,
					TaskQueue:                                run.TaskQueue(),
					WorkflowExecutionTimeout:                 timeout,
					WorkflowExecutionErrorWhenAlreadyStarted: !resumingFromPreviousRun, // don't fail when resuming
					SearchAttributes: map[string]interface{}{
						ThroughputStressScenarioIdSearchAttribute: run.ScenarioInfo.RunID,
					},
				},
				"throughputStress",
				&result,
				throughputstress.WorkflowParams{
					SkipSleep:                    skipSleep,
					Iterations:                   internalIterations,
					ContinueAsNewAfterEventCount: continueAsNewCount,
					NexusEndpoint:                nexusEndpoint,
					SleepActivityPerPriority:     sleepActivityPerPriority,
				})

			if err == nil {
				t.updateState(result, curIteration)

				if info.StatusCallback != nil {
					stateJson, err := json.Marshal(t.state)
					if err != nil {
						info.Logger.Warn("failed to marshal scenario's state: ", err)
					} else {
						info.StatusCallback(string(stateJson))
					}
				}
			}

			return err
		},
	}
	err = genericExec.Run(ctx, info)
	if err != nil {
		return err
	}

	// Post-scenario, verify reported count from Visibility matches the expected count.
	info.Logger.Info("Total workflows executed: ", t.state.WorkflowCount)
	return loadgen.VisibilityCountIsEventually(
		ctx,
		info.Client,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				ThroughputStressScenarioIdSearchAttribute, info.RunID),
		},
		int(t.state.WorkflowCount),
		visibilityVerificationTimeout,
	)
}

func (t *tpsExecutor) initFirstRun(ctx context.Context, info loadgen.ScenarioInfo) error {
	info.Logger.Infof("Initialising Search Attribute %s", ThroughputStressScenarioIdSearchAttribute)

	// Add search attribute, if it doesn't exist yet, to query for workflows by run ID.
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

	if info.StatusCallback != nil {
		// Init empty state to trigger a resume on a re-run.
		info.StatusCallback("{}")
	}

	return nil
}

func (t *tpsExecutor) updateState(result throughputstress.WorkflowOutput, completedIter int) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Update the workflow count. (the 1 is for the final workflow run)
	t.state.WorkflowCount += result.TimesContinued + result.ChildrenSpawned + 1

	// Update the completed iteration.
	t.completedIterMap[completedIter] = struct{}{}
	for {
		nextCompletedIter := t.state.CompletedIteration + 1
		if _, ok := t.completedIterMap[nextCompletedIter]; ok {
			t.state.CompletedIteration = nextCompletedIter
		} else {
			break
		}
	}
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' '%s' to control internal parameters",
			IterFlag, CANEventFlag, SkipSleepFlag),
		Executor: &tpsExecutor{
			state:            &tpsState{},
			completedIterMap: map[int]struct{}{},
		},
	})
}
