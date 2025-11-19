package scenarios

import (
	"context"
	"time"

	"github.com/temporalio/omes/loadgen"
)

type fuzzerExampleExecutor struct {
	fuzzExecutor       loadgen.FuzzExecutor
	completionVerifier *loadgen.WorkflowCompletionVerifier
}

func (e *fuzzerExampleExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Create completion verifier
	verifier, err := loadgen.NewWorkflowCompletionChecker(ctx, info, 30*time.Second)
	if err != nil {
		return err
	}
	e.completionVerifier = verifier

	// Run the fuzz executor
	return e.fuzzExecutor.Run(ctx, info)
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario runs the kitchen sink input generation tool `example` " +
			"command to help with basic verification of KS implementations.",
		ExecutorFn: func() loadgen.Executor {
			return &fuzzerExampleExecutor{
				fuzzExecutor: loadgen.FuzzExecutor{
					InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
						return loadgen.FileOrArgs{
							Args: []string{"example"},
						}
					},
					DefaultConfiguration: loadgen.RunConfiguration{},
				},
			}
		},
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*fuzzerExampleExecutor)
			if e.completionVerifier == nil {
				return nil
			}
			// Get state from the embedded generic executor (FuzzExecutor creates one internally)
			state := loadgen.ExecutorState{
				CompletedIterations: info.Configuration.Iterations,
			}
			return e.completionVerifier.VerifyRun(ctx, info, state)
		},
	})
}
