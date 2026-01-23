package clioptions

import (
	"time"

	"github.com/spf13/pflag"
)

// WorkflowOptions for the workflow command - running workflow load tests.
type WorkflowOptions struct {
	// Local build flags
	ProjectDir string
	BuildDir   string

	// Remote mode flags (presence determines hybrid/remote mode)
	ClientURL string
	WorkerURL string

	// Load config
	Iterations          int
	Duration            time.Duration
	MaxConcurrent       int
	MaxIterationsPerSec float64
	Timeout             time.Duration
	RunID               string

	fs *pflag.FlagSet
}

// FlagSet returns flags for workflow options.
func (w *WorkflowOptions) FlagSet() *pflag.FlagSet {
	if w.fs != nil {
		return w.fs
	}
	w.fs = pflag.NewFlagSet("workflow_options", pflag.ExitOnError)

	// Local build flags
	w.fs.StringVar(&w.ProjectDir, "project-dir", "", "Path to test project directory (required when building locally)")
	w.fs.StringVar(&w.BuildDir, "build-dir", "", "Directory for SDK build output (cached)")

	// Remote mode flags (presence determines hybrid/remote mode)
	w.fs.StringVar(&w.ClientURL, "client-url", "", "URL of running client starter")
	w.fs.StringVar(&w.WorkerURL, "worker-url", "", "URL of running worker starter")

	// Load config flags
	w.fs.IntVar(&w.Iterations, "iterations", 0, "Number of iterations (cannot be used with --duration)")
	w.fs.DurationVar(&w.Duration, "duration", 0, "Test duration (cannot be used with --iterations)")
	w.fs.IntVar(&w.MaxConcurrent, "max-concurrent", 10, "Max concurrent iterations")
	w.fs.Float64Var(&w.MaxIterationsPerSec, "max-iterations-per-second", 0, "Max iterations per second rate limit")
	w.fs.DurationVar(&w.Timeout, "timeout", 0, "Test timeout (0 = no timeout)")
	w.fs.StringVar(&w.RunID, "run-id", "", "Run ID (auto-generated if not provided)")

	return w.fs
}
