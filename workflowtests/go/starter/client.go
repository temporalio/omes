package starter

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/client"
)

// RunClient starts an HTTP server for client lifecycle management.
// Called by Run() when the "client" subcommand is used.
//
// Provides endpoints:
//   - POST /execute: Run user's execute function for one iteration
//   - POST /shutdown: Graceful shutdown with request draining
//   - GET /info: SDK metadata
func RunClient(fn ExecuteFunc) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP port")
	taskQueue := fs.String("task-queue", "", "Task queue name (required)")
	serverAddress := fs.String("server-address", "localhost:7233", "Temporal server address")
	namespace := fs.String("namespace", "default", "Temporal namespace")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *taskQueue == "" {
		fmt.Fprintln(os.Stderr, "error: --task-queue is required")
		os.Exit(1)
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort:  *serverAddress,
		Namespace: *namespace,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	server := &clientServer{
		client:    c,
		taskQueue: *taskQueue,
		executeFn: fn,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", server.handleExecute)
	mux.HandleFunc("/shutdown", server.handleShutdown)
	mux.HandleFunc("/info", server.handleInfo)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}
	server.httpServer = httpServer

	fmt.Printf("Client starter listening on port %d\n", *port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

type clientServer struct {
	client       client.Client
	taskQueue    string
	executeFn    ExecuteFunc
	httpServer   *http.Server
	activeReqs   int64
	shuttingDown int32
	shutdownOnce sync.Once
}

type executeRequest struct {
	RunID     string `json:"run_id"`
	Iteration int    `json:"iteration"`
}

type executeResponse struct {
	Success   bool   `json:"success"`
	Error     string `json:"error,omitempty"`
	Traceback string `json:"traceback,omitempty"`
}

func (s *clientServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if atomic.LoadInt32(&s.shuttingDown) == 1 {
		writeJSON(w, executeResponse{
			Success: false,
			Error:   "Starter is shutting down",
		})
		return
	}

	atomic.AddInt64(&s.activeReqs, 1)
	defer atomic.AddInt64(&s.activeReqs, -1)

	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, executeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode request: %v", err),
		})
		return
	}

	config := &ClientConfig{
		Client:    s.client,
		TaskQueue: s.taskQueue,
		RunID:     req.RunID,
		Iteration: req.Iteration,
	}

	if err := s.executeFn(config); err != nil {
		writeJSON(w, executeResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	writeJSON(w, executeResponse{Success: true})
}

type shutdownRequest struct {
	DrainTimeoutMs int `json:"drain_timeout_ms"`
}

func (s *clientServer) handleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req shutdownRequest
	// Ignore decode errors - use default timeout
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.DrainTimeoutMs == 0 {
		req.DrainTimeoutMs = 30000
	}

	writeJSON(w, map[string]string{"status": "shutting_down"})

	go s.shutdownWithDrain(time.Duration(req.DrainTimeoutMs) * time.Millisecond)
}

func (s *clientServer) shutdownWithDrain(timeout time.Duration) {
	s.shutdownOnce.Do(func() {
		atomic.StoreInt32(&s.shuttingDown, 1)
		start := time.Now()

		// Wait for active requests to complete
		for atomic.LoadInt64(&s.activeReqs) > 0 && time.Since(start) < timeout {
			time.Sleep(100 * time.Millisecond)
		}

		if active := atomic.LoadInt64(&s.activeReqs); active > 0 {
			log.Printf("Forcing shutdown with %d active requests", active)
		}

		if s.httpServer != nil {
			_ = s.httpServer.Close()
		}
	})
}

func (s *clientServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Note: SDK version would need to be determined from module info
	// For now, we report it as "unknown" since Go doesn't expose it easily at runtime
	writeJSON(w, map[string]string{
		"sdk_language":    "go",
		"sdk_version":     "unknown",
		"starter_version": "0.1.0",
	})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}
