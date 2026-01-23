package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/internal/progbuild"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var sdkOpts clioptions.SdkOptions
	var execOpts clioptions.ExecOptions

	cmd := &cobra.Command{
		Use:   "exec --language <lang> [--version <ver>] -- <program-args>",
		Short: "Build SDK and run program",
		Long: `Build SDK and run program with the given arguments.

Version is auto-detected from project files (pyproject.toml, package.json).
Use --version to override or specify a local SDK path.

Example:
  omes exec --language python -- worker --task-queue my-queue
  omes exec --language ts --version 1.11.0 -- client --port 8080`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()

			// Auto-detect version if not provided
			version := sdkOpts.Version
			if version == "" {
				var err error
				version, err = progbuild.DetectSDKVersion(cmd.Context(), sdkOpts.Language.String(), execOpts.ProjectDir)
				if err != nil {
					return fmt.Errorf("failed to detect SDK version (use --version to specify): %w", err)
				}
				sugar.Infof("Auto-detected SDK version: %s", version)
			}

			// Build the program
			builder := &progbuild.ProgramBuilder{
				Language:   sdkOpts.Language.String(),
				SDKVersion: version,
				ProjectDir: execOpts.ProjectDir,
				BuildDir:   execOpts.BuildDir,
				Logger:     sugar,
			}

			prog, err := builder.BuildProgram(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			// Build runtime args: for Python, prepend module name derived from project dir
			// For TypeScript, prepend path to compiled entry point
			var runtimeArgs []string
			absProjectDir, err := filepath.Abs(execOpts.ProjectDir)
			if err != nil {
				return fmt.Errorf("failed to resolve project directory: %w", err)
			}
			projectName := filepath.Base(absProjectDir)
			switch sdkOpts.Language {
			case clioptions.LangPython:
				moduleName := strings.ReplaceAll(projectName, "-", "_")
				runtimeArgs = []string{moduleName}
			case clioptions.LangTypeScript:
				runtimeArgs = []string{fmt.Sprintf("tslib/tests/%s/main.js", projectName)}
			}
			runtimeArgs = append(runtimeArgs, args...)

			// Run the program
			execCmd, err := prog.NewCommand(cmd.Context(), runtimeArgs...)
			if err != nil {
				return fmt.Errorf("failed to create command: %w", err)
			}

			sugar.Infof("Running: %v", execCmd.Args)
			return execCmd.Run()
		},
	}

	// Add flag sets
	sdkOpts.AddCLIFlags(cmd.Flags())
	cmd.Flags().AddFlagSet(execOpts.FlagSet())

	cmd.MarkFlagRequired("language")

	return cmd
}
