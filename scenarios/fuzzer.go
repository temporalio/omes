package scenarios

import (
	"context"
	"time"

	"github.com/temporalio/omes/loadgen"
)

type fuzzerExecutor struct {
	fuzzExecutor       loadgen.FuzzExecutor
	completionVerifier *loadgen.WorkflowCompletionVerifier
}

func (e *fuzzerExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
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
		Description: "This scenario uses the kitchen sink input generation tool to run fuzzy" +
			" workflows",
		ExecutorFn: func() loadgen.Executor {
			return &fuzzerExecutor{
				fuzzExecutor: loadgen.FuzzExecutor{
					InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
						fPath, ok := info.ScenarioOptions["input-file"]
						if ok && fPath != "" {
							return loadgen.FileOrArgs{
								FilePath: fPath,
							}
						}

						args := []string{"generate"}
						seed, ok := info.ScenarioOptions["seed"]
						if ok && seed != "" {
							args = append(args, "--explicit-seed", seed)
						}
						config, ok := info.ScenarioOptions["config"]
						if ok && config != "" {
							args = append(args, "--generator-config-override", config)
						}
						_, ok = info.ScenarioOptions["no-output-file"]
						if !ok {
							args = append(args, "--output-path", "last_fuzz_run.proto")
						}
						return loadgen.FileOrArgs{
							Args: args,
						}
					},
					DefaultConfiguration: loadgen.RunConfiguration{},
				},
			}
		},
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*fuzzerExecutor)
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
