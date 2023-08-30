package scenarios

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughputstress"
	"go.temporal.io/sdk/client"
)

// --option flags
const (
	IterFlag     = "internal-iterations"
	CANEventFlag = "continue-as-new-after-event-count"
)

const ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"

type ThroughputStressStorage struct {
	workflowCount atomic.Uint64
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --%s and --%s to control internal parameters",
			IterFlag, CANEventFlag),
		Executor: &loadgen.GenericExecutor[ThroughputStressStorage]{
			DefaultConfiguration: loadgen.RunConfiguration{
				Iterations:    20,
				MaxConcurrent: 5,
			},
			PreScenarioImpl: func(ctx context.Context, info loadgen.ScenarioInfo, storage *ThroughputStressStorage) error {
				storage = &ThroughputStressStorage{
					workflowCount: atomic.Uint64{},
				}
				attribMap := map[string]enums.IndexedValueType{
					ThroughputStressScenarioIdSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
				}
				_, err := info.Client.OperatorService().AddSearchAttributes(ctx,
					&operatorservice.AddSearchAttributesRequest{
						Namespace:        info.Namespace,
						SearchAttributes: attribMap,
					})
				var svcErr *serviceerror.AlreadyExists
				if errors.As(err, &svcErr) {
					return nil
				}
				return err
			},
			Execute: func(ctx context.Context, run *loadgen.Run, storage *ThroughputStressStorage) error {
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
				storage.workflowCount.Add(uint64(result.TimesContinued + result.ChildrenSpawned + 1))
				return err
			},
			PostScenarioImpl: func(ctx context.Context, info loadgen.ScenarioInfo, storage *ThroughputStressStorage) error {
				totalWorkflowCount := storage.workflowCount.Load()
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
			},
		},
	})
}
