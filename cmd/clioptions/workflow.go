package clioptions

import (
	"github.com/spf13/pflag"
)

type WorkflowOptions struct {
	SpawnWorker bool
}

func (w *WorkflowOptions) AddCLIFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&w.SpawnWorker, "spawn-worker", false, "Spawn a local worker process (managed by the runner)")
}
