package worker

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers/go/harness"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

func genericArgumentProbe(_ context.Context, text string, number int32) (string, error) {
	return fmt.Sprintf("%s:%d", text, number), nil
}

func TestGenericActivityProductionDispatch(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	c, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer c.Close()
	queue := "generic-activity-" + uuid.NewString()
	w := buildWorker(c, harness.WorkerContext{TaskQueue: queue})
	w.RegisterActivityWithOptions(genericArgumentProbe, activity.RegisterOptions{Name: "generic_argument_probe"})
	require.NoError(t, w.Start())
	defer w.Stop()
	dc := converter.GetDefaultDataConverter()
	args, err := dc.ToPayloads("left", int32(23))
	require.NoError(t, err)
	action := ks.DefaultRemoteActivity(&ks.ExecuteActivityAction{
		ActivityType: &ks.ExecuteActivityAction_Generic{
			Generic: &ks.ExecuteActivityAction_GenericActivity{Type: "generic_argument_probe", Arguments: args.Payloads},
		},
	})
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	run, err := c.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: queue, TaskQueue: queue},
		"kitchenSink", &ks.WorkflowInput{InitialActions: []*ks.ActionSet{ks.SingleActionSet(action, ks.NewEmptyReturnResultAction())}})
	require.NoError(t, err)
	require.NoError(t, run.Get(ctx, nil))
	history := c.GetWorkflowHistory(ctx, run.GetID(), run.GetRunID(), false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	var scheduled, completed int
	for history.HasNext() {
		event, err := history.Next()
		require.NoError(t, err)
		if attributes := event.GetActivityTaskScheduledEventAttributes(); attributes != nil {
			scheduled++
			require.Equal(t, "generic_argument_probe", attributes.ActivityType.Name)
		}
		if attributes := event.GetActivityTaskCompletedEventAttributes(); attributes != nil {
			completed++
			var result string
			require.NoError(t, dc.FromPayloads(attributes.Result, &result))
			require.Equal(t, "left:23", result)
		}
	}
	require.Equal(t, 1, scheduled)
	require.Equal(t, 1, completed)
}
