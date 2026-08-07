package loadgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

// DeclareFuzzExecutorOptions declares the options [FuzzExecutor] itself reads.
// Scenarios using it should call this from their own Options declaration so the
// options validate and appear in `list-scenarios`.
func DeclareFuzzExecutorOptions(o *OptionSet) {
	declareNexusEndpointOption(o)
	o.String("kitchen-sink-gen", "", "Name of, or absolute path to, the kitchen-sink-gen executable.")
}

func (k FuzzExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	if k.InitInputs == nil {
		return fmt.Errorf("InitInputs must be specified")
	}

	// The generated actions drive Nexus operations against this endpoint, so it has
	// to exist before they run, and InitInputs below names the same one. An endpoint
	// the user named is theirs to have created.
	if info.OptionString(NexusEndpointOption) == "" {
		if _, err := info.EnsureNexusEndpoint(ctx); err != nil {
			return fmt.Errorf("failed to create nexus endpoint: %w", err)
		}
	}
	info.Logger.Infof("Using nexus endpoint %q", info.NexusEndpointName())

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
		var cmd *exec.Cmd
		// The value of the 'kitchen-sink-gen' option is the name of or absolute
		// path to the executable.
		if executable := info.OptionString("kitchen-sink-gen"); executable != "" {
			cmd = exec.CommandContext(ctx, executable, fileOrArgs.Args...)
		} else {
			projDir := filepath.Join(info.RootPath, "loadgen", "kitchen-sink-gen", "Cargo.toml")
			args := []string{"run", "--manifest-path", projDir, "--"}
			args = append(args, fileOrArgs.Args...)
			cmd = exec.CommandContext(ctx, "cargo", args...)
		}
		cmd.Stderr = os.Stderr
		protoBytes, err := cmd.Output()
		if err != nil {
			return fmt.Errorf("failed to run rust generator (%v): %w", cmd, err)
		}
		asTInput := &kitchensink.TestInput{}
		err = proto.Unmarshal(protoBytes, asTInput)
		testInputs = append(testInputs, asTInput)
	} else {
		return fmt.Errorf("InitInputs must specify either a file or args")
	}

	// Create generic executor and run it
	ge := &GenericExecutor{
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
