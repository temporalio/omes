package scenarios

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
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
			o.String("load-input-file", "", "LoadInput file to replay, with generated search attributes and "+
				"visibility reads. A .json path is read as protojson, any other path as binary proto. "+
				"See docs/load-input.md.")
			o.String("seed", "", "Explicit seed for action generation.")
			o.String("config", "", "Generator config override.")
			o.String("output-file", defaultFuzzOutputFile,
				"Path to write the generated actions to for replay. Empty writes nothing.")
		},
		ExecutorFn: func() loadgen.Executor {
			return loadgen.FuzzExecutor{
				InitLoadInput: func(ctx context.Context, info loadgen.ScenarioInfo) (*kitchensink.LoadInput, error) {
					loadPath := info.OptionString("load-input-file")
					if loadPath == "" {
						return nil, nil
					}
					// These options only shape the input file or generated actions that a LoadInput replaces.
					for _, name := range []string{"input-file", "seed", "config", "output-file", "kitchen-sink-gen"} {
						if info.OptionUserSpecified(name) {
							return nil, fmt.Errorf("option %s can't be used with load-input-file", name)
						}
					}
					return loadgen.ReadLoadInput(loadPath)
				},
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
					args = append(args, "--nexus-endpoint", info.NexusEndpointName())
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
