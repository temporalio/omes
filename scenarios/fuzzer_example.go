package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario runs the kitchen sink input generation tool `example` " +
			"command to help with basic verification of KS implementations.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.FuzzExecutor{
				InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
					return loadgen.FileOrArgs{
						Args: []string{"example"},
					}
				},
				DefaultConfiguration: loadgen.RunConfiguration{},
			}
		},
	})
}
