package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
)

// defaultFuzzOutputFile is where generated actions are written so a run can be
// replayed with --option input-file.
const defaultFuzzOutputFile = "last_fuzz_run.proto"

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "This scenario uses the kitchen sink input generation tool to run fuzzy" +
			" workflows",
		Options: func(o *loadgen.OptionSet) {
			loadgen.DeclareFuzzExecutorOptions(o)
			o.String("input-file", "", "Pre-generated input file to replay instead of generating actions.")
			o.String("seed", "", "Explicit seed for action generation.")
			o.String("config", "", "Generator config override.")
			o.String("output-file", defaultFuzzOutputFile,
				"Path to write the generated actions to for replay. Empty writes nothing.")
		},
		ExecutorFn: func() loadgen.Executor {
			return loadgen.FuzzExecutor{
				InitInputs: func(ctx context.Context, info loadgen.ScenarioInfo) loadgen.FileOrArgs {
					if fPath := info.OptionString("input-file"); fPath != "" {
						return loadgen.FileOrArgs{
							FilePath: fPath,
						}
					}

					args := []string{"generate"}
					if seed := info.OptionString("seed"); seed != "" {
						args = append(args, "--explicit-seed", seed)
					}
					if config := info.OptionString("config"); config != "" {
						args = append(args, "--generator-config-override", config)
					}
					nexusEndpoint := info.OptionString("nexus-endpoint")
					if nexusEndpoint == "" {
						nexusEndpoint = loadgen.NexusEndpointForRun(info.RunID)
					}
					args = append(args, "--nexus-endpoint", nexusEndpoint)
					if outputFile := info.OptionString("output-file"); outputFile != "" {
						args = append(args, "--output-path", outputFile)
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
