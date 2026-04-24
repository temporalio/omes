package clioptions

import (
	"github.com/spf13/pflag"
)

// WorkerOptions for setting up worker parameters
type WorkerOptions struct {
	BuildID                      string
	DeploymentName               string
	DeploymentBuildID            string
	DefaultVersioningBehavior    string
	MaxConcurrentActivityPollers int
	MaxConcurrentWorkflowPollers int
	ActivityPollerAutoscaleMax   int // overrides MaxConcurrentActivityPollers
	WorkflowPollerAutoscaleMax   int // overrides MaxConcurrentWorkflowPollers
	MaxConcurrentActivities      int
	MaxConcurrentWorkflowTasks   int
	WorkerActivitiesPerSecond    float64
	ErrOnUnimplemented           bool

	fs *pflag.FlagSet
}

// FlagSet adds the relevant flags to populate the options struct and returns a pflag.FlagSet.
func (m *WorkerOptions) FlagSet() *pflag.FlagSet {
	if m.fs != nil {
		return m.fs
	}
	m.fs = pflag.NewFlagSet("worker_options", pflag.ExitOnError)
	m.fs.StringVar(&m.BuildID, "build-id", "", "Build ID for legacy Build-ID-based worker versioning. Mutually exclusive with --deployment-name")
	m.fs.StringVar(&m.DeploymentName, "deployment-name", "", "Worker Deployment name. When set, enables Worker Deployment Versioning and must be combined with --deployment-build-id")
	m.fs.StringVar(&m.DeploymentBuildID, "deployment-build-id", "", "Build ID within the Worker Deployment. Required when --deployment-name is set")
	m.fs.StringVar(&m.DefaultVersioningBehavior, "default-versioning-behavior", "", "Default versioning behavior for workflows that don't set one at registration. One of: pinned, auto-upgrade. Defaults to auto-upgrade when --deployment-name is set")
	m.fs.IntVar(&m.MaxConcurrentActivityPollers, "worker-max-concurrent-activity-pollers", 0, "Max concurrent activity pollers")
	m.fs.IntVar(&m.MaxConcurrentWorkflowPollers, "worker-max-concurrent-workflow-pollers", 0, "Max concurrent workflow pollers")
	m.fs.IntVar(&m.MaxConcurrentActivities, "worker-max-concurrent-activities", 0, "Max concurrent activities")
	m.fs.IntVar(&m.MaxConcurrentWorkflowTasks, "worker-max-concurrent-workflow-tasks", 0, "Max concurrent workflow tasks")
	m.fs.IntVar(&m.ActivityPollerAutoscaleMax, "worker-activity-poller-autoscale-max", 0, "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers")
	m.fs.IntVar(&m.WorkflowPollerAutoscaleMax, "worker-workflow-poller-autoscale-max", 0, "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers")
	m.fs.Float64Var(&m.WorkerActivitiesPerSecond, "worker-activities-per-second", 0, "Per-worker activity rate limit")
	m.fs.BoolVar(&m.ErrOnUnimplemented, "worker-err-on-unimplemented", false, "Fail on unimplemented actions (currently this only applies to concurrent client actions)")
	return m.fs
}
