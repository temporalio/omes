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
		var cmd *exec.Cmd
		// The value of the 'kitchen-sink-gen' option is the name of or absolute
		// path to the executable.
		executable, ok := info.ScenarioOptions["kitchen-sink-gen"]
		if ok && executable != "" {
			cmd = exec.CommandContext(ctx, executable, fileOrArgs.Args...)
		} else {
			projDir := filepath.Join(info.RootPath, "loadgen", "kitchen-sink-gen", "Cargo.toml")
			args := []string{"run", "--manifest-path", projDir, "--"}
			args = append(args, fileOrArgs.Args...)
			cmd = exec.CommandContext(ctx, "cargo", args...)
		}
		// We are capturing stderr for use in the error message. This is just for development and
		// should not be merged, since we do really want stderr to go to os.Stderr
		protoBytes, err := cmd.Output()
		if err != nil {
			msg := fmt.Sprintf("failed to run kitchen-sink-gen (%v)", cmd)
			if exitErr, ok := err.(*exec.ExitError); ok {
				msg = fmt.Sprintf("%s: %s", msg, exitErr.Stderr)
			}
			return fmt.Errorf("%s: %w", msg, err)
		}
		asTInput := &kitchensink.TestInput{}
		err = proto.Unmarshal(protoBytes, asTInput)
		testInputs = append(testInputs, asTInput)
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
