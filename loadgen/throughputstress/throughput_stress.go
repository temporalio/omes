package throughputstress

import (
	"github.com/temporalio/omes/loadgen"
)

// WorkflowParams is the single input for the throughput stress workflow.
type WorkflowParams struct {
	// Number of times we should loop through the steps in the workflow.
	Iterations int `json:"iterations"`
	// If true, skip sleeps. This makes workflow end to end latency more informative.
	SkipSleep bool `json:"skipSleep"`
	// What iteration to start on. If we have continued-as-new, we might be starting at a nonzero number.
	InitialIteration int `json:"initialIteration"`
	// If nonzero, we will continue-as-new after specified iteration.
	ContinueAsNewAfterIterCount int `json:"continueAsNewAfterIterCount"`

	// Set internally and incremented every time the workflow spawns a child.
	ChildrenSpawned int `json:"childrenSpawned"`

	// If set, the workflow will run nexus tests.
	// The endpoint should be created ahead of time.
	NexusEndpoint string `json:"nexusEndpoint"`

	// SleepActivities defines the configuration for sleep activities.
	// If not set, no sleep activities will be run.
	SleepActivities *loadgen.SleepActivityConfig `json:"sleepActivities"`
}

type WorkflowOutput struct {
	// The total number of children that were spawned across all continued runs of the workflow.
	ChildrenSpawned int `json:"childrenSpawned"`
}
