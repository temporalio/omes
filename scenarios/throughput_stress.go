package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/throughput_stress"
	"go.temporal.io/sdk/client"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Throughput stress scenario",
		Executor: &loadgen.GenericExecutor{
			DefaultConfiguration: loadgen.RunConfiguration{
				Iterations:    20,
				MaxConcurrent: 5,
			},
			Execute: func(ctx context.Context, run *loadgen.Run) error {
				wfID := fmt.Sprintf("throughputStress-%s-%d", run.ID, run.IterationInTest)
				return run.ExecuteAnyWorkflow(ctx,
					client.StartWorkflowOptions{
						ID:                                       wfID,
						TaskQueue:                                run.TaskQueue(),
						WorkflowExecutionTimeout:                 30 * time.Minute,
						WorkflowExecutionErrorWhenAlreadyStarted: true,
					},
					"throughputStress",
					throughput_stress.WorkflowParams{
						Iterations:                   5,
						ContinueAsNewAfterEventCount: 100,
					})
			},
		},
	})
}
