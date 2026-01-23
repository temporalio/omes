package simpletest

import (
	"context"

	"github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"
	"go.temporal.io/sdk/client"
)

func clientMain(config *starter.ClientConfig) error {
	opts := client.StartWorkflowOptions{
		TaskQueue: config.TaskQueue,
	}

	wf, err := config.Client.ExecuteWorkflow(context.Background(), opts, Workflow, "World")
	if err != nil {
		return err
	}

	return wf.Get(context.Background(), nil)
}
