package scenarios

import (
	"fmt"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughputstress"
)

// --option arguments
const (
	IterFlag     = "internal-iterations"
	CANEventFlag = "continue-as-new-after-event-count"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Throughput stress scenario. Use --%s and --%s to control internal parameters",
			IterFlag, CANEventFlag),
		Executor: &loadgen.WorkflowExecutor{
			WorkflowType: "ThroughputStressExecutorWorkflow",
			WorkflowArgCreator: func(info loadgen.ScenarioInfo) []interface{} {
				internalIterations := info.ScenarioOptionInt(IterFlag, 5)
				continueAsNewCount := info.ScenarioOptionInt(CANEventFlag, 100)
				return []interface{}{throughputstress.ExecutorWorkflowInput{
					Iterations:    20,
					MaxConcurrent: 5,
					RunID:         info.RunID,
					LoadParams: throughputstress.WorkflowParams{
						Iterations:                   internalIterations,
						ContinueAsNewAfterEventCount: continueAsNewCount,
					},
				}}
			},
		},
	})
}
