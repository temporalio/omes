package throughputstress

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

	// Store client options so activities can create their own clients for self-queries/signals/updates
	setWorkerClientOptions(config.ConnectionOptions)

	w := worker.New(c, config.TaskQueue, worker.Options{})
	w.RegisterWorkflow(ThroughputStressWorkflow)
	w.RegisterWorkflow(ChildWorkflow)
	w.RegisterActivity(PayloadActivity)
	w.RegisterActivity(RetryableErrorActivity)
	w.RegisterActivity(TimeoutActivity)
	w.RegisterActivity(HeartbeatActivity)
	w.RegisterActivity(ClientQueryActivity)
	w.RegisterActivity(ClientSignalActivity)
	w.RegisterActivity(ClientUpdateActivity)
	return w.Run(worker.InterruptCh())
}
