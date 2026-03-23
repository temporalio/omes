package helloworld

import (
	harness "github.com/temporalio/omes/workers/go/projects/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func workerMain(c client.Client, config harness.WorkerConfig) error {
	w := worker.New(c, config.TaskQueue, worker.Options{})
	w.RegisterWorkflow(Workflow)
	return w.Run(worker.InterruptCh())
}
