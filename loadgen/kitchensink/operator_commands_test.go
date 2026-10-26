package kitchensink

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	activitypb "go.temporal.io/api/activity/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
)

type operatorStartTestClient struct {
	client.Client
	service workflowservicepb.WorkflowServiceClient
	start   func(context.Context, client.StartActivityOptions) (client.ActivityHandle, error)
}

func (c *operatorStartTestClient) WorkflowService() workflowservicepb.WorkflowServiceClient {
	return c.service
}

func (c *operatorStartTestClient) ExecuteActivity(ctx context.Context, options client.StartActivityOptions, _ any, _ ...any) (client.ActivityHandle, error) {
	return c.start(ctx, options)
}

type operatorCleanupTestService struct {
	workflowservicepb.WorkflowServiceClient
	describe  func(context.Context, *workflowservicepb.DescribeActivityExecutionRequest) (*workflowservicepb.DescribeActivityExecutionResponse, error)
	terminate func(context.Context, *workflowservicepb.TerminateActivityExecutionRequest) (*workflowservicepb.TerminateActivityExecutionResponse, error)
}

func (s *operatorCleanupTestService) DescribeActivityExecution(ctx context.Context, request *workflowservicepb.DescribeActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	return s.describe(ctx, request)
}

func (s *operatorCleanupTestService) TerminateActivityExecution(ctx context.Context, request *workflowservicepb.TerminateActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.TerminateActivityExecutionResponse, error) {
	return s.terminate(ctx, request)
}

func TestStandaloneActivityOperatorCleanup(t *testing.T) {
	run := func(ctx context.Context, c client.Client) error {
		executor := ClientActionsExecutor{Client: c, Namespace: "default"}
		return executor.ExecuteClientSequence(ctx, ClientActions(&ClientAction{
			Variant: &ClientAction_DoStandaloneActivityOperatorCommands{
				DoStandaloneActivityOperatorCommands: &DoStandaloneActivityOperatorCommands{
					CommandType: DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
					Activity: &ExecuteActivityAction{
						TaskQueue: "operators", ActivityType: &ExecuteActivityAction_Noop{},
					},
				},
			},
		}))
	}

	t.Run("AlreadyCanceled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			var starts, describes int
			c := &operatorStartTestClient{
				start: func(ctx context.Context, _ client.StartActivityOptions) (client.ActivityHandle, error) {
					starts++
					return nil, ctx.Err()
				},
				service: &operatorCleanupTestService{
					describe: func(context.Context, *workflowservicepb.DescribeActivityExecutionRequest) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
						describes++
						return nil, serviceerror.NewNotFound("never started")
					},
				},
			}

			start := time.Now()
			err := run(ctx, c)
			require.ErrorIs(t, err, context.Canceled)
			require.Zero(t, starts, "must not attempt a start after cancellation")
			require.Zero(t, describes, "an unattempted start needs no cleanup")
			require.Zero(t, time.Since(start), "must not wait for the cleanup deadline")
			require.NotContains(t, err.Error(), "clean up standalone activity")
		})
	})

	t.Run("StartMayHaveSucceeded", func(t *testing.T) {
		cases := []struct {
			name string
			err  error
		}{
			{"CanceledAfterDispatch", context.Canceled},
			{"Unavailable", serviceerror.NewUnavailable("response lost")},
			{"DeadlineExceeded", context.DeadlineExceeded},
			{"ResourceExhaustedAfterRetry", serviceerror.NewResourceExhausted(enums.RESOURCE_EXHAUSTED_CAUSE_RPS_LIMIT, "retry rejected")},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					var activityID string
					active := false
					var describes int
					service := &operatorCleanupTestService{
						describe: func(ctx context.Context, request *workflowservicepb.DescribeActivityExecutionRequest) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
							require.NoError(t, ctx.Err(), "cleanup must work after caller cancellation")
							require.Equal(t, activityID, request.ActivityId)
							describes++
							if describes <= 2 {
								return nil, serviceerror.NewNotFound("not observable yet")
							}
							return &workflowservicepb.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{
								RunId: "created-run", Status: enums.ACTIVITY_EXECUTION_STATUS_RUNNING,
							}}, nil
						},
						terminate: func(ctx context.Context, request *workflowservicepb.TerminateActivityExecutionRequest) (*workflowservicepb.TerminateActivityExecutionResponse, error) {
							require.NoError(t, ctx.Err())
							require.Equal(t, "default", request.Namespace)
							require.Equal(t, activityID, request.ActivityId)
							require.Equal(t, "created-run", request.RunId)
							active = false
							return &workflowservicepb.TerminateActivityExecutionResponse{}, nil
						},
					}
					c := &operatorStartTestClient{
						service: service,
						start: func(_ context.Context, options client.StartActivityOptions) (client.ActivityHandle, error) {
							activityID, active = options.ID, true
							if errors.Is(tc.err, context.Canceled) {
								cancel()
							}
							return nil, tc.err
						},
					}

					err := run(ctx, c)
					require.ErrorIs(t, err, tc.err)
					require.False(t, active, "a lost start response must not leave an activity running")
					require.NotContains(t, err.Error(), "clean up standalone activity")
				})
			})
		}
	})

	t.Run("CleanupFailure", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			startErr := serviceerror.NewUnavailable("response lost")
			cleanupErr := serviceerror.NewPermissionDenied("cannot terminate", "test")
			c := &operatorStartTestClient{
				start: func(context.Context, client.StartActivityOptions) (client.ActivityHandle, error) {
					return nil, startErr
				},
				service: &operatorCleanupTestService{
					describe: func(context.Context, *workflowservicepb.DescribeActivityExecutionRequest) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
						return &workflowservicepb.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{
							RunId: "created-run", Status: enums.ACTIVITY_EXECUTION_STATUS_RUNNING,
						}}, nil
					},
					terminate: func(context.Context, *workflowservicepb.TerminateActivityExecutionRequest) (*workflowservicepb.TerminateActivityExecutionResponse, error) {
						return nil, cleanupErr
					},
				},
			}

			err := run(context.Background(), c)
			require.ErrorIs(t, err, startErr)
			require.ErrorIs(t, err, cleanupErr, "cleanup failure must not be hidden by the start error")
		})
	})
}
