package cmdoptions

import (
	"github.com/spf13/pflag"
	"strconv"
)

// WorkerOptions for setting up worker parameters
type WorkerOptions struct {
	MaxConcurrentActivityTaskPollers       int
	MaxConcurrentWorkflowTaskPollers       int
	MaxConcurrentActivityExecutionSize     int
	MaxConcurrentWorkflowTaskExecutionSize int
}

// AddCLIFlags adds the relevant flags to populate the options struct.
func (m *WorkerOptions) AddCLIFlags(fs *pflag.FlagSet, prefix string) {
	fs.IntVar(&m.MaxConcurrentActivityTaskPollers, prefix+"max-concurrent-activity-task-pollers", 0, "Max concurrent activity task pollers")
	fs.IntVar(&m.MaxConcurrentWorkflowTaskPollers, prefix+"max-concurrent-workflow-task-pollers", 0, "Max concurrent workflow task pollers")
	fs.IntVar(&m.MaxConcurrentActivityExecutionSize, prefix+"max-concurrent-activity-execution-size", 0, "Max concurrent activity execution size")
	fs.IntVar(&m.MaxConcurrentWorkflowTaskExecutionSize, prefix+"max-concurrent-workflow-task-execution-size", 0, "Max concurrent workflow task execution size")
}

// ToFlags converts these options to string flags.
func (m *WorkerOptions) ToFlags() (flags []string) {
	if m.MaxConcurrentActivityTaskPollers != 0 {
		flags = append(flags, "--max-concurrent-activity-task-pollers", strconv.Itoa(m.MaxConcurrentActivityTaskPollers))
	}
	if m.MaxConcurrentWorkflowTaskPollers != 0 {
		flags = append(flags, "--max-concurrent-workflow-task-pollers", strconv.Itoa(m.MaxConcurrentWorkflowTaskPollers))
	}
	if m.MaxConcurrentActivityExecutionSize != 0 {
		flags = append(flags, "--max-concurrent-activity-execution-size", strconv.Itoa(m.MaxConcurrentActivityExecutionSize))
	}
	if m.MaxConcurrentWorkflowTaskExecutionSize != 0 {
		flags = append(flags, "--max-concurrent-workflow-task-execution-size", strconv.Itoa(m.MaxConcurrentWorkflowTaskExecutionSize))
	}
	return
}
