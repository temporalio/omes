package starter

import (
	"context"
	"crypto/tls"

	"go.temporal.io/sdk/client"
)

// ClientConfig is passed to client's execute function.
type ClientConfig struct {
	// ConnectionOptions are starter-provided defaults for client.Dial.
	ConnectionOptions client.Options
	// TaskQueue for workflows
	TaskQueue string
	// RunID is a unique ID for this load test run (from /execute request)
	RunID string
	// Iteration is the current iteration number (from /execute request)
	Iteration int
}

// WorkerConfig is passed to worker's configure function.
type WorkerConfig struct {
	// ConnectionOptions are starter-provided defaults for client.Dial.
	ConnectionOptions client.Options
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

type staticHeadersProvider map[string]string

func (s staticHeadersProvider) GetHeaders(context.Context) (map[string]string, error) {
	return s, nil
}

func buildClientOptions(
	serverAddress string,
	namespace string,
	enableTLS bool,
	tlsServerName string,
	authHeader string,
) client.Options {
	opts := client.Options{
		HostPort:  serverAddress,
		Namespace: namespace,
	}
	if enableTLS || tlsServerName != "" {
		opts.ConnectionOptions.TLS = &tls.Config{
			MinVersion: tls.VersionTLS13,
			ServerName: tlsServerName,
		}
	}
	if authHeader != "" {
		opts.HeadersProvider = staticHeadersProvider{
			"Authorization": authHeader,
		}
	}
	return opts
}
