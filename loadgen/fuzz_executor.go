package loadgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type FileOrArgs struct {
	// If set, the file to load the input from
	FilePath string
	// If set, args to pass to the Rust input generator. Do not specify output args, as it is
	// expected that the proto output is written to stdout (the default).
	Args []string
}

type FuzzExecutor struct {
	// Must be specified, called once on startup, and determines how TestInputs will be used for
	// iterations of the scenario. If a file is specified, it will be loaded and used as the input
	// for every iteration. If generator args are specified, it will be invoked once per iteration
	// and those inputs will be saved and then fed out to each iteration.
	InitInputs func(context.Context, ScenarioInfo) FileOrArgs

	DefaultConfiguration RunConfiguration
}

func (k FuzzExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	if k.InitInputs == nil {
		return fmt.Errorf("InitInputs must be specified")
	}
	var testInputs []*kitchensink.TestInput
	// Load or generate inputs
	fileOrArgs := k.InitInputs(ctx, info)
	if fileOrArgs.FilePath != "" {
		fDat, err := os.ReadFile(fileOrArgs.FilePath)
		asTInput := &kitchensink.TestInput{}
		err = proto.Unmarshal(fDat, asTInput)
		if err != nil {
			return fmt.Errorf("failed to unmarshal test input from file on disk: %w", err)
		}
		testInputs = append(testInputs, asTInput)
	} else if fileOrArgs.Args != nil {
		// Check if using external executable first
		executable, ok := info.ScenarioOptions["kitchen-sink-gen"]
		if ok && executable != "" {
			// Use external executable
			cmd := exec.CommandContext(ctx, executable, fileOrArgs.Args...)
			cmd.Stderr = os.Stderr
			protoBytes, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("failed to run kitchen-sink generator (%v): %w", cmd, err)
			}
			asTInput := &kitchensink.TestInput{}
			err = proto.Unmarshal(protoBytes, asTInput)
			testInputs = append(testInputs, asTInput)
		} else {
			// Use integrated Go generator
			asTInput, err := generateKitchenSinkInput(fileOrArgs.Args, info.ScenarioOptions)
			if err != nil {
				return fmt.Errorf("failed to generate kitchen sink input: %w", err)
			}
			testInputs = append(testInputs, asTInput)
		}
	} else {
		return fmt.Errorf("InitInputs must specify either a file or args")
	}

	// Create generic executor and run it
	ge := &GenericExecutor{
		DefaultConfiguration: k.DefaultConfiguration,
		Execute: func(ctx context.Context, run *Run) error {
			options := run.DefaultKitchenSinkWorkflowOptions()
			// TODO: Iterations
			testInputClone, ok := proto.Clone(testInputs[0]).(*kitchensink.TestInput)
			if !ok {
				panic("failed to clone test input")
			}
			options.Params = testInputClone
			// Run the workflow while we perform the client actions in the background
			return run.ExecuteKitchenSinkWorkflow(ctx, &options)
		},
	}
	return ge.Run(ctx, info)
}

func (k FuzzExecutor) GetDefaultConfiguration() RunConfiguration {
	return k.DefaultConfiguration
}

// generateKitchenSinkInput generates kitchen sink input using the integrated generator
func generateKitchenSinkInput(args []string, options map[string]string) (*kitchensink.TestInput, error) {
	// Parse arguments
	var command string
	var seed *uint64
	var configOverride *string

	for i, arg := range args {
		switch arg {
		case "generate":
			command = "generate"
		case "example":
			command = "example"
		case "--seed", "-seed":
			if i+1 < len(args) {
				seedVal, err := strconv.ParseUint(args[i+1], 10, 64)
				if err != nil {
					return nil, fmt.Errorf("invalid seed value: %w", err)
				}
				seed = &seedVal
			}
		case "--generator-config-override":
			if i+1 < len(args) {
				configOverride = &args[i+1]
			}
		}
	}

	// Handle scenario options
	if seedStr, ok := options["seed"]; ok && seedStr != "" {
		seedVal, err := strconv.ParseUint(seedStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid seed option value: %w", err)
		}
		seed = &seedVal
	}

	if configStr, ok := options["config"]; ok && configStr != "" {
		configOverride = &configStr
	}

	switch command {
	case "example":
		return GenerateExampleKitchenSinkInput(), nil
	case "generate":
		var seedVal uint64
		if seed != nil {
			seedVal = *seed
		} else {
			var err error
			seedVal, err = GenerateRandomSeed()
			if err != nil {
				return nil, fmt.Errorf("failed to generate random seed: %w", err)
			}
		}

		return GenerateKitchenSinkTestInput(seedVal, configOverride)
	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}
