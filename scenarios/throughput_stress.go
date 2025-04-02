package scenarios

import (
	"context"
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
)

const ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"

type tpsExecutor struct {
	workflowCount atomic.Uint64
}

func (t *tpsExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Make sure the search attribute is registered
	attribMap := map[string]enums.IndexedValueType{
		ThroughputStressScenarioIdSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
	}

	_, err := info.Client.OperatorService().AddSearchAttributes(ctx,
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
			internalIterations := run.ScenarioInfo.ScenarioOptionInt(IterFlag, 5)
			internalIterTimeout := run.ScenarioInfo.ScenarioOptionDuration(IterTimeout, time.Minute)
			continueAsNewCount := run.ScenarioInfo.ScenarioOptionInt(CANEventFlag, 120)
			// Disabled by default.
			nexusEndpoint := run.ScenarioInfo.ScenarioOptions[NexusEndpointFlag]
			skipSleep := run.ScenarioInfo.ScenarioOptionBool(SkipSleepFlag, false)
			timeout := time.Duration(internalIterations) * internalIterTimeout

			wfID := fmt.Sprintf("throughputStress-%s-%d", run.RunID, run.Iteration)
			var result throughputstress.WorkflowOutput
			err := run.ExecuteAnyWorkflow(ctx,
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
		3*time.Minute,
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
