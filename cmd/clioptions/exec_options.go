package clioptions

import (
	"github.com/spf13/pflag"
)

// ExecOptions for the exec command - building and running user entry points.
type ExecOptions struct {
	// Path to user's test project directory
	ProjectDir string
	// Directory for SDK build output (cached)
	BuildDir string
	// Remote worker mode: when > 0, wrap subprocess with HTTP lifecycle server on this port
	RemoteWorkerPort int

	fs *pflag.FlagSet
}

// FlagSet returns flags for exec options.
func (e *ExecOptions) FlagSet() *pflag.FlagSet {
	if e.fs != nil {
		return e.fs
	}
	e.fs = pflag.NewFlagSet("exec_options", pflag.ExitOnError)
	e.fs.StringVar(&e.ProjectDir, "project-dir", "", "Path to test project directory (required)")
	e.fs.StringVar(&e.BuildDir, "build-dir", "", "Output directory for SDK build output")
	e.fs.IntVar(&e.RemoteWorkerPort, "remote-worker", 0, "Wrap subprocess with HTTP lifecycle server on this port")
	return e.fs
}
