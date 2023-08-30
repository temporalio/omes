package scenarios

import (
	"context"
	"errors"
	"fmt"
	"go.temporal.io/api/workflowservice/v1"
	"sync/atomic"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughputstress"
	"go.temporal.io/sdk/client"
)

// --option arguments
const (
	IterFlag     = "internal-iterations"
	CANEventFlag = "continue-as-new-after-event-count"
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
	var svcErr *serviceerror.AlreadyExists
	if !errors.As(err, &svcErr) {
		return err
	}

	genericExec := &loadgen.GenericExecutor{
		DefaultConfiguration: loadgen.RunConfiguration{
			Iterations:    20,
			MaxConcurrent: 5,
		},
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			internalIterations := run.ScenarioInfo.ScenarioOptionInt(IterFlag, 5)
			continueAsNewCount := run.ScenarioInfo.ScenarioOptionInt(CANEventFlag, 100)
			timeout := time.Duration(1*internalIterations) * time.Minute

			wfID := fmt.Sprintf("throughputStress-%s-%d", run.RunID, run.Iteration)
			var result throughputstress.WorkflowOutput
			err := run.ExecuteAnyWorkflow(ctx,
				client.StartWorkflowOptions{
					ID:                                       wfID,
					TaskQueue:                                run.TaskQueue(),
					WorkflowExecutionTimeout:                 timeout,
					WorkflowExecutionErrorWhenAlreadyStarted: true,
					SearchAttributes: map[string]interface{}{
						ThroughputStressScenarioIdSearchAttribute: run.ScenarioInfo.UniqueRunID(),
					},
				},
				"throughputStress",
				&result,
				throughputstress.WorkflowParams{
					Iterations:                   internalIterations,
					ContinueAsNewAfterEventCount: continueAsNewCount,
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
				ThroughputStressScenarioIdSearchAttribute, info.UniqueRunID()),
		},
		int(totalWorkflowCount),
		3*time.Minute,
	)

}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --%s and --%s to control internal parameters",
			IterFlag, CANEventFlag),
		Executor: &tpsExecutor{
			workflowCount: atomic.Uint64{},
		},
	})
}
