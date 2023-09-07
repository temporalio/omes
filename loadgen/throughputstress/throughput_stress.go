package throughputstress

const ThroughputStressScenarioIdSearchAttribute = "ThroughputStressScenarioId"

// WorkflowParams is the single input for the throughput stress workflow.
type WorkflowParams struct {
	// Number of times we should loop through the steps in the workflow.
	Iterations int `json:"iterations"`
	// If nonzero, we will continue as new after history has grown to be at least this many events.
	ContinueAsNewAfterEventCount int `json:"continueAsNewAfterEventCount"`

	// Set internally. What iteration to start on. If we have continued-as-new, we might be starting
	// at a nonzero number.
	InitialIteration int `json:"initialIteration"`
	// Set internally and incremented every time the workflow continues as new.
	TimesContinued int `json:"timesContinued"`
	// Set internally and incremented every time the workflow spawns a child.
	ChildrenSpawned int `json:"childrenSpawned"`
}

type WorkflowOutput struct {
	// The total number of children that were spawned across all continued runs of the workflow.
	ChildrenSpawned int `json:"childrenSpawned"`
	// The total number of times the workflow continued as new.
	TimesContinued int `json:"timesContinued"`
}

type ExecutorWorkflowInput struct {
	// Number of iterations to run of the load workflow
	Iterations int `json:"iterations"`
	// Maximum number of instances of the load workflow to run concurrently
	MaxConcurrent int `json:"maxConcurrent"`
	// The RunID as from ScenarioInfo
	RunID string `json:"runId"`
	// Parameters for the actual load workflow
	LoadParams WorkflowParams `json:"loadParams"`
}
