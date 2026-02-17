package cli

import (
	"fmt"
	"os"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/programbuild"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var r execRunner

	cmd := &cobra.Command{
		Use:   "exec --language <lang> -- <program-args>",
		Short: "Build and run a program",
		Long: `Build a test project and exec into the resulting program.

Replaces the current process with the built program (syscall.Exec).

Examples:
  omes exec --language python --project-dir ./my-test -- worker --task-queue q
  omes exec --language python --project-dir ./my-test -- client --port 8080
  omes exec --language python --project-dir . --version ../sdk-python -- worker --task-queue q`,
		PreRun: func(cmd *cobra.Command, args []string) {
			r.preRun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			builder := &programbuild.ProgramBuilder{
				Language:   r.sdkOpts.Language.String(),
				ProjectDir: r.programOpts.ProgramDir,
				BuildDir:   r.programOpts.BuildDir,
				Logger:     r.logger,
			}

			prog, err := builder.BuildProgram(cmd.Context(), r.sdkOpts.Version)
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			runtimeArgs, err := programbuild.BuildRuntimeArgs(r.sdkOpts.Language, r.programOpts.ProgramDir)
			if err != nil {
				return err
			}
			runtimeArgs = append(runtimeArgs, args...)

			execCmd, err := prog.NewCommand(cmd.Context(), runtimeArgs...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			if execCmd.Dir != "" {
				if err := os.Chdir(execCmd.Dir); err != nil {
					return fmt.Errorf("failed to change directory: %w", err)
				}
			}

			env := execCmd.Env
			if env == nil {
				env = os.Environ()
			}

			r.logger.Infof("Exec: %v", execCmd.Args)
			return syscall.Exec(execCmd.Path, execCmd.Args, env)
		},
	}

	r.sdkOpts.AddCLIFlags(cmd.Flags())
	r.programOpts.AddFlags(cmd.Flags())
	cmd.Flags().AddFlagSet(r.loggingOpts.FlagSet())

	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("project-dir")

	return cmd
}

type execRunner struct {
	sdkOpts     clioptions.SdkOptions
	programOpts clioptions.ProgramOptions
	loggingOpts clioptions.LoggingOptions

	logger *zap.SugaredLogger
}

func (r *execRunner) preRun() {
	r.logger = r.loggingOpts.MustCreateLogger()
}
