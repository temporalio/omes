package worker

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/devserver"
	"github.com/temporalio/omes/internal/versions"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers/go/harness"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type holdFirstScheduledWorkflow struct {
	interceptor.WorkerInterceptorBase
	workflowID string
}

func (hold *holdFirstScheduledWorkflow) InterceptWorkflow(_ workflow.Context, next interceptor.WorkflowInboundInterceptor) interceptor.WorkflowInboundInterceptor {
	return &holdFirstScheduledWorkflowInbound{
		WorkflowInboundInterceptorBase: interceptor.WorkflowInboundInterceptorBase{Next: next},
		workflowID:                     hold.workflowID,
	}
}

type holdFirstScheduledWorkflowInbound struct {
	interceptor.WorkflowInboundInterceptorBase
	workflowID string
}

func (hold *holdFirstScheduledWorkflowInbound) ExecuteWorkflow(ctx workflow.Context, input *interceptor.ExecuteWorkflowInput) (any, error) {
	if workflow.GetInfo(ctx).WorkflowExecution.ID == hold.workflowID {
		workflow.GetSignalChannel(ctx, "release-disruption").Receive(ctx, nil)
	}
	return hold.Next.ExecuteWorkflow(ctx, input)
}

func TestFiniteScheduleOverlapRecoveryContract(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	ref, err := versions.Get("SERVER_VERSION")
	require.NoError(t, err)
	server, err := devserver.Start(t.Context(), devserver.Options{
		Ref: ref,
		DynamicConfigValues: map[string]any{
			"history.enableCHASMSchedulerCreation":         false,
			"history.chasmSchedulerCreationRolloutPercent": 0,
			"history.enableCHASMSchedulerMigration":        false,
		},
	})
	require.NoError(t, err)
	defer func() { require.NoError(t, server.Stop()) }()
	c, err := client.Dial(client.Options{HostPort: server.FrontendHostPort()})
	require.NoError(t, err)
	defer c.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 85*time.Second)
	defer cancel()
	queue := "finite-overlap-" + uuid.NewString()
	input := ks.FiniteScheduleInput{ScheduleID: "omes-finite-" + uuid.NewString()}
	options, err := ks.FiniteScheduleOptions(input, queue)
	require.NoError(t, err)
	handle, err := c.ScheduleClient().Create(ctx, options)
	require.NoError(t, err)
	scheduler, err := c.DescribeWorkflowExecution(ctx, "temporal-sys-scheduler:"+input.ScheduleID, "")
	require.NoError(t, err)
	require.Equal(t, "temporal-sys-scheduler-workflow", scheduler.WorkflowExecutionInfo.Type.Name)
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		require.NoError(t, ks.DeleteFiniteSchedule(cleanupCtx, c.ScheduleClient(), input, queue))
	}()
	var description *client.ScheduleDescription
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		description, err = handle.Describe(ctx)
		require.NoError(collect, err)
		if err == nil {
			require.Len(collect, description.Info.RecentActions, 1)
		}
	}, 5*time.Second, 100*time.Millisecond)
	firstID := description.Info.RecentActions[0].StartWorkflowResult.WorkflowID
	w := buildWorker(c, harness.WorkerContext{TaskQueue: queue, WorkerOptions: sdkworker.Options{
		Interceptors: []interceptor.WorkerInterceptor{&holdFirstScheduledWorkflow{workflowID: firstID}},
	}})
	require.NoError(t, w.Start())
	defer w.Stop()
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		description, err = handle.Describe(ctx)
		require.NoError(collect, err)
		if err == nil {
			require.Positive(collect, description.Info.NumActionsSkippedOverlap)
			require.Equal(collect, 1, description.Info.NumActions)
			require.EqualValues(collect, 3, description.Schedule.State.RemainingActions)
		}
	}, 25*time.Second, 100*time.Millisecond)
	require.NoError(t, c.SignalWorkflow(ctx, firstID, "", "release-disruption", nil))
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		description, err = handle.Describe(ctx)
		require.NoError(collect, err)
		if err == nil {
			require.Zero(collect, description.Schedule.State.RemainingActions)
			require.Equal(collect, 4, description.Info.NumActions)
		}
	}, 60*time.Second, 100*time.Millisecond)
	require.Positive(t, description.Info.NumActionsSkippedOverlap)
	require.Zero(t, description.Info.NumActionsMissedCatchupWindow)
	require.Len(t, description.Info.RecentActions, 4)
	seen := make(map[string]bool)
	for _, action := range description.Info.RecentActions {
		execution := action.StartWorkflowResult
		require.NotNil(t, execution)
		require.NotEmpty(t, execution.FirstExecutionRunID)
		require.False(t, seen[execution.FirstExecutionRunID])
		seen[execution.FirstExecutionRunID] = true
		var result string
		require.NoError(t, c.GetWorkflow(ctx, execution.WorkflowID, execution.FirstExecutionRunID).Get(ctx, &result))
		require.Equal(t, "scheduled", result)
		closed, err := c.DescribeWorkflowExecution(ctx, execution.WorkflowID, execution.FirstExecutionRunID)
		require.NoError(t, err)
		require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, closed.WorkflowExecutionInfo.Status)
	}
	t.Logf("recovered: actions=%d, skipped-overlap=%d, remaining=%d, completed=%d", description.Info.NumActions, description.Info.NumActionsSkippedOverlap, description.Schedule.State.RemainingActions, len(seen))
	require.ErrorContains(t, ks.VerifyFiniteSchedule(ctx, c, input, queue), "extra, skipped or missed actions")
	require.NoError(t, ks.VerifyFiniteScheduleAfterDisruption(ctx, c, input, queue))
}
