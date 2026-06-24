package singlenexusopworkflow

import (
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	workerks "github.com/temporalio/omes/workers/go/workerlib/kitchensink"
	"go.temporal.io/sdk/workflow"
)

const (
	payloadSyncOperation  = "payload-sync"
	payloadAsyncOperation = "payload-async"
)

func SingleNexusOpWorkflow(ctx workflow.Context, endpoint string, input []byte, outputSize int32) ([]byte, error) {
	return executePayloadNexusOperation(ctx, endpoint, payloadSyncOperation, input, outputSize)
}

func SingleAsyncNexusOpWorkflow(ctx workflow.Context, endpoint string, input []byte, outputSize int32) ([]byte, error) {
	return executePayloadNexusOperation(ctx, endpoint, payloadAsyncOperation, input, outputSize)
}

func executePayloadNexusOperation(ctx workflow.Context, endpoint string, operation string, input []byte, outputSize int32) ([]byte, error) {
	nc := workflow.NewNexusClient(endpoint, workerks.KitchenSinkServiceName)
	fut := nc.ExecuteOperation(ctx, operation, &kitchensink.PayloadNexusInput{
		Input:         input,
		BytesToReturn: outputSize,
	}, workflow.NexusOperationOptions{
		ScheduleToCloseTimeout: 120 * time.Second,
	})
	var output []byte
	if err := fut.Get(ctx, &output); err != nil {
		return nil, err
	}
	return output, nil
}
