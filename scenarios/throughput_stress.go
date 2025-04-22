package scenarios

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughputstress"
	"go.temporal.io/sdk/client"
)

// --option arguments
const (
	IterFlag          = "internal-iterations"
	IterTimeout       = "internal-iterations-timeout"
	SkipSleepFlag     = "skip-sleep"
	CANEventFlag      = "continue-as-new-after-event-count"
	NexusEndpointFlag = "nexus-endpoint"
	// WorkflowIDPrefix is the prefix for each run's workflow ID. Use it to ensure that the workflow IDs are unique.
	WorkflowIDPrefix = "workflow-id-prefix"
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

type tpsExecutor struct {
	workflowCount atomic.Uint64
}

func (t *tpsExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Parse scenario options
	internalIterations := info.ScenarioOptionInt(IterFlag, 5)
	internalIterTimeout := info.ScenarioOptionDuration(IterTimeout, time.Minute)
	continueAsNewCount := info.ScenarioOptionInt(CANEventFlag, 120)
	workflowIDPrefix := cmp.Or(info.ScenarioOptions[WorkflowIDPrefix], defaultWorkflowIDPrefix)
	nexusEndpoint := info.ScenarioOptions[NexusEndpointFlag] // disabled by default
	skipSleep := info.ScenarioOptionBool(SkipSleepFlag, false)

	var sleepActivityPerPriority throughputstress.SleepActivity
	if sleepActivitiesWithPriorityStr, ok := info.ScenarioOptions[SleepActivityPerPriorityJsonFlag]; ok {
		err := json.Unmarshal([]byte(sleepActivitiesWithPriorityStr), &sleepActivityPerPriority)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", SleepActivityPerPriorityJsonFlag, err)
		}
	}

	visibilityVerificationTimeout, err := time.ParseDuration(cmp.Or(info.ScenarioOptions[VisibilityVerificationTimeout], "3m"))
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", VisibilityVerificationTimeout, err)
	}
	timeout := time.Duration(1*internalIterations) * internalIterTimeout

	// Make sure the search attribute is registered
	attribMap := map[string]enums.IndexedValueType{
		ThroughputStressScenarioIdSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
	}

	_, err = info.Client.OperatorService().AddSearchAttributes(ctx,
		&operatorservice.AddSearchAttributesRequest{
			Namespace:        info.Namespace,
			SearchAttributes: attribMap,
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

	// Complain if there are already existing workflows with the provided run id
	visibilityCount, err := info.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: info.Namespace,
		Query:     fmt.Sprintf("%s='%s'", ThroughputStressScenarioIdSearchAttribute, info.RunID),
	})
	if err != nil {
		return err
	}
	if visibilityCount.Count > 0 {
		return fmt.Errorf("there are already %d workflows with scenario Run ID '%s'",
			visibilityCount.Count, info.RunID)
	}

	genericExec := &loadgen.GenericExecutor{
		DefaultConfiguration: loadgen.RunConfiguration{
			Iterations:    20,
			MaxConcurrent: 5,
		},
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			wfID := fmt.Sprintf("%s-%s-%d", workflowIDPrefix, run.RunID, run.Iteration)
			var result throughputstress.WorkflowOutput
			err = run.ExecuteAnyWorkflow(ctx,
				client.StartWorkflowOptions{
					ID:                                       wfID,
					TaskQueue:                                run.TaskQueue(),
					WorkflowExecutionTimeout:                 timeout,
					WorkflowExecutionErrorWhenAlreadyStarted: true,
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
			// The 1 is for the final workflow run
			t.workflowCount.Add(uint64(result.TimesContinued + result.ChildrenSpawned + 1))
			return err
		},
	}
	err = genericExec.Run(ctx, info)
	if err != nil {
		return err
	}

	// Post-scenario, verify visibility counts
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
