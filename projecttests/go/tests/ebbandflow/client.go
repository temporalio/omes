package ebbandflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/temporalio/omes/projecttests/go/harness"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// executePayload is the per-iteration payload from the executor.
type executePayload struct {
	ActivityCount int64 `json:"activity_count"`
}

var pool = harness.NewClientPool()

func clientMain(ctx context.Context, config *harness.Config) error {
	c, err := pool.GetOrDial("default", config.ConnectionOptions)
	if err != nil {
		return err
	}

	params := buildWorkflowInput(config.Payload)

	wf, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:        fmt.Sprintf("w-%s-%s-%d", config.RunID, config.ExecutionID, config.Iteration),
		TaskQueue: config.TaskQueue,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(harness.OmesSearchAttributeKey).ValueSet(config.ExecutionID),
		),
	}, Workflow, params)
	if err != nil {
		return err
	}

	var result workflowOutput
	return wf.Get(ctx, &result)
}

// buildWorkflowInput creates the workflow input from the executor payload.
// If payload contains activity_count, generates that many copies of the activity
// template from the project config. Otherwise falls back to a single default activity.
func buildWorkflowInput(payload []byte) workflowInput {
	sleepNanos := cfg.SleepDurationNanos
	if sleepNanos == 0 {
		sleepNanos = 1_000_000 // 1ms fallback
	}

	activityCount := int64(1)
	if len(payload) > 0 {
		var ep executePayload
		if err := json.Unmarshal(payload, &ep); err == nil && ep.ActivityCount > 0 {
			activityCount = ep.ActivityCount
		}
	}

	activities := make([]workflowActivityInput, activityCount)
	for i := range activities {
		activities[i] = workflowActivityInput{SleepDurationNanos: sleepNanos}
	}
	return workflowInput{Activities: activities}
}
