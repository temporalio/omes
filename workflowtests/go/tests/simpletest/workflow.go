package simpletest

import "go.temporal.io/sdk/workflow"

// Workflow is a simple workflow that returns a greeting.
func Workflow(ctx workflow.Context, name string) (string, error) {
	return "Hello " + name, nil
}
