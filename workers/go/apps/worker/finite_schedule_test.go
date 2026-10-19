package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers/go/harness"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
)

type finiteScheduleResponseLoss struct {
	interceptor.ClientInterceptorBase
	lose  atomic.Bool
	calls atomic.Int32
}

func (loss *finiteScheduleResponseLoss) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	return &finiteScheduleResponseLossOutbound{ClientOutboundInterceptorBase: interceptor.ClientOutboundInterceptorBase{Next: next}, loss: loss}
}

type finiteScheduleResponseLossOutbound struct {
	interceptor.ClientOutboundInterceptorBase
	loss *finiteScheduleResponseLoss
}

func (out *finiteScheduleResponseLossOutbound) CreateSchedule(ctx context.Context, input *interceptor.ScheduleClientCreateInput) (client.ScheduleHandle, error) {
	out.loss.calls.Add(1)
	handle, err := out.Next.CreateSchedule(ctx, input)
	if err == nil && out.loss.lose.CompareAndSwap(true, false) {
		return nil, serviceerror.NewUnavailable("injected loss after finite schedule creation")
	}
	return handle, err
}

func TestFiniteScheduleProductionActivity(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	control, err := client.Dial(client.Options{HostPort: address})
	require.NoError(t, err)
	defer control.Close()
	loss := &finiteScheduleResponseLoss{}
	var describes atomic.Int32
	c, err := client.Dial(client.Options{
		HostPort: address, DataConverter: clioptions.OmesDataConverter(),
		Interceptors: []interceptor.ClientInterceptor{loss},
		ConnectionOptions: client.ConnectionOptions{DialOptions: []grpc.DialOption{
			grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				if _, ok := req.(*workflowservice.DescribeScheduleRequest); ok {
					describes.Add(1)
				}
				return invoke(ctx, method, req, reply, cc, opts...)
			}),
		}},
	})
	require.NoError(t, err)
	defer c.Close()
	queue := "finite-schedule-" + uuid.NewString()
	w := buildWorker(c, harness.WorkerContext{TaskQueue: queue})
	require.NoError(t, w.Start())
	defer w.Stop()
	ctx, cancel := context.WithTimeout(t.Context(), 85*time.Second)
	defer cancel()
	runActivity := func(id string) error {
		args, err := clioptions.OmesDataConverter().ToPayloads(ks.FiniteScheduleInput{ScheduleID: id})
		require.NoError(t, err)
		action := ks.RemoteActivityWithRetry(10*time.Second, 2, time.Second, 1)(&ks.ExecuteActivityAction{
			ActivityType: &ks.ExecuteActivityAction_Generic{
				Generic: &ks.ExecuteActivityAction_GenericActivity{Type: ks.CreateFiniteScheduleActivityName, Arguments: args.Payloads},
			},
		})
		run, err := control.ExecuteWorkflow(ctx, client.StartWorkflowOptions{ID: uuid.NewString(), TaskQueue: queue, WorkflowRunTimeout: 20 * time.Second},
			"kitchenSink", &ks.WorkflowInput{InitialActions: ks.ListActionSet(action, ks.NewEmptyReturnResultAction())})
		require.NoError(t, err)
		return run.Get(ctx, nil)
	}
	ids := []string{"omes-finite-" + uuid.NewString(), "omes-finite-" + uuid.NewString()}
	for _, id := range ids {
		handle := control.ScheduleClient().GetHandle(ctx, id)
		defer func() {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cleanupCancel()
			if err := handle.Delete(cleanupCtx); err != nil {
				var notFound *serviceerror.NotFound
				require.ErrorAs(t, err, &notFound)
			}
			require.EventuallyWithT(t, func(collect *assert.CollectT) {
				_, err := handle.Describe(cleanupCtx)
				var notFound *serviceerror.NotFound
				require.ErrorAs(collect, err, &notFound)
			}, 4*time.Second, 100*time.Millisecond)
		}()
	}
	require.NoError(t, runActivity(ids[0]))
	require.Equal(t, int32(1), loss.calls.Load())
	require.Zero(t, describes.Load(), "normal creation must not add schedule reads")
	loss.lose.Store(true)
	require.NoError(t, runActivity(ids[1]))
	require.Equal(t, int32(3), loss.calls.Load(), "activity retry must reuse the same finite schedule")
	require.Equal(t, int32(1), describes.Load(), "only the duplicate attempt reconciles")
	observationDeadline := time.Now().Add(65 * time.Second)
	for _, id := range ids {
		handle := control.ScheduleClient().GetHandle(ctx, id)
		var description *client.ScheduleDescription
		remaining := time.Until(observationDeadline)
		require.Positive(t, remaining, "shared schedule observation deadline exhausted")
		require.EventuallyWithT(t, func(collect *assert.CollectT) {
			var err error
			description, err = handle.Describe(ctx)
			require.NoError(collect, err)
			if err == nil {
				require.Zero(collect, description.Schedule.State.RemainingActions)
				require.Equal(collect, 4, description.Info.NumActions)
			}
		}, remaining, 250*time.Millisecond)
		require.Zero(t, description.Info.NumActionsSkippedOverlap)
		require.Zero(t, description.Info.NumActionsMissedCatchupWindow)
		require.Len(t, description.Info.RecentActions, 4)
		runs := make(map[string]bool)
		for _, action := range description.Info.RecentActions {
			require.NotNil(t, action.StartWorkflowResult)
			require.NotEmpty(t, action.StartWorkflowResult.FirstExecutionRunID)
			require.False(t, runs[action.StartWorkflowResult.FirstExecutionRunID])
			runs[action.StartWorkflowResult.FirstExecutionRunID] = true
			var result string
			require.NoError(t, control.GetWorkflow(ctx, action.StartWorkflowResult.WorkflowID, action.StartWorkflowResult.FirstExecutionRunID).Get(ctx, &result))
			require.Equal(t, "scheduled", result)
		}
	}
	handle := control.ScheduleClient().GetHandle(ctx, ids[0])
	require.NoError(t, handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			input.Description.Schedule.Action.(*client.ScheduleWorkflowAction).TaskQueue = "altered-after-settlement"
			return &client.ScheduleUpdate{Schedule: &input.Description.Schedule}, nil
		},
	}))
	before := loss.calls.Load()
	err = runActivity(ids[0])
	var applicationError *temporal.ApplicationError
	require.ErrorAs(t, err, &applicationError)
	require.Equal(t, "FiniteScheduleCollision", applicationError.Type())
	require.True(t, applicationError.NonRetryable())
	require.Equal(t, before+1, loss.calls.Load(), "configuration collision must not retry")
}
