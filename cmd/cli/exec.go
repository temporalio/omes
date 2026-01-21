package cli

import (
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
	"go.uber.org/zap"
)

func execCmd() *cobra.Command {
	var language string
	var sdkVersion string
	var buildDir string
	var useExisting bool
	var remoteWorkerPort int

	cmd := &cobra.Command{
		Use:   "exec [flags] -- <command...>",
		Short: "Build SDK and run command with the built SDK",
		Long: `Build the SDK (if needed) and run a command with the built SDK.

If no command is provided after --, just builds and exits (useful for Dockerfiles).
With --use-existing, skips build and fails if build doesn't exist.

With --remote-worker <port>, starts an HTTP server for lifecycle management:
  - Worker spawns immediately with the command after --
  - POST /shutdown - Graceful shutdown via SIGTERM
  - GET /metrics   - Process CPU/memory metrics (Prometheus format)
  - GET /info      - SDK metadata and worker PID

Example:
  omes exec --language python --sdk-version 1.9.0 --remote-worker 8081 -- \
    python worker.py --task-queue my-queue --server-address localhost:7233`,
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewDevelopment()
			sugar := logger.Sugar()

			runner := &loadgen.SDKRunner{
				Language:    language,
				SDKVersion:  sdkVersion,
				BuildDir:    buildDir,
				UseExisting: useExisting,
				Logger:      sugar,
			}

			resolvedBuildDir, err := runner.EnsureBuild(cmd.Context())
			if err != nil {
				return err
			}

			if len(args) == 0 {
				sugar.Infof("Build complete at %s", resolvedBuildDir)
				return nil
			}

			// Remote worker mode: spawn worker immediately and start HTTP server
			if remoteWorkerPort > 0 {
				s := &loadgen.WorkerLifecycleServer{
					Language:   language,
					SDKVersion: sdkVersion,
					BuildDir:   resolvedBuildDir,
					Command:    args,
					Port:       remoteWorkerPort,
					Logger:     sugar,
				}
				return s.Serve(cmd.Context())
			}

			// Normal mode: run command once and exit
			return loadgen.RunCommandWithOverride(cmd.Context(), sugar, language, resolvedBuildDir, sdkVersion, args)
		},
	}

	cmd.Flags().StringVar(&language, "language", "", "SDK language (python, typescript)")
	cmd.Flags().StringVar(&sdkVersion, "sdk-version", "", "SDK version or local path")
	cmd.Flags().StringVar(&buildDir, "build-dir", "", "Directory for SDK build output")
	cmd.Flags().BoolVar(&useExisting, "use-existing", false, "Use existing build, fail if not found")
	cmd.Flags().IntVar(&remoteWorkerPort, "remote-worker", 0, "Run as remote worker with HTTP server on specified port (0 = disabled)")

	cmd.MarkFlagRequired("language")

	return cmd
}
