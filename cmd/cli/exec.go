package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/metrics"
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
				s := &remoteWorkerServer{
					language:   language,
					sdkVersion: sdkVersion,
					buildDir:   resolvedBuildDir,
					command:    args,
					port:       remoteWorkerPort,
					logger:     sugar,
				}
				return s.serve(cmd.Context())
			}

			// Normal mode: run command once and exit
			return loadgen.RunCommandWithOverride(cmd.Context(), sugar, language, resolvedBuildDir, args)
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

type remoteWorkerServer struct {
	language   string
	sdkVersion string
	buildDir   string
	command    []string
	port       int
	logger     *zap.SugaredLogger

	process    *os.Process
	pid        int
	server     *http.Server
	shutdownCh chan struct{}
}

func (s *remoteWorkerServer) serve(ctx context.Context) error {
	// Spawn worker IMMEDIATELY (command includes --task-queue, --server-address, etc.)
	if err := s.spawnWorker(); err != nil {
		return fmt.Errorf("failed to spawn worker: %w", err)
	}

	s.shutdownCh = make(chan struct{})

	// Set up HTTP server WITHOUT /init endpoint
	mux := http.NewServeMux()
	mux.HandleFunc("/shutdown", s.handleShutdown)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/info", s.handleInfo)

	addr := fmt.Sprintf(":%d", s.port)
	s.server = &http.Server{Addr: addr, Handler: mux}

	s.logger.Infof("Remote worker server listening on %s (worker PID: %d)", addr, s.pid)

	errCh := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.shutdownWorker()
		s.server.Shutdown(context.Background())
		return ctx.Err()
	case <-s.shutdownCh:
		s.shutdownWorker()
		s.server.Shutdown(context.Background())
		return nil
	}
}

func (s *remoteWorkerServer) spawnWorker() error {
	// Command args are complete from CLI (user passes --task-queue, --server-address, etc.)
	cmd, err := loadgen.BuildCommandWithOverride(s.language, s.buildDir, s.command[0], s.command[1:])
	if err != nil {
		return err
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	s.process = cmd.Process
	s.pid = cmd.Process.Pid
	s.logger.Infof("Started worker process with PID %d", s.pid)
	return nil
}

func (s *remoteWorkerServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})

	// Signal the serve() loop to initiate shutdown
	close(s.shutdownCh)
}

func (s *remoteWorkerServer) shutdownWorker() {
	if s.process == nil {
		return
	}

	s.logger.Infof("Sending SIGTERM to worker PID %d", s.pid)
	s.process.Signal(syscall.SIGTERM)

	// Wait up to 10s for process to exit
	done := make(chan struct{})
	go func() {
		s.process.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.Info("Worker process exited")
	case <-time.After(10 * time.Second):
		s.logger.Warn("Worker process did not exit in 10s, killing")
		s.process.Kill()
	}
}

func (s *remoteWorkerServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.pid == 0 {
		w.Write([]byte("# No process running\n"))
		return
	}

	collector, err := metrics.NewProcessCollector(s.pid)
	if err != nil {
		w.Write([]byte(fmt.Sprintf("# Error creating collector: %v\n", err)))
		return
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func (s *remoteWorkerServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(loadgen.InfoResponse{
		SDKLanguage:    s.language,
		SDKVersion:     s.sdkVersion,
		StarterVersion: "omes-remote-worker",
		WorkerPID:      s.pid,
	})
}
