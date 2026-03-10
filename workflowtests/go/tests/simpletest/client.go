package simpletest

import (
	"context"
	"log"

	"github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"
	"go.temporal.io/sdk/client"
)

var pool = starter.NewClientPool()

func clientMain(config *starter.ClientConfig) error {
	c, err := pool.GetOrDial("default", config.ConnectionOptions)
	if err != nil {
		return err
	}

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

// Alternative: create a fresh connection per iteration using the SDK directly.
// This properly cleans up via defer c.Close(), but the pool above avoids
// redundant connection setup under load.
//
// import "go.temporal.io/sdk/client"
//
// func clientMain(config *starter.ClientConfig) error {
// 	c, err := client.Dial(config.ConnectionOptions)
// 	if err != nil {
// 		return err
// 	}
// 	defer c.Close()
//
// 	opts := client.StartWorkflowOptions{
// 		TaskQueue: config.TaskQueue,
// 	}
//
// 	wf, err := c.ExecuteWorkflow(context.Background(), opts, Workflow, "World")
// 	if err != nil {
// 		return err
// 	}
//
// 	var result string
// 	if err := wf.Get(context.Background(), &result); err != nil {
// 		return err
// 	}
// 	log.Printf("Workflow result: %s", result)
// 	return nil
// }
