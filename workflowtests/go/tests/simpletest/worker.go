package simpletest

import (
	"github.com/temporalio/omes-workflow-tests/workflowtests/go/starter"
	"go.temporal.io/sdk/worker"
)

func workerMain(config *starter.WorkerConfig) error {
	w := worker.New(config.Client, config.TaskQueue, worker.Options{})
	w.RegisterWorkflow(Workflow)
	return w.Run(worker.InterruptCh())
}
