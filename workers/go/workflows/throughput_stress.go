package workflows

import (
	"github.com/temporalio/omes/loadgen/througput_stress"
	"go.temporal.io/sdk/workflow"
)

// ThroughputStressWorkflow is meant to mimic the throughputstress scenario from bench-go of days
// past, but in a less-opaque way.
func ThroughputStressWorkflow(ctx workflow.Context, params *througput_stress.WorkflowParams) (string, error) {
	return "hi", nil
}
