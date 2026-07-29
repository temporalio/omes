package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario uses the kitchen sink input generation tool to run fuzzy" +
			" workflows",
		Options: func(o *loadgen.OptionSet) {
			loadgen.DeclareFuzzExecutorOptions(o)
			o.String("input-file", "", "Pre-generated input file to replay instead of generating actions.")
			o.String("seed", "", "Explicit seed for action generation.")
			o.String("config", "", "Generator config override.")
			o.Bool("no-output-file", false, "Skip writing last_fuzz_run.proto.")
		},
		ExecutorFn: func() loadgen.Executor {
			return loadgen.FuzzExecutor{
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
					nexusEndpoint := info.ScenarioOptions["nexus-endpoint"]
					if nexusEndpoint == "" {
						nexusEndpoint = loadgen.NexusEndpointForRun(info.RunID)
					}
					args = append(args, "--nexus-endpoint", nexusEndpoint)
					if !info.ScenarioOptionBool("no-output-file", false) {
						args = append(args, "--output-path", "last_fuzz_run.proto")
					}
					return loadgen.FileOrArgs{
						Args: args,
					}
				},
				DefaultConfiguration: loadgen.RunConfiguration{},
			}
		},
	})
}
