package helloworld

import (
	"context"
	"log"

	harness "github.com/temporalio/omes/workers/go/projects/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

func clientMain(ctx context.Context, c client.Client, iter harness.ExecuteInfo) error {
	opts := client.StartWorkflowOptions{
		TaskQueue: iter.TaskQueue,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(harness.OmesSearchAttributeKey).ValueSet(iter.ExecutionID),
		),
	}

	wf, err := c.ExecuteWorkflow(ctx, opts, Workflow, "World")
	if err != nil {
		return err
	}

	var result string
	if err := wf.Get(ctx, &result); err != nil {
		return err
	}
	log.Printf("Workflow result: %s", result)
	return nil
}
