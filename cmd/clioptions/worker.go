package clioptions

import (
	"github.com/spf13/pflag"
)

// WorkerOptions for setting up worker parameters
type WorkerOptions struct {
	BuildID                      string
	MaxConcurrentActivityPollers int
	MaxConcurrentWorkflowPollers int
	MaxConcurrentActivities      int
	MaxConcurrentWorkflowTasks   int

	fs         *pflag.FlagSet
	usedPrefix string
}

// FlagSet adds the relevant flags to populate the options struct and returns a pflag.FlagSet.
func (m *WorkerOptions) FlagSet(prefix string) *pflag.FlagSet {
	if m.fs != nil {
		if prefix != m.usedPrefix {
			panic("prefix mismatch")
		}
		return m.fs
	}
	m.usedPrefix = prefix
	m.fs = pflag.NewFlagSet("worker_options", pflag.ExitOnError)
	m.fs.StringVar(&m.BuildID, prefix+"build-id", "", "Build ID")
	m.fs.IntVar(&m.MaxConcurrentActivityPollers, prefix+"max-concurrent-activity-pollers", 0, "Max concurrent activity pollers")
	m.fs.IntVar(&m.MaxConcurrentWorkflowPollers, prefix+"max-concurrent-workflow-pollers", 0, "Max concurrent workflow pollers")
	m.fs.IntVar(&m.MaxConcurrentActivities, prefix+"max-concurrent-activities", 0, "Max concurrent activities")
	m.fs.IntVar(&m.MaxConcurrentWorkflowTasks, prefix+"max-concurrent-workflow-tasks", 0, "Max concurrent workflow tasks")
	return m.fs
}
