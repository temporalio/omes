package starter

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// RunWorker creates a Temporal client and runs the user's worker configuration function.
// Called by Run() when the "worker" subcommand is used.
//
// The user's ConfigureWorkerFunc should:
//  1. Create a client with client.Dial(config.ConnectionOptions)
//  2. Create a worker with worker.New(client, config.TaskQueue, options)
//  3. Register workflows and activities
//  4. Call worker.Run(worker.InterruptCh()) - blocking until interrupted
func RunWorker(fn ConfigureWorkerFunc) {
	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	taskQueue := fs.String("task-queue", "", "Task queue name (required)")
	serverAddress := fs.String("server-address", "localhost:7233", "Temporal server address")
	namespace := fs.String("namespace", "default", "Temporal namespace")
	authHeader := fs.String("auth-header", "", "Authorization header value")
	enableTLS := fs.Bool("tls", false, "Enable TLS")
	tlsServerName := fs.String("tls-server-name", "", "TLS target server name")
	promListenAddress := fs.String("prom-listen-address", "", "Prometheus metrics address")

	if err := fs.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Failed to parse flags: %v", err)
	}

	if *taskQueue == "" {
		fmt.Fprintln(os.Stderr, "error: --task-queue is required")
		os.Exit(1)
	}

	config := &WorkerConfig{
		ConnectionOptions: buildClientOptions(
			*serverAddress,
			*namespace,
			*enableTLS,
			*tlsServerName,
			*authHeader,
		),
		TaskQueue:         *taskQueue,
		PromListenAddress: *promListenAddress,
	}

	metricsCleanup, err := setupPrometheusMetrics(config)
	if err != nil {
		log.Fatalf("Failed to setup metrics: %v", err)
	}
	defer metricsCleanup()

	fmt.Printf("Worker starting on task queue: %s\n", *taskQueue)
	if err := fn(config); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
