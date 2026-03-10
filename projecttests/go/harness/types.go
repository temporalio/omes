package harness

import (
	"context"

	"go.temporal.io/sdk/client"
)

// ExecuteFunc is called once per Execute RPC (one per iteration).
type ExecuteFunc func(context.Context, *Config) error

// InitFunc is called once during the Init RPC.
type InitFunc func(*InitConfig) error

// Config is the per-iteration data passed to ExecuteFunc.
type Config struct {
	ConnectionOptions client.Options
	TaskQueue         string
	RunID             string
	ExecutionID       string
	Iteration         int64
	Payload           []byte // per-iteration data from executor
}

// InitConfig is the data passed to InitFunc during the Init RPC.
type InitConfig struct {
	ConnectionOptions client.Options
	TaskQueue         string
	RunID             string
	ExecutionID       string
	ProjectConfig     []byte // from --config file
}

// WorkerConfig is passed to given worker function.
type WorkerConfig struct {
	// ConnectionOptions are starter-provided defaults for client.Dial.
	ConnectionOptions client.Options
	// TaskQueue for the worker
	TaskQueue string
	// PromListenAddress is the optional Prometheus metrics endpoint address
	PromListenAddress string
}

// WorkerFunc is the function type for running the worker.
type WorkerFunc func(*WorkerConfig) error
