package starter

import "go.temporal.io/sdk/client"

// ClientConfig is passed to client's execute function.
type ClientConfig struct {
	// Client is a pre-created Temporal client connection
	Client client.Client
	// TaskQueue for workflows
	TaskQueue string
	// RunID is a unique ID for this load test run (from /execute request)
	RunID string
	// Iteration is the current iteration number (from /execute request)
	Iteration int
}

// WorkerConfig is passed to worker's configure function.
type WorkerConfig struct {
	// Client is a pre-created Temporal client connection
	Client client.Client
	// TaskQueue for the worker
	TaskQueue string
	// PromListenAddress is the optional Prometheus metrics endpoint address
	PromListenAddress string
}

// ExecuteFunc is the function type for handling /execute requests.
// It is called for each workflow execution iteration.
type ExecuteFunc func(*ClientConfig) error

// ConfigureWorkerFunc is the function type for running the worker.
// It should create a worker, register workflows/activities, and call worker.Run().
// Returns when the worker exits.
type ConfigureWorkerFunc func(*WorkerConfig) error
