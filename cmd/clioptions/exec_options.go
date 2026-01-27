package clioptions

import (
	"github.com/spf13/pflag"
)

type ProjectOptions struct {
	// Path to user's test project directory
	ProjectDir string
}

func (p *ProjectOptions) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&p.ProjectDir, "project-dir", "", "Path to test project directory (required)")
}

// ExecOptions for the exec command - building and running user entry points.
type ExecOptions struct {
	ProjectOptions
	// Remote worker mode: when > 0, wrap subprocess with HTTP lifecycle server on this port
	RemoteWorkerPort int
}

func (e *ExecOptions) AddFlags(fs *pflag.FlagSet) {
	e.ProjectOptions.AddFlags(fs)
	fs.IntVar(&e.RemoteWorkerPort, "remote-worker", 0, "Wrap subprocess with HTTP lifecycle server on this port")
}

type WorkflowOptionsNew struct {
	ProjectOptions
	// Remote mode flags (presence determines hybrid/remote mode)
	ClientURL string
	WorkerURL string
}

func (w *WorkflowOptionsNew) AddFlags(fs *pflag.FlagSet) {
	w.ProjectOptions.AddFlags(fs)
	// Remote mode flags (presence determines hybrid/remote mode)
	fs.StringVar(&w.ClientURL, "client-url", "", "URL of running client starter")
	fs.StringVar(&w.WorkerURL, "worker-url", "", "URL of running worker starter")
}
