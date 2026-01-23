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
	var clientOpts clioptions.ClientOptions
	var execOpts clioptions.ExecOptions

	cmd := &cobra.Command{
		Use:   "exec [flags]",
		Short: "Build SDK and run entry point",
		Long: `Build SDK and run a client or worker entry point.

Use --mode to select client or worker execution.
Use --remote-worker <port> to run worker with HTTP lifecycle endpoints.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()

			// Validate flags
			if execOpts.Mode == "" {
				return fmt.Errorf("--mode is required (client or worker)")
			}
			if execOpts.Mode != "client" && execOpts.Mode != "worker" {
				return fmt.Errorf("--mode must be 'client' or 'worker'")
			}

			// Build the program
			builder := &progbuild.ProgramBuilder{
				Language:   sdkOpts.Language.String(),
				SDKVersion: sdkOpts.Version,
				ProjectDir: execOpts.ProjectDir,
				BuildDir:   execOpts.BuildDir,
				Logger:     sugar,
			}

			prog, err := builder.BuildProgram(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			// Build runtime args: for Python, first arg is module name derived from project dir
			var runtimeArgs []string
			if sdkOpts.Language == clioptions.LangPython {
				// Derive module name: "simple-test" → "simple_test"
				moduleName := strings.ReplaceAll(filepath.Base(execOpts.ProjectDir), "-", "_")
				runtimeArgs = []string{moduleName}
			}
			runtimeArgs = append(runtimeArgs, execOpts.Mode)
			if execOpts.Port > 0 {
				runtimeArgs = append(runtimeArgs, "--port", fmt.Sprintf("%d", execOpts.Port))
			}
			if execOpts.TaskQueue != "" {
				runtimeArgs = append(runtimeArgs, "--task-queue", execOpts.TaskQueue)
			}
			if clientOpts.Address != "" {
				runtimeArgs = append(runtimeArgs, "--server-address", clientOpts.Address)
			}
			if clientOpts.Namespace != "" {
				runtimeArgs = append(runtimeArgs, "--namespace", clientOpts.Namespace)
			}

			// Remote worker mode: spawn worker and start HTTP lifecycle server
			if execOpts.RemoteWorkerPort > 0 && execOpts.Mode == "worker" {
				s := &progbuild.WorkerLifecycleServer{
					Program: prog,
					Args:    runtimeArgs,
					Port:    execOpts.RemoteWorkerPort,
					Logger:  sugar,
				}
				return s.Serve(cmd.Context())
			}

			// Normal mode: run the program and exit
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
	cmd.Flags().AddFlagSet(clientOpts.FlagSet())
	cmd.Flags().AddFlagSet(execOpts.FlagSet())

	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("version")
	cmd.MarkFlagRequired("mode")

	return cmd
}
