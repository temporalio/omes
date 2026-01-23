package starter

import (
	"flag"
	"fmt"
	"log"
	"os"

	"go.temporal.io/sdk/client"
)

// RunWorker creates a Temporal client and runs the user's worker configuration function.
// Called by Run() when the "worker" subcommand is used.
//
// The user's ConfigureWorkerFunc should:
//  1. Create a worker with worker.New(config.Client, config.TaskQueue, options)
//  2. Register workflows and activities
//  3. Call worker.Run(worker.InterruptCh()) - blocking until interrupted
func RunWorker(fn ConfigureWorkerFunc) {
	fs := flag.NewFlagSet("worker", flag.ExitOnError)
	taskQueue := fs.String("task-queue", "", "Task queue name (required)")
	serverAddress := fs.String("server-address", "localhost:7233", "Temporal server address")
	namespace := fs.String("namespace", "default", "Temporal namespace")
	promListenAddress := fs.String("prom-listen-address", "", "Prometheus metrics address")

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

	config := &WorkerConfig{
		Client:            c,
		TaskQueue:         *taskQueue,
		PromListenAddress: *promListenAddress,
	}

	fmt.Printf("Worker starting on task queue: %s\n", *taskQueue)
	if err := fn(config); err != nil {
		log.Fatalf("Worker error: %v", err)
	}
}
