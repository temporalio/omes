package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var language string
	var sdkVersion string
	var projectDir string
	var entry string
	var mode string
	var buildDir string
	var remoteWorkerPort int

	// Runtime args passed to the entry point
	var port int
	var taskQueue string
	var serverAddress string
	var namespace string

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
  omes exec --language python --sdk-version 1.21.0 \
    --project-dir ./my-test --entry main.py --mode client \
    --port 8080 --task-queue my-queue

  # Run worker
  omes exec --language python --sdk-version 1.21.0 \
    --project-dir ./my-test --entry main.py --mode worker \
    --task-queue my-queue --server-address localhost:7233`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()

			// Validate flags
			if mode == "" {
				return fmt.Errorf("--mode is required (client or worker)")
			}
			if mode != "client" && mode != "worker" {
				return fmt.Errorf("--mode must be 'client' or 'worker'")
			}

			// Set default entry file based on language
			if entry == "" {
				switch language {
				case "python":
					entry = "main.py"
				case "typescript":
					entry = "main.ts"
				default:
					return fmt.Errorf("--entry is required")
				}
			}

			// Build the program
			builder := &loadgen.ProgramBuilder{
				Language:   language,
				SDKVersion: sdkVersion,
				ProjectDir: projectDir,
				BuildDir:   buildDir,
				Logger:     sugar,
			}

			prog, err := builder.BuildProgram(cmd.Context(), entry)
			if err != nil {
				return fmt.Errorf("failed to build program: %w", err)
			}

			// Build runtime args - mode is first arg (subcommand)
			runtimeArgs := []string{mode}
			if port > 0 {
				runtimeArgs = append(runtimeArgs, "--port", fmt.Sprintf("%d", port))
			}
			if taskQueue != "" {
				runtimeArgs = append(runtimeArgs, "--task-queue", taskQueue)
			}
			if serverAddress != "" {
				runtimeArgs = append(runtimeArgs, "--server-address", serverAddress)
			}
			if namespace != "" {
				runtimeArgs = append(runtimeArgs, "--namespace", namespace)
			}

			// Remote worker mode: spawn worker and start HTTP lifecycle server
			if remoteWorkerPort > 0 && mode == "worker" {
				s := &loadgen.WorkerLifecycleServer{
					Program: prog,
					Args:    runtimeArgs,
					Port:    remoteWorkerPort,
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

	// SDK and project flags
	cmd.Flags().StringVar(&language, "language", "", "SDK language (python, typescript)")
	cmd.Flags().StringVar(&sdkVersion, "sdk-version", "", "SDK version or local path")
	cmd.Flags().StringVar(&projectDir, "project-dir", ".", "Path to user's test project")
	cmd.Flags().StringVar(&entry, "entry", "", "Path to entry file (e.g., main.py). Defaults: main.py (python), main.ts (typescript)")
	cmd.Flags().StringVar(&mode, "mode", "", "Mode to run: client or worker")
	cmd.Flags().StringVar(&buildDir, "build-dir", "", "Directory for SDK build output (cached)")
	cmd.Flags().IntVar(&remoteWorkerPort, "remote-worker", 0, "Run worker with HTTP lifecycle server on specified port")

	// Runtime args
	cmd.Flags().IntVar(&port, "port", 0, "HTTP port for client entry point")
	cmd.Flags().StringVar(&taskQueue, "task-queue", "", "Temporal task queue name")
	cmd.Flags().StringVar(&serverAddress, "server-address", "", "Temporal server address")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Temporal namespace")

	cmd.MarkFlagRequired("language")
	cmd.MarkFlagRequired("sdk-version")
	cmd.MarkFlagRequired("mode")

	return cmd
}
