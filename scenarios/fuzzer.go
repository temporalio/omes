package scenarios

import (
	"context"
	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario uses the kitchen sink input generation tool to run fuzzy" +
			" workflows",
		Executor: loadgen.FuzzExecutor{
			InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
				args := []string{"generate"}
				seed, ok := info.ScenarioOptions["seed"]
				if ok && seed != "" {
					args = append(args, "--explicit-seed", seed)
				}
				return loadgen.FileOrArgs{
					Args: args,
				}
			},
			DefaultConfiguration: loadgen.RunConfiguration{},
		},
	})
}
