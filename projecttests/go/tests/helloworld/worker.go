package helloworld

import (
	"github.com/temporalio/omes/projecttests/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func workerMain(config *harness.WorkerConfig) error {
	c, err := client.Dial(config.ConnectionOptions)
	if err != nil {
		return err
	}
	defer c.Close()

	w := worker.New(c, config.TaskQueue, worker.Options{})
	w.RegisterWorkflow(Workflow)
	return w.Run(worker.InterruptCh())
}
