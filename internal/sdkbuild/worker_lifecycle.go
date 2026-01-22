package sdkbuild

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
	"github.com/temporalio/features/sdkbuild"
	"github.com/temporalio/omes/metrics"
	"go.uber.org/zap"
)

// WorkerLifecycleServer manages a worker subprocess with HTTP lifecycle endpoints.
type WorkerLifecycleServer struct {
	Program sdkbuild.Program // Built program to run
	Args    []string         // Runtime args (--task-queue, etc.)
	Port    int
	Logger  *zap.SugaredLogger

	process    *os.Process
	pid        int
	server     *http.Server
	shutdownCh chan struct{}
}

// Serve spawns the worker and starts the HTTP server. Blocks until shutdown.
func (s *WorkerLifecycleServer) Serve(ctx context.Context) error {
	// Spawn worker IMMEDIATELY (command includes --task-queue, --server-address, etc.)
	if err := s.spawnWorker(); err != nil {
		return fmt.Errorf("failed to spawn worker: %w", err)
	}

	s.shutdownCh = make(chan struct{})

	// Set up HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/shutdown", s.handleShutdown)
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/info", s.handleInfo)

	addr := fmt.Sprintf(":%d", s.Port)
	s.server = &http.Server{Addr: addr, Handler: mux}

	s.Logger.Infof("Worker lifecycle server listening on %s (worker PID: %d)", addr, s.pid)

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

// URL returns the server's HTTP URL.
func (s *WorkerLifecycleServer) URL() string {
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

// Kill forcefully kills the worker subprocess (SIGKILL).
func (s *WorkerLifecycleServer) Kill() {
	if s.process != nil {
		s.process.Kill()
	}
}

func (s *WorkerLifecycleServer) spawnWorker() error {
	if s.Program == nil {
		return fmt.Errorf("Program is required")
	}

	cmd, err := s.Program.NewCommand(context.Background(), s.Args...)
	if err != nil {
		return fmt.Errorf("failed to create command from program: %w", err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	s.process = cmd.Process
	s.pid = cmd.Process.Pid
	s.Logger.Infof("Started worker process with PID %d", s.pid)
	return nil
}

func (s *WorkerLifecycleServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "shutting_down"})

	// Signal the Serve() loop to initiate shutdown
	close(s.shutdownCh)
}

func (s *WorkerLifecycleServer) shutdownWorker() {
	if s.process == nil {
		return
	}

	s.Logger.Infof("Sending SIGTERM to worker PID %d", s.pid)
	s.process.Signal(syscall.SIGTERM)

	// Wait up to 10s for process to exit
	done := make(chan struct{})
	go func() {
		s.process.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.Logger.Info("Worker process exited")
	case <-time.After(10 * time.Second):
		s.Logger.Warn("Worker process did not exit in 10s, killing")
		s.process.Kill()
	}
}

func (s *WorkerLifecycleServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
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

func (s *WorkerLifecycleServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"starter_version": "omes-remote-worker",
		"worker_pid":      s.pid,
	})
}
