package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/progbuild"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var r execRunner
	cmd := &cobra.Command{
		Use:   "exec --language <lang> [--version <ver>] -- <program-args>",
		Short: "Build SDK and run program",
		Long: `Build SDK and run program with the given arguments.

Project directory specifies the path to the test project directory
containing your workflow code and project file (pyproject.toml, package.json, etc.).

Version is auto-detected from project files. Use --version to override
or specify a local SDK path (e.g., --version ../sdk-python or --version 1.22.0).

Examples:
  omes exec --language python --project-dir . -- worker --task-queue my-queue
  omes exec --language ts --project-dir ./my-test -- client --port 8080
  omes exec --language python --project-dir . --version ../sdk-python -- worker -q q
  omes exec --language python --project-dir . --remote-worker 8081 -- worker --task-queue myqueue`,
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Auto-detect version if not provided
			version := r.sdkOpts.Version
			if version == "" {
				var err error
				version, err = progbuild.ResolveSDKVersion(cmd.Context(), r.sdkOpts.Language.String(), r.execOpts.ProjectDir, r.logger)
				if err != nil {
					return err
				}
			}

			// Build the program
			builder := &progbuild.ProjectBuilder{
				Language:   r.sdkOpts.Language.String(),
				SDKVersion: version,
				ProjectDir: r.execOpts.ProjectDir,
				Logger:     r.logger,
			}

			prog, err := builder.BuildProgram(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			// Build runtime args with language-specific prefix
			runtimeArgs, err := progbuild.BuildRuntimeArgs(r.sdkOpts.Language, r.execOpts.ProjectDir)
			if err != nil {
				return err
			}
			runtimeArgs = append(runtimeArgs, args...)

			// Remote worker mode: spawn worker immediately and start HTTP server
			if r.execOpts.RemoteWorkerPort > 0 {
				server := &progbuild.WorkerLifecycleServer{
					Program: prog,
					Args:    runtimeArgs,
					Port:    r.execOpts.RemoteWorkerPort,
					Logger:  r.logger,
				}
				return server.Serve(cmd.Context())
			}

			// Run the program
			execCmd, err := prog.NewCommand(cmd.Context(), runtimeArgs...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			r.logger.Infof("Running: %v", execCmd.Args)
			return execCmd.Run()
		},
	}

	// Add flag sets
	r.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("project-dir")

	return cmd
}

type execRunner struct {
	sdkOpts     clioptions.SdkOptions
	execOpts    clioptions.ExecOptions
	loggingOpts clioptions.LoggingOptions

	logger *zap.SugaredLogger
}

func (r *execRunner) addCLIFlags(fs *pflag.FlagSet) {
	r.sdkOpts.AddCLIFlags(fs)
	r.execOpts.AddFlags(fs)
	fs.AddFlagSet(r.loggingOpts.FlagSet())
}

func (r *execRunner) preRun() {
	r.logger = r.loggingOpts.MustCreateLogger()
}
