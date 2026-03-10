package helloworld

import (
	"context"
	"log"

	"github.com/temporalio/omes/projecttests/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

var pool = harness.NewClientPool()

func clientMain(ctx context.Context, config *harness.Config) error {
	c, err := pool.GetOrDial("default", config.ConnectionOptions)
	if err != nil {
		return err
	}

	opts := client.StartWorkflowOptions{
		TaskQueue: config.TaskQueue,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(harness.OmesSearchAttributeKey).ValueSet(config.ExecutionID),
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
