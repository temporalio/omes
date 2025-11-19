package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type manyTaskQueuesExecutor struct {
	*loadgen.KitchenSinkExecutor
	completionVerifier *loadgen.WorkflowCompletionVerifier
}

func (e *manyTaskQueuesExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Create completion verifier
	verifier, err := loadgen.NewWorkflowCompletionChecker(ctx, info, 30*time.Second)
	if err != nil {
		return err
	}
	e.completionVerifier = verifier

	// Run the kitchen sink executor
	return e.KitchenSinkExecutor.Run(ctx, info)
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow on one of the task queues. " +
			"Workers must be started with --task-queue-suffix-index-end as one less than task queue count here. " +
			"Additional options: task-queue-count (required).",
		ExecutorFn: func() loadgen.Executor {
			return &manyTaskQueuesExecutor{
				KitchenSinkExecutor: &loadgen.KitchenSinkExecutor{
					TestInput: &kitchensink.TestInput{
						WorkflowInput: &kitchensink.WorkflowInput{
							InitialActions: []*kitchensink.ActionSet{
								kitchensink.NoOpSingleActivityActionSet(),
							},
						},
					},
					PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
						// Require task queue count
						if opts.ScenarioOptionInt("task-queue-count", 0) == 0 {
							return fmt.Errorf("task-queue-count option required")
						}
						return nil
					},
					UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
						// Add suffix to the task queue based on modulus of iteration
						options.StartOptions.TaskQueue +=
							fmt.Sprintf("-%v", run.Iteration%run.ScenarioInfo.ScenarioOptionInt("task-queue-count", 0))
						return nil
					},
				},
			}
		},
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*manyTaskQueuesExecutor)
			if e.completionVerifier == nil {
				return nil
			}
			state := e.KitchenSinkExecutor.GetState()
			return e.completionVerifier.VerifyRun(ctx, info, state)
		},
	})
}
