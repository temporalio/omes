package loadgen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/loadgen/visibility"
)

type FileOrArgs struct {
	// If set, the file to load the input from
	FilePath string
	// If set, args to pass to the Rust input generator. Do not specify output args, as it is
	// expected that the proto output is written to stdout (the default).
	Args []string
}

type FuzzExecutor struct {
	// Must be specified unless InitLoadInput always returns a LoadInput. Called once on startup, and
	// determines how TestInputs will be used for iterations of the scenario. If a file is specified,
	// it will be loaded and used as the input for every iteration. If generator args are specified,
	// it will be invoked once per iteration and those inputs will be saved and then fed out to each
	// iteration.
	InitInputs func(context.Context, ScenarioInfo) FileOrArgs

	// If set, called once on startup before InitInputs. A non-nil LoadInput takes the place of
	// InitInputs: each iteration materializes one of its weighted inputs, and the run issues its
	// visibility reads.
	InitLoadInput func(context.Context, ScenarioInfo) (*kitchensink.LoadInput, error)

	DefaultConfiguration RunConfiguration

	// Tests replace the server with these. Unset, workflows run through Run.ExecuteKitchenSinkWorkflow and reads go
	// to ScenarioInfo.Client.
	executeWorkflow  func(context.Context, *Run, *KitchenSinkWorkflowOptions) error
	visibilityClient visibility.Client
}

// DeclareFuzzExecutorOptions declares the options [FuzzExecutor] itself reads.
// Scenarios using it should call this from their own Options declaration so the
// options validate and appear in `list-scenarios`.
func DeclareFuzzExecutorOptions(o *OptionSet) {
	declareNexusEndpointOption(o)
	o.String("kitchen-sink-gen", "", "Name of, or absolute path to, the kitchen-sink-gen executable.")
}

func (k FuzzExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	if k.InitLoadInput != nil {
		in, err := k.InitLoadInput(ctx, info)
		if err != nil {
			return err
		}
		if in != nil {
			return k.runLoadInput(ctx, info, in)
		}
	}
	if k.InitInputs == nil {
		return fmt.Errorf("InitInputs must be specified")
	}

	if err := ensureFuzzNexusEndpoint(ctx, info); err != nil {
		return err
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
			return k.execute(ctx, run, &options)
		},
	}
	return ge.Run(ctx, info)
}

func (k FuzzExecutor) GetDefaultConfiguration() RunConfiguration {
	return k.DefaultConfiguration
}

// runLoadInput validates the LoadInput, including its read templates, before it creates the Nexus endpoint or starts
// any workflow.
func (k FuzzExecutor) runLoadInput(ctx context.Context, info ScenarioInfo, in *kitchensink.LoadInput) error {
	vis := k.visibilityClient
	if vis == nil {
		vis = info.Client
	}
	loadInput, err := newLoadInputRun(info, in, vis)
	if err != nil {
		return err
	}
	if err := ensureFuzzNexusEndpoint(ctx, info); err != nil {
		return err
	}
	if !info.Configuration.DoNotRegisterSearchAttributes {
		if err := info.RegisterSearchAttributes(ctx, loadInput.defs); err != nil {
			return err
		}
	}
	return loadInput.run(ctx, info, &GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			options, err := loadInput.options(run)
			if err != nil {
				return err
			}
			return k.execute(ctx, run, &options)
		},
	})
}

func (k FuzzExecutor) execute(ctx context.Context, run *Run, options *KitchenSinkWorkflowOptions) error {
	if k.executeWorkflow != nil {
		return k.executeWorkflow(ctx, run, options)
	}
	return run.ExecuteKitchenSinkWorkflow(ctx, options)
}

// ensureFuzzNexusEndpoint creates the run's Nexus endpoint unless the user named one. The generated actions drive
// Nexus operations against this endpoint, so it has to exist before they run, and InitInputs names the same one. An
// endpoint the user named is theirs to have created.
func ensureFuzzNexusEndpoint(ctx context.Context, info ScenarioInfo) error {
	if info.OptionString(NexusEndpointOption) == "" {
		if _, err := info.EnsureNexusEndpoint(ctx); err != nil {
			return fmt.Errorf("failed to create nexus endpoint: %w", err)
		}
	}
	info.Logger.Infof("Using nexus endpoint %q", info.NexusEndpointName())
	return nil
}
