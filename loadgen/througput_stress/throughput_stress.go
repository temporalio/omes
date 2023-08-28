package througput_stress

// WorkflowParams is the single input for the throughput stress workflow.
type WorkflowParams struct {
	// Number of times we should loop through the steps in the workflow.
	Iterations int
	// What iteration to start on. If we have continued-as-new, we might be starting at a nonzero
	// number.
	InitialIteration int
	// If nonzero, we will continue as new after history has grown to be at least this many events.
	ContinueAsNewAfterEventNumber int
}
