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
	})
}
