package simpletest

import (
	"context"
	"log"

	"github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"
	"go.temporal.io/sdk/client"
)

// Optional ClientPool usage:
// var pool = starter.NewClientPool()
//
// c, err := pool.GetOrDial("default", config.ConnectionOptions)
// if err != nil {
// 	return err
// }

func clientMain(config *starter.ClientConfig) error {
	c, err := client.Dial(config.ConnectionOptions)
	if err != nil {
		return err
	}
	defer c.Close()

	opts := client.StartWorkflowOptions{
		TaskQueue: config.TaskQueue,
	}

	wf, err := c.ExecuteWorkflow(context.Background(), opts, Workflow, "World")
	if err != nil {
		return err
	}

	var result string
	if err := wf.Get(context.Background(), &result); err != nil {
		return err
	}
	log.Printf("Workflow result: %s", result)
	return nil
}
