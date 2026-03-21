package harness

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// TODO: encode expected CLI-args in proto schema (so that we have parity across workers)

// startWorker creates a Temporal client and runs the user's worker configuration function.
// Called by Run() when the "worker" subcommand is used.
func (h *Harness) startWorker() {
	if h.workerFunc == nil {
		log.Fatalf("Attempted to start worker, but a worker was not registered")
	}

	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	taskQueue := fs.String("task-queue", "", "Task queue name (required)")
	serverAddress := fs.String("server-address", "localhost:7233", "Temporal server address")
	namespace := fs.String("namespace", "default", "Temporal namespace")
	authHeader := fs.String("auth-header", "", "Authorization header value")
	promListenAddress := fs.String("prom-listen-address", "", "Prometheus metrics address")
	enableTLS := fs.Bool("tls", false, "Enable TLS")
	tlsCertPath := fs.String("tls-cert-path", "", "Path to client TLS certificate")
	tlsKeyPath := fs.String("tls-key-path", "", "Path to client TLS private key")
	tlsServerName := fs.String("tls-server-name", "", "TLS target server name")
	disableHostVerification := fs.Bool("disable-tls-host-verification", false, "Disable TLS host verification")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *taskQueue == "" {
		fmt.Fprintln(os.Stderr, "error: --task-queue is required")
		os.Exit(1)
	}

	opts, err := buildClientOptions(
		*serverAddress,
		*namespace,
		*authHeader,
		tlsOptions{
			EnableTLS:               *enableTLS,
			CertPath:                *tlsCertPath,
			KeyPath:                 *tlsKeyPath,
			ServerName:              *tlsServerName,
			DisableHostVerification: *disableHostVerification,
		},
	)
	if err != nil {
		log.Fatalf("Failed to build client options: %v", err)
	}

	config := &WorkerConfig{
		ConnectionOptions: opts,
		TaskQueue:         *taskQueue,
		PromListenAddress: *promListenAddress,
	}

	metricsCleanup, err := setupPrometheusMetrics(config)
	if err != nil {
		log.Fatalf("Failed to setup metrics: %v", err)
	}
	defer metricsCleanup()

	fmt.Printf("Worker starting on task queue: %s\n", *taskQueue)

	if err := h.workerFunc(config); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
