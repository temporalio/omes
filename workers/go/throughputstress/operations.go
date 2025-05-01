package throughputstress

import (
	"context"
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporalnexus"
	"go.temporal.io/sdk/workflow"
)

var ThroughputStressServiceName = "throughput-stress"

var EchoSyncOperation = nexus.NewSyncOperation("echo-sync", func(ctx context.Context, s string, opts nexus.StartOperationOptions) (string, error) {
	return s, nil
})

func EchoWorkflow(ctx workflow.Context, s string) (string, error) {
	return s, nil
}

func WaitForCancelWorkflow(ctx workflow.Context, input nexus.NoValue) (nexus.NoValue, error) {
	return nil, workflow.Await(ctx, func() bool {
		return false
	})
}

var EchoAsyncOperation = temporalnexus.NewWorkflowRunOperation("echo-async", EchoWorkflow, func(ctx context.Context, s string, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
	return client.StartWorkflowOptions{
		ID: opts.RequestID,
	}, nil
})

var WaitForCancelOperation = temporalnexus.NewWorkflowRunOperation("wait-for-cancel", WaitForCancelWorkflow, func(ctx context.Context, _ nexus.NoValue, opts nexus.StartOperationOptions) (client.StartWorkflowOptions, error) {
	return client.StartWorkflowOptions{
		ID: opts.RequestID,
	}, nil
})

func MustCreateNexusService() *nexus.Service {
	service := nexus.NewService(ThroughputStressServiceName)
	err := service.Register(EchoSyncOperation, EchoAsyncOperation, WaitForCancelOperation)
	if err != nil {
		panic(fmt.Sprintf("failed to register operations: %v", err))
	}
	return service
}
