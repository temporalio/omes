package starter

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"go.temporal.io/sdk/client"
)

// RunClient starts an HTTP server for client lifecycle management.
// Called by Run() when the "client" subcommand is used.
//
// Provides endpoints:
//   - POST /execute: Run user's execute function for one iteration
//   - GET /info: readiness probe
func RunClient(fn ExecuteFunc) {
	fs := flag.NewFlagSet("client", flag.ExitOnError)
	port := fs.Int("port", 8080, "HTTP port")
	taskQueue := fs.String("task-queue", "", "Task queue name (required)")
	serverAddress := fs.String("server-address", "localhost:7233", "Temporal server address")
	namespace := fs.String("namespace", "default", "Temporal namespace")
	authHeader := fs.String("auth-header", "", "Authorization header value")
	enableTLS := fs.Bool("tls", false, "Enable TLS")
	tlsServerName := fs.String("tls-server-name", "", "TLS target server name")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *taskQueue == "" {
		fmt.Fprintln(os.Stderr, "error: --task-queue is required")
		os.Exit(1)
	}

	server := &clientServer{
		connectionOptions: buildClientOptions(
			*serverAddress,
			*namespace,
			*enableTLS,
			*tlsServerName,
			*authHeader,
		),
		taskQueue: *taskQueue,
		executeFn: fn,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/execute", server.handleExecute)
	mux.HandleFunc("/info", server.handleInfo)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
	}

	fmt.Printf("Client starter listening on port %d\n", *port)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

type clientServer struct {
	connectionOptions client.Options
	taskQueue         string
	executeFn         ExecuteFunc
}

type executeRequest struct {
	RunID     string `json:"run_id"`
	Iteration int    `json:"iteration"`
}

type executeResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (s *clientServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req executeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, executeResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to decode request: %v", err),
		})
		return
	}

	config := &ClientConfig{
		ConnectionOptions: s.connectionOptions,
		TaskQueue:         s.taskQueue,
		RunID:             req.RunID,
		Iteration:         req.Iteration,
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

func (s *clientServer) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeJSON(w, map[string]any{})
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("Failed to write JSON response: %v", err)
	}
}
