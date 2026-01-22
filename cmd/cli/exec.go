package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var sdkOpts clioptions.SdkOptions
	var clientOpts clioptions.ClientOptions
	var execOpts clioptions.ExecOptions

	cmd := &cobra.Command{
		Use:   "exec [flags]",
		Short: "Build SDK and run entry point",
		Long: `Build the SDK using sdkbuild and run a client or worker entry point.

User code pattern:
  - User writes main.py/main.ts that calls: run(client=client_main, worker=worker_main)
  - The program is invoked with --mode to select client or worker

With --remote-worker <port>, starts an HTTP server for lifecycle management:
  - Worker spawns immediately
  - POST /shutdown - Graceful shutdown via SIGTERM
  - GET /metrics   - Process CPU/memory metrics (Prometheus format)
  - GET /info      - SDK metadata and worker PID

Example:
  # Run client
  omes exec --language python --version 1.21.0 \
    --project-dir ./my-test --entry main.py --mode client \
    --port 8080 --task-queue my-queue

  # Run worker
  omes exec --language python --version 1.21.0 \
    --project-dir ./my-test --entry main.py --mode worker \
    --task-queue my-queue --server-address localhost:7233`,
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

			// Set default entry file based on language
			if execOpts.Entry == "" {
				switch sdkOpts.Language {
				case clioptions.LangPython:
					execOpts.Entry = "main.py"
				case clioptions.LangTypeScript:
					execOpts.Entry = "main.ts"
				default:
					return fmt.Errorf("--entry is required")
				}
			}

			// Build the program
			builder := &loadgen.ProgramBuilder{
				Language:   sdkOpts.Language.String(),
				SDKVersion: sdkOpts.Version,
				ProjectDir: execOpts.ProjectDir,
				BuildDir:   execOpts.BuildDir,
				Logger:     sugar,
			}

			prog, err := builder.BuildProgram(cmd.Context(), execOpts.Entry)
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			// Build runtime args - mode is first arg (subcommand)
			runtimeArgs := []string{execOpts.Mode}
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
				s := &loadgen.WorkerLifecycleServer{
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
