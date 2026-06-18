package singlenexusopworkflow

import (
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	workerks "github.com/temporalio/omes/workers/go/workerlib/kitchensink"
	"go.temporal.io/sdk/workflow"
)

func SingleNexusOpWorkflow(ctx workflow.Context, endpoint string, input []byte, outputSize int32) ([]byte, error) {
	nc := workflow.NewNexusClient(endpoint, workerks.KitchenSinkServiceName)
	fut := nc.ExecuteOperation(ctx, "payload-sync", &kitchensink.PayloadNexusInput{
		Input:         input,
		BytesToReturn: outputSize,
	}, workflow.NexusOperationOptions{
		ScheduleToCloseTimeout: 30 * time.Second,
	})
	var output []byte
	if err := fut.Get(ctx, &output); err != nil {
		return nil, err
	}
	return output, nil
}
