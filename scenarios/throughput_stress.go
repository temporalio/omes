package scenarios

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
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
	// IterResumeAt is the iteration from which to resume a previous run from. If set, it will skip the
	// run initialization and start the next iteration starting from the given iteration.
	IterResumeAt      = "internal-iterations-resume-at"
	SkipSleepFlag     = "skip-sleep"
	CANEventFlag      = "continue-as-new-after-event-count"
	NexusEndpointFlag = "nexus-endpoint"
	// WorkflowIDPrefix is the prefix for each run's workflow ID. Use it to ensure that the workflow IDs are unique.
	WorkflowIDPrefix = "workflow-id-prefix"
	// WorkflowCountResumeAt is the number of completed workflows from a previous run that is being resumed.
	WorkflowCountResumeAt = "workflow-count-resume-at"
	// VisibilityVerificationTimeout is the timeout for verifying the total visibility count at the end of the scenario.
	// It needs to account for a backlog of tasks and, if used, ElasticSearch's eventual consistency.
	VisibilityVerificationTimeout = "visibility-count-timeout"
	// SleepActivityPerPriorityJsonFlag is a JSON string that defines the sleep activity's priorities and sleep duration.
	// See throughputstress.SleepActivity for more details.
	SleepActivityPerPriorityJsonFlag = "sleep-activity-per-priority-json"
)

const (
	ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"
	defaultWorkflowIDPrefix                   = "throughputStress"
)

type ThroughputStressScenarioStatusUpdate struct {
	// CompletedIteration is the iteration that has been completed.
	CompletedIteration int
	// CompletedWorkflows is the total number of workflows that have been completed so far.
	CompletedWorkflows uint64
}

type tpsExecutor struct {
	workflowCount atomic.Uint64
}

// Run executes the throughput stress scenario.
//
// It executes `throughputStress` workflows in parallel - up to the configured maximum cocurrency limit - and
// waits for the results. At the end, it verifies that the total number of executed workflows matches Visibility's count.
//
// To resume a previous run, set the following options:
//
// --option internal-iterations-resume-at=<value>
// --option workflow-count-resume-at=<value>
//
// Note that the caller is responsible for adjusting the scenario's iterations/timeout accordingly.
func (t *tpsExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Parse scenario options
	internalIterations := info.ScenarioOptionInt(IterFlag, 5)
	internalIterTimeout := info.ScenarioOptionDuration(IterTimeout, time.Minute)
	internalIterResumeFrom := info.ScenarioOptionInt(IterResumeAt, 0)
	_, resumingFromPreviousRun := info.ScenarioOptions[IterResumeAt]
	continueAsNewCount := info.ScenarioOptionInt(CANEventFlag, 120)
	workflowIDPrefix := cmp.Or(info.ScenarioOptions[WorkflowIDPrefix], defaultWorkflowIDPrefix)
	workflowCountStartAt := info.ScenarioOptionInt(WorkflowCountResumeAt, 0)
	nexusEndpoint := info.ScenarioOptions[NexusEndpointFlag] // disabled by default
	skipSleep := info.ScenarioOptionBool(SkipSleepFlag, false)
	if info.StatusCallback == nil {
		info.StatusCallback = func(data any) {}
	}

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
	if !resumingFromPreviousRun {
		err = t.initFirstRun(ctx, info)
		if err != nil {
			return err
		}
	} else {
		info.Logger.Info("Resuming from previous run")
		t.workflowCount.Add(uint64(workflowCountStartAt))
	}

	// Start the scenario run.
	genericExec := &loadgen.GenericExecutor{
		DefaultConfiguration: loadgen.RunConfiguration{
			Iterations:    20,
			MaxConcurrent: 5,
		},
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			curIteration := internalIterResumeFrom + run.Iteration
			wfID := fmt.Sprintf("%s/%s/iter-%d", workflowIDPrefix, run.RunID, curIteration)

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
				// The 1 is for the final workflow run.
				curTotal := t.workflowCount.Add(uint64(result.TimesContinued + result.ChildrenSpawned + 1))

				info.StatusCallback(ThroughputStressScenarioStatusUpdate{
					CompletedIteration: curIteration,
					CompletedWorkflows: curTotal,
				})
			}

			return err
		},
	}
	err = genericExec.Run(ctx, info)
	if err != nil {
		return err
	}

	// Post-scenario, verify reported count from Visibility matches the expected count.
	totalWorkflowCount := t.workflowCount.Load()
	info.Logger.Info("Total workflows executed: ", totalWorkflowCount)
	return loadgen.VisibilityCountIsEventually(
		ctx,
		info.Client,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				ThroughputStressScenarioIdSearchAttribute, info.RunID),
		},
		int(totalWorkflowCount),
		visibilityVerificationTimeout,
	)
}

func (t *tpsExecutor) initFirstRun(ctx context.Context, info loadgen.ScenarioInfo) error {
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

	return nil
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --option with '%s', '%s' '%s' to control internal parameters",
			IterFlag, CANEventFlag, SkipSleepFlag),
		Executor: &tpsExecutor{
			workflowCount: atomic.Uint64{},
		},
	})
}
