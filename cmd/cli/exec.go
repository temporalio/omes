package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

Required: --project-dir specifies the path to the test project directory
containing your workflow code and project file (pyproject.toml, package.json).

Version is auto-detected from project files. Use --version to override
or specify a local SDK path (e.g., --version ../sdk-python).

Examples:
  omes exec --language python --project-dir . -- worker --task-queue my-queue
  omes exec --language ts --project-dir ./my-test -- client --port 8080
  omes exec --language python --project-dir . --version ../sdk-python -- worker -q q
  omes exec --language python --project-dir . --remote-worker 8081 -- worker --task-queue myqueue`,
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

			// Build runtime args with language-specific prefix
			runtimeArgs, err := progbuild.BuildRuntimeArgs(sdkOpts.Language, execOpts.ProjectDir)
			if err != nil {
				return err
			}
			runtimeArgs = append(runtimeArgs, args...)

			// Remote worker mode: spawn worker immediately and start HTTP server
			if execOpts.RemoteWorkerPort > 0 {
				// Set up signal handling for graceful shutdown
				ctx, cancel := context.WithCancel(cmd.Context())
				defer cancel()

				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
				go func() {
					select {
					case sig := <-sigCh:
						sugar.Infof("Received signal %v, initiating shutdown", sig)
						cancel()
					case <-ctx.Done():
					}
				}()

				server := &progbuild.WorkerLifecycleServer{
					Program: prog,
					Args:    runtimeArgs,
					Port:    execOpts.RemoteWorkerPort,
					Logger:  sugar,
				}
				return server.Serve(ctx)
			}

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
	cmd.MarkFlagRequired("project-dir")

	return cmd
}
