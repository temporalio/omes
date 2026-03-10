package throughputstress

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/projecttests/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

var (
	cfg  ThroughputStressConfig
	pool = harness.NewClientPool()
)

func clientMain(ctx context.Context, config *harness.Config) error {
	c, err := pool.GetOrDial("default", config.ConnectionOptions)
	if err != nil {
		return err
	}

	input := ThroughputStressInput{
		InternalIterations:     cfg.InternalIterations,
		ContinueAsNewAfterIter: cfg.ContinueAsNewAfterIter,
		SleepDuration:          parseSleepDuration(cfg.SleepDuration),
		IncludeRetryScenarios:  cfg.IncludeRetryScenarios,
		NexusEndpoint:          cfg.NexusEndpoint,
		PayloadSizeBytes:       cfg.PayloadSizeBytes,
	}

	wf, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        fmt.Sprintf("throughput-stress-%s-%d", config.ExecutionID, config.Iteration),
		TaskQueue: config.TaskQueue,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(harness.OmesSearchAttributeKey).ValueSet(config.ExecutionID),
		),
	}, ThroughputStressWorkflow, input)
	if err != nil {
		return err
	}

	return wf.Get(ctx, nil)
}
