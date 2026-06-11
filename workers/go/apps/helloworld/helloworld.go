package helloworld

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/workers/go/harness"
	sdkclient "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const workflowName = "HelloWorldWorkflow"

var App = harness.App{
	Worker:        buildWorker,
	ClientFactory: harness.DefaultClientFactory,
	Project: &harness.ProjectHandlers{
		Execute: executeProjectIteration,
	},
}

func buildWorker(client sdkclient.Client, context harness.WorkerContext) sdkworker.Worker {
	w := sdkworker.New(client, context.TaskQueue, context.WorkerOptions)
	w.RegisterWorkflowWithOptions(helloWorldWorkflow, workflow.RegisterOptions{Name: workflowName})
	return w
}

func helloWorldWorkflow(_ workflow.Context, name string) (string, error) {
	return fmt.Sprintf("Hello %s", name), nil
}

func executeProjectIteration(client sdkclient.Client, executeContext harness.ProjectExecuteContext) error {
	ctx := context.Background()
	run, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-%d", executeContext.Run.ExecutionID, executeContext.Iteration),
		TaskQueue: executeContext.TaskQueue,
	}, workflowName, "World")
	if err != nil {
		return err
	}
	var result string
	if err := run.Get(ctx, &result); err != nil {
		return err
	}
	fmt.Println(result)
	return nil
}
