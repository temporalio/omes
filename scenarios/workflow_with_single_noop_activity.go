package scenarios

import (
	"context"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type noopActivityExecutor struct {
	*loadgen.KitchenSinkExecutor
	completionVerifier *loadgen.WorkflowCompletionVerifier
}

func (e *noopActivityExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
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
		Description: "Each iteration executes a single workflow with a noop activity.",
		ExecutorFn: func() loadgen.Executor {
			return &noopActivityExecutor{
				KitchenSinkExecutor: &loadgen.KitchenSinkExecutor{
					TestInput: &kitchensink.TestInput{
						WorkflowInput: &kitchensink.WorkflowInput{
							InitialActions: []*kitchensink.ActionSet{
								kitchensink.NoOpSingleActivityActionSet(),
							},
						},
					},
				},
			}
		},
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			// e := executor.(*noopActivityExecutor)
			// if e.completionVerifier == nil {
			// 	return nil
			// }
			// state := e.KitchenSinkExecutor.GetState()
			// return e.completionVerifier.VerifyRun(ctx, info, state)
			return nil
		},
	})
}
