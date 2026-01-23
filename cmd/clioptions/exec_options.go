package clioptions

import (
	"github.com/spf13/pflag"
)

// ExecOptions for the exec command - building and running user entry points.
type ExecOptions struct {
	// Path to user's test project directory
	ProjectDir string
	// Mode to run: "client" or "worker"
	Mode string
	// Directory for SDK build output (cached)
	BuildDir string
	// Port for HTTP lifecycle server when running worker remotely
	RemoteWorkerPort int

	// Runtime args passed to the entry point
	Port      int    // HTTP port for client entry point
	TaskQueue string // Temporal task queue name

	fs *pflag.FlagSet
}

// FlagSet returns flags for exec options.
func (e *ExecOptions) FlagSet() *pflag.FlagSet {
	if e.fs != nil {
		return e.fs
	}
	e.fs = pflag.NewFlagSet("exec_options", pflag.ExitOnError)
	e.fs.StringVar(&e.ProjectDir, "project-dir", ".", "Path to user's test project")
	e.fs.StringVar(&e.Mode, "mode", "", "Mode to run: client or worker")
	e.fs.StringVar(&e.BuildDir, "build-dir", "", "Directory for SDK build output (cached)")
	e.fs.IntVar(&e.RemoteWorkerPort, "remote-worker", 0, "Run worker with HTTP lifecycle server on specified port")
	e.fs.IntVar(&e.Port, "port", 0, "HTTP port for client entry point")
	e.fs.StringVar(&e.TaskQueue, "task-queue", "", "Temporal task queue name")
	return e.fs
}
