package clioptions

import (
	"github.com/spf13/pflag"
)

// WorkerOptions for setting up worker parameters
type WorkerOptions struct {
	BuildID                      string
	MaxConcurrentActivityPollers int
	MaxConcurrentWorkflowPollers int
	ActivityPollerAutoscaleMax   int // overrides MaxConcurrentActivityPollers
	WorkflowPollerAutoscaleMax   int // overrides MaxConcurrentWorkflowPollers
	MaxConcurrentActivities      int
	MaxConcurrentWorkflowTasks   int
	WorkerActivitiesPerSecond    float64

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
	m.fs.IntVar(&m.ActivityPollerAutoscaleMax, prefix+"activity-poller-autoscale-max", 0, "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers")
	m.fs.IntVar(&m.WorkflowPollerAutoscaleMax, prefix+"workflow-poller-autoscale-max", 0, "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers")
	m.fs.Float64Var(&m.WorkerActivitiesPerSecond, prefix+"worker-activities-per-second", 0, "Per-worker activity rate limit")
	return m.fs
}
