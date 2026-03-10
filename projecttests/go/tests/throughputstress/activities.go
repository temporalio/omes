package throughputstress

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// NoopActivity does nothing. Used to exercise the local activity scheduling path.
func NoopActivity(ctx context.Context) error {
	return nil
}

// PayloadActivity accepts a payload and returns a payload of the same size.
func PayloadActivity(ctx context.Context, input []byte) ([]byte, error) {
	return input, nil
}

// RetryableErrorActivity fails on the first attempt, succeeds on retry.
func RetryableErrorActivity(ctx context.Context) error {
	info := activity.GetInfo(ctx)
	if info.Attempt < 2 {
		return temporal.NewApplicationError("intentional failure", "RETRY_ME", nil)
	}
	return nil
}

// TimeoutActivity sleeps longer than the timeout on the first attempt, succeeds on retry.
func TimeoutActivity(ctx context.Context) error {
	info := activity.GetInfo(ctx)
	if info.Attempt < 2 {
		time.Sleep(10 * time.Second)
	}
	return nil
}

// HeartbeatActivity sends heartbeats. On the first attempt it fails; on retry it succeeds.
func HeartbeatActivity(ctx context.Context) error {
	info := activity.GetInfo(ctx)
	if info.Attempt < 2 {
		return temporal.NewApplicationError("intentional heartbeat failure", "RETRY_ME", nil)
	}
	for i := 0; i < 3; i++ {
		activity.RecordHeartbeat(ctx, i)
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// ClientQueryActivity queries the running workflow.
func ClientQueryActivity(ctx context.Context, workflowID, runID string) error {
	c, err := pool.GetOrDial("worker", getWorkerClientOptions())
	if err != nil {
		return err
	}
	_, err = c.QueryWorkflow(ctx, workflowID, runID, "status")
	return err
}

// ClientSignalActivity sends a signal to the running workflow.
func ClientSignalActivity(ctx context.Context, workflowID, signalName string) error {
	c, err := pool.GetOrDial("worker", getWorkerClientOptions())
	if err != nil {
		return err
	}
	return c.SignalWorkflow(ctx, workflowID, "", signalName, nil)
}

// ClientUpdateActivity sends an update to the running workflow.
func ClientUpdateActivity(ctx context.Context, workflowID, updateName string, payload []byte) error {
	c, err := pool.GetOrDial("worker", getWorkerClientOptions())
	if err != nil {
		return err
	}
	args := []interface{}{}
	if payload != nil {
		args = append(args, payload)
	}
	handle, err := c.UpdateWorkflow(ctx, client.UpdateWorkflowOptions{
		WorkflowID:   workflowID,
		UpdateName:   updateName,
		Args:         args,
		WaitForStage: client.WorkflowUpdateStageCompleted,
	})
	if err != nil {
		return err
	}
	return handle.Get(ctx, nil)
}

// workerClientOptions is set during worker startup so activities can create clients.
var workerClientOptions *client.Options

func setWorkerClientOptions(opts client.Options) {
	workerClientOptions = &opts
}

func getWorkerClientOptions() client.Options {
	if workerClientOptions == nil {
		panic(fmt.Sprintf("worker client options not set; activities requiring client access must run on a worker"))
	}
	return *workerClientOptions
}
