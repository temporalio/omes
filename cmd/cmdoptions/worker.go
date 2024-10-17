package cmdoptions

import (
	"strconv"

	"github.com/spf13/pflag"
)

// WorkerOptions for setting up worker parameters
type WorkerOptions struct {
	BuildID                      string
	MaxConcurrentActivityPollers int
	MaxConcurrentWorkflowPollers int
	MaxConcurrentActivities      int
	MaxConcurrentWorkflowTasks   int
}

// AddCLIFlags adds the relevant flags to populate the options struct.
func (m *WorkerOptions) AddCLIFlags(fs *pflag.FlagSet, prefix string) {
	fs.StringVar(&m.BuildID, prefix+"build-id", "", "Build ID")
	fs.IntVar(&m.MaxConcurrentActivityPollers, prefix+"max-concurrent-activity-pollers", 0, "Max concurrent activity pollers")
	fs.IntVar(&m.MaxConcurrentWorkflowPollers, prefix+"max-concurrent-workflow-pollers", 0, "Max concurrent workflow pollers")
	fs.IntVar(&m.MaxConcurrentActivities, prefix+"max-concurrent-activities", 0, "Max concurrent activities")
	fs.IntVar(&m.MaxConcurrentWorkflowTasks, prefix+"max-concurrent-workflow-tasks", 0, "Max concurrent workflow tasks")
}

// ToFlags converts these options to string flags.
func (m *WorkerOptions) ToFlags() (flags []string) {
	if m.BuildID != "" {
		flags = append(flags, "--build-id", m.BuildID)
	}
	if m.MaxConcurrentActivityPollers != 0 {
		flags = append(flags, "--max-concurrent-activity-pollers", strconv.Itoa(m.MaxConcurrentActivityPollers))
	}
	if m.MaxConcurrentWorkflowPollers != 0 {
		flags = append(flags, "--max-concurrent-workflow-pollers", strconv.Itoa(m.MaxConcurrentWorkflowPollers))
	}
	if m.MaxConcurrentActivities != 0 {
		flags = append(flags, "--max-concurrent-activities", strconv.Itoa(m.MaxConcurrentActivities))
	}
	if m.MaxConcurrentWorkflowTasks != 0 {
		flags = append(flags, "--max-concurrent-workflow-tasks", strconv.Itoa(m.MaxConcurrentWorkflowTasks))
	}
	return
}
