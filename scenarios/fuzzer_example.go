package scenarios

import (
	"context"
	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario uses the kitchen sink input generation tool to run fuzzy workflows",
		Executor: loadgen.FuzzExecutor{
			InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
				return loadgen.FileOrArgs{
					Args: []string{"example"},
				}
			},
			DefaultConfiguration: loadgen.RunConfiguration{},
		},
	})
}
