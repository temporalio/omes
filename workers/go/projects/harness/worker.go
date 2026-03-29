package harness

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// startWorker parses CLI flags, creates a client via InitFunc, and runs the worker.
func (h *Harness) startWorker() {
	if h.workerFunc == nil {
		log.Fatalf("Attempted to start worker, but a worker was not registered")
	}

	fs := flag.NewFlagSet("worker", flag.ContinueOnError)
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

	_ = fs.Parse(os.Args[1:])

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

	// Set up Prometheus metrics if address provided
	if *promListenAddress != "" {
		registry := prometheus.NewRegistry()
		opts.MetricsHandler = newPromMetricsHandler(registry)

		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		listener, err := net.Listen("tcp", *promListenAddress)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", *promListenAddress, err)
		}
		server := &http.Server{Handler: mux}
		go func() {
			if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Printf("Metrics server error: %v", err)
			}
		}()
	}

	// Create client via ClientFunc (no project-specific init for worker)
	c, err := h.clientFunc(opts, ClientConfig{
		TaskQueue: *taskQueue,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	fmt.Printf("Worker starting on task queue: %s\n", *taskQueue)

	if err := h.workerFunc(c, WorkerConfig{
		TaskQueue:         *taskQueue,
		PromListenAddress: *promListenAddress,
	}); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
