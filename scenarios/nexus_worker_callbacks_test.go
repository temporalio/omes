package scenarios

import (
	"context"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/devserver"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	callbackpb "go.temporal.io/api/callback/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	nexusoperationpb "go.temporal.io/api/nexusoperation/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	sdkworker "go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// workerCallbackDynamicConfig enables standalone Nexus operations and worker callbacks on them.
var workerCallbackDynamicConfig = map[string]any{
	"nexusoperation.enableStandalone":     true,
	"nexusoperation.enabledCallbackKinds": []string{"nexusHandler"},
	// Bounds how long an undeliverable callback attempt stays SCHEDULED before it backs off. The
	// default is 10s; shorter keeps the describe lifecycle test quick.
	"callback.request.timeout": "3s",
}

func TestNexusWorkerCallbacksConfigure(t *testing.T) {
	t.Parallel()

	configure := func(t *testing.T, options map[string]string) (*tpsExecutor, error) {
		executor := newThroughputStressExecutor()
		err := executor.Configure(loadgen.ScenarioInfo{
			RunID:   "tps-worker-callbacks",
			Logger:  zap.NewNop().Sugar(),
			Options: capableOfStandaloneNexus(t, options),
		})
		return executor, err
	}

	t.Run("off by default", func(t *testing.T) {
		t.Parallel()
		executor, err := configure(t, nil)
		require.NoError(t, err)
		require.False(t, executor.config.IncludeNexusWorkerCallbacks)
	})

	t.Run("enabled with standalone Nexus", func(t *testing.T) {
		t.Parallel()
		executor, err := configure(t, map[string]string{
			IncludeNexusWorkerCallbacksFlag:                "true",
			NexusWorkerCallbacksMaxConcurrentFlag:          "5",
			NexusWorkerCallbacksMaxIterationsPerSecondFlag: "2.5",
			NexusWorkerCallbacksDrainTimeoutFlag:           "45s",
		})
		require.NoError(t, err)
		require.True(t, executor.config.IncludeNexusWorkerCallbacks)
		require.Equal(t, independentLoadConfig{
			MaxConcurrent:          5,
			MaxIterationsPerSecond: 2.5,
			DrainTimeout:           45 * time.Second,
		}, executor.config.NexusWorkerCallbacks)
	})

	for name, options := range map[string]map[string]string{
		"requires standalone Nexus": {
			IncludeNexusWorkerCallbacksFlag: "true",
			IncludeStandaloneNexusFlag:      "false",
		},
		"requires Nexus": {
			IncludeNexusWorkerCallbacksFlag: "true",
			NexusEnabledFlag:                "false",
		},
		"non-positive max concurrent": {
			IncludeNexusWorkerCallbacksFlag:       "true",
			NexusWorkerCallbacksMaxConcurrentFlag: "0",
		},
		"negative rate": {
			IncludeNexusWorkerCallbacksFlag:                "true",
			NexusWorkerCallbacksMaxIterationsPerSecondFlag: "-1",
		},
		"non-positive drain timeout": {
			IncludeNexusWorkerCallbacksFlag:      "true",
			NexusWorkerCallbacksDrainTimeoutFlag: "0s",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			_, err := configure(t, options)
			require.Error(t, err)
		})
	}

	t.Run("limits are ignored while off", func(t *testing.T) {
		t.Parallel()
		_, err := configure(t, map[string]string{NexusWorkerCallbacksMaxConcurrentFlag: "0"})
		require.NoError(t, err)
	})
}

func TestBuildWorkerCallbackStartRequest(t *testing.T) {
	t.Parallel()

	for _, expect := range []ks.WorkerCallbackExpectation{ks.WorkerCallbackExpectSuccess, ks.WorkerCallbackExpectTimeout} {
		t.Run(string(expect), func(t *testing.T) {
			t.Parallel()
			req, err := buildWorkerCallbackStartRequest("ns", "endpoint", "tq", "op-id", expect)
			require.NoError(t, err)

			require.Equal(t, "ns", req.GetNamespace())
			require.Equal(t, "endpoint", req.GetEndpoint())
			require.Equal(t, "op-id", req.GetOperationId())
			require.Equal(t, ks.KitchenSinkNexusServiceName, req.GetService())
			require.Equal(t, ks.KitchenSinkNexusOperationName, req.GetOperation())
			require.NotEmpty(t, req.GetRequestId())

			require.Len(t, req.GetCompletionCallbacks(), 1)
			handler := req.GetCompletionCallbacks()[0].GetNexusHandler()
			require.NotNil(t, handler, "callback must be the NexusHandler variant")
			require.Equal(t, "tq", handler.GetTaskQueueName())
			require.Equal(t, ks.WorkerCallbackServiceName, handler.GetService())
			require.Equal(t, ks.WorkerCallbackOperationName, handler.GetOperation())

			var sc ks.WorkerCallbackSourceContext
			require.NoError(t, converter.GetDefaultDataConverter().FromPayload(handler.GetSourceContext(), &sc))
			require.Equal(t, "op-id", sc.OperationID)
			require.Equal(t, expect, sc.Expect)

			var input ks.NexusOperationRequest
			require.NoError(t, converter.GetDefaultDataConverter().FromPayload(req.GetInput(), &input))
			switch expect {
			case ks.WorkerCallbackExpectSuccess:
				require.NotEmpty(t, sc.EchoValue)
				require.Equal(t, sc.EchoValue, input.GetEcho(), "the handler must expect what the operation echoes")
				require.Equal(t, workerCallbackSuccessScheduleToClose, req.GetScheduleToCloseTimeout().AsDuration())
			case ks.WorkerCallbackExpectTimeout:
				require.NotNil(t, input.GetWorkflowAction().GetStart())
				require.Less(t, req.GetScheduleToCloseTimeout().AsDuration(), workerCallbackTimeoutWorkflowSleep,
					"the operation must time out before its workflow completes")
			}
		})
	}

	_, err := buildWorkerCallbackStartRequest("ns", "endpoint", "tq", "op-id", "bogus")
	require.Error(t, err)
}

func TestWorkerCallbackExpectationForIteration(t *testing.T) {
	t.Parallel()
	// Iterations are 1-based, so a single-iteration run covers the success case.
	require.Equal(t, ks.WorkerCallbackExpectSuccess, workerCallbackExpectationForIteration(1))
	require.Equal(t, ks.WorkerCallbackExpectTimeout, workerCallbackExpectationForIteration(2))
	require.Equal(t, ks.WorkerCallbackExpectSuccess, workerCallbackExpectationForIteration(3))
}

// TestNexusWorkerCallbacks runs worker callbacks end to end against a dev server: as the standalone
// scenario, as an independent load in throughput_stress, and through each callback state as
// reported by DescribeNexusOperationExecution. The subtests share one dev server.
func TestNexusWorkerCallbacks(t *testing.T) {
	t.Parallel()

	server := workertest.StartDevServer(t, workertest.WithDynamicConfig(workerCallbackDynamicConfig))

	t.Run("describe", func(t *testing.T) { testNexusWorkerCallbacksDescribe(t, server) })

	t.Run("scenario", func(t *testing.T) {
		env := workertest.SetupTestEnvironment(t,
			workertest.WithExecutorTimeout(3*time.Minute),
			workertest.WithDevServer(server))

		scenario := loadgen.GetScenario("nexus_worker_callbacks")
		require.NotNil(t, scenario)
		info := loadgen.ScenarioInfo{
			RunID: "nexus-worker-callbacks",
			// Four iterations cover the success and timeout cases twice each.
			Configuration: loadgen.RunConfiguration{Iterations: 4, MaxConcurrent: 4},
			Options:       loadgen.MustResolveScenarioOptions("nexus_worker_callbacks", nil),
		}
		_, err := env.RunExecutorTest(t, scenario.ExecutorFn(), info, clioptions.LangGo)
		require.NoError(t, err)
	})

	t.Run("throughput_stress", func(t *testing.T) {
		env := workertest.SetupTestEnvironment(t,
			workertest.WithExecutorTimeout(3*time.Minute),
			workertest.WithDevServer(server))

		info := loadgen.ScenarioInfo{
			RunID:         "tps-nexus-worker-callbacks",
			Configuration: loadgen.RunConfiguration{Iterations: 1},
			Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
				IterFlag:                              "1",
				ContinueAsNewAfterIterFlag:            "1",
				SleepTimeFlag:                         "1ms",
				VisibilityVerificationTimeoutFlag:     "10s",
				NexusEnabledFlag:                      "true",
				IncludeStandaloneNexusFlag:            "true",
				IncludeNexusWorkerCallbacksFlag:       "true",
				NexusWorkerCallbacksMaxConcurrentFlag: "2",
			}),
		}
		result, err := env.RunExecutorTest(t, newThroughputStressExecutor(), info, clioptions.LangGo)
		require.NoError(t, err)

		summaries := result.ObservedLogs.FilterMessageSnippet("Nexus worker callback load summary").All()
		require.Len(t, summaries, 1)
		match := regexp.MustCompile(`started=(\d+) completed=(\d+)`).FindStringSubmatch(summaries[0].Message)
		require.NotNil(t, match, "unexpected summary: %s", summaries[0].Message)
		started, _ := strconv.Atoi(match[1])
		completed, _ := strconv.Atoi(match[2])
		require.Positive(t, completed, "the worker callback load must complete at least one test")
		require.Equal(t, started, completed, "every started worker callback test must complete")
	})
}

// testNexusWorkerCallbacksDescribe drives a SANO's worker callback through each state it can be
// observed in, and checks how DescribeNexusOperationExecution reports it at every step.
func testNexusWorkerCallbacksDescribe(t *testing.T, server *devserver.Server) {
	// STANDBY -> SCHEDULED -> BACKING_OFF -> SCHEDULED -> SUCCEEDED: the callback waits while the
	// operation runs, can't be delivered while nothing polls its task queue, and is delivered on a
	// retry once a handler starts polling.
	t.Run("standby, backing off, then succeeded", func(t *testing.T) {
		t.Parallel()
		runWithKitchenSinkWorker(t, server, "wcb-describe-lifecycle", func(ctx context.Context, info loadgen.ScenarioInfo, endpoint string) {
			operationID := "wcb-describe-lifecycle-" + uuid.NewString()
			workflowID := operationID + "-workflow"
			// Nothing polls this task queue until the test starts a handler on it.
			callbackTaskQueue := operationID + "-callbacks"

			// The operation starts a kitchen sink workflow that runs until it is signaled.
			input := &ks.NexusOperationRequest{Action: &ks.NexusOperationRequest_WorkflowAction{
				WorkflowAction: &ks.NexusWorkflowAction{
					WorkflowId: workflowID,
					Action:     &ks.NexusWorkflowAction_Start{Start: &emptypb.Empty{}},
				},
			}}
			const result = "released"
			req := newWorkerCallbackStartRequest(info.Namespace, endpoint, operationID, input, time.Minute,
				callbackTaskQueue, ks.WorkerCallbackSourceContext{
					OperationID: operationID,
					Expect:      ks.WorkerCallbackExpectSuccess,
					EchoValue:   result,
				})
			want := req.GetCompletionCallbacks()[0].GetNexusHandler()
			started, err := info.Client.WorkflowService().StartNexusOperationExecution(ctx, req)
			require.NoError(t, err)
			op := describedOperation{info: info, operationID: operationID, runID: started.GetRunId(), want: want}

			// While the operation runs, the callback waits for it.
			resp, cb := op.describe(ctx, t)
			require.Equal(t, enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING, resp.GetInfo().GetStatus())
			require.Equal(t, enumspb.CALLBACK_STATE_STANDBY, cb.GetState())
			require.Zero(t, cb.GetAttempt())

			// Complete the operation by releasing its workflow. The workflow may not have started yet.
			require.EventuallyWithT(t, func(c *assert.CollectT) {
				err := info.Client.SignalWorkflow(ctx, workflowID, "", "do_actions_signal", &ks.DoSignal_DoSignalActions{
					Variant: &ks.DoSignal_DoSignalActions_DoActions{
						DoActions: ks.SingleActionSet(ks.NewReturnResultAction(ks.ConvertToPayload(result))),
					},
				})
				assert.NoError(c, err)
			}, 30*time.Second, 100*time.Millisecond)

			// With nothing polling its task queue, the first attempt is in flight until it times
			// out, then backs off.
			resp, cb, seen := op.awaitState(ctx, t, enumspb.CALLBACK_STATE_BACKING_OFF)
			require.Equal(t, enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED, resp.GetInfo().GetStatus())
			require.Contains(t, seen, enumspb.CALLBACK_STATE_SCHEDULED,
				"the callback must be SCHEDULED while its first attempt is in flight")
			require.EqualValues(t, 1, cb.GetAttempt())
			require.NotEmpty(t, cb.GetLastAttemptFailure().GetMessage())
			require.True(t, cb.GetNextAttemptScheduleTime().AsTime().After(cb.GetLastAttemptCompleteTime().AsTime()),
				"the next attempt must be scheduled after the failed one")

			// Once a handler polls the task queue, a retry delivers the callback.
			startWorkerCallbackHandler(t, info.Client, callbackTaskQueue)
			resp, cb, _ = op.awaitState(ctx, t, enumspb.CALLBACK_STATE_SUCCEEDED)
			require.Equal(t, enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED, resp.GetInfo().GetStatus())
			require.GreaterOrEqual(t, cb.GetAttempt(), int32(2))
			require.Nil(t, cb.GetLastAttemptFailure(), "a successful attempt clears the earlier failure")
		})
	})

	// SCHEDULED -> FAILED: the handler rejects the delivered outcome, which fails the callback
	// without retrying it.
	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		runWithKitchenSinkWorker(t, server, "wcb-describe-failed", func(ctx context.Context, info loadgen.ScenarioInfo, endpoint string) {
			operationID := "wcb-describe-failed-" + uuid.NewString()
			input := &ks.NexusOperationRequest{Action: &ks.NexusOperationRequest_Echo{Echo: "echoed"}}
			// The Go worker's handler on the run's task queue rejects a result it doesn't expect.
			req := newWorkerCallbackStartRequest(info.Namespace, endpoint, operationID, input, time.Minute,
				loadgen.TaskQueueForRun(info.RunID), ks.WorkerCallbackSourceContext{
					OperationID: operationID,
					Expect:      ks.WorkerCallbackExpectSuccess,
					EchoValue:   "something else",
				})
			started, err := info.Client.WorkflowService().StartNexusOperationExecution(ctx, req)
			require.NoError(t, err)
			op := describedOperation{
				info: info, operationID: operationID, runID: started.GetRunId(),
				want: req.GetCompletionCallbacks()[0].GetNexusHandler(),
			}

			resp, cb, _ := op.awaitState(ctx, t, enumspb.CALLBACK_STATE_FAILED)
			require.Equal(t, enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED, resp.GetInfo().GetStatus(),
				"a failed callback doesn't change the operation's outcome")
			require.EqualValues(t, 1, cb.GetAttempt(), "a non-retryable failure must not be retried")
			require.Contains(t, cb.GetLastAttemptFailure().GetMessage(), "result mismatch")
		})
	})
}

// runWithKitchenSinkWorker runs fn with the Go worker polling the run's task queue, and a Nexus
// endpoint for the kitchen sink operation that targets it.
func runWithKitchenSinkWorker(
	t *testing.T,
	server *devserver.Server,
	runID string,
	fn func(ctx context.Context, info loadgen.ScenarioInfo, endpoint string),
) {
	t.Helper()
	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(3*time.Minute),
		workertest.WithDevServer(server))
	info := loadgen.ScenarioInfo{
		RunID:   runID,
		Options: loadgen.MustResolveScenarioOptions("nexus_worker_callbacks", nil),
	}
	_, err := env.RunExecutorTest(t, loadgen.ExecutorFunc(func(ctx context.Context, info loadgen.ScenarioInfo) error {
		endpoint, err := info.EnsureNexusEndpoint(ctx)
		if err != nil {
			return err
		}
		fn(ctx, info, endpoint)
		return nil
	}), info, clioptions.LangGo)
	require.NoError(t, err)
}

// startWorkerCallbackHandler starts a worker on taskQueue that hosts the worker callback handler.
func startWorkerCallbackHandler(t *testing.T, c client.Client, taskQueue string) {
	t.Helper()
	service := nexus.NewService(ks.WorkerCallbackServiceName)
	require.NoError(t, service.Register(ks.WorkerCallbackOperation))
	w := sdkworker.New(c, taskQueue, sdkworker.Options{})
	w.RegisterNexusService(service)
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)
}

// describedOperation describes a SANO with a worker callback, checking every response with
// checkWorkerCallbackDescription.
type describedOperation struct {
	info               loadgen.ScenarioInfo
	operationID, runID string
	want               *commonpb.Callback_NexusHandler
}

func (o describedOperation) describe(ctx context.Context, t *testing.T) (
	*workflowservice.DescribeNexusOperationExecutionResponse, *callbackpb.CallbackInfo,
) {
	t.Helper()
	resp, err := describeNexusOperation(ctx, o.info, o.operationID, o.runID)
	require.NoError(t, err)
	cb, err := checkWorkerCallbackDescription(resp, o.want)
	require.NoError(t, err)
	return resp, cb
}

// awaitState polls until the callback reaches state, and also returns each distinct state seen on
// the way, in order.
func (o describedOperation) awaitState(ctx context.Context, t *testing.T, state enumspb.CallbackState) (
	*workflowservice.DescribeNexusOperationExecutionResponse, *callbackpb.CallbackInfo, []enumspb.CallbackState,
) {
	t.Helper()
	var seen []enumspb.CallbackState
	deadline := time.Now().Add(time.Minute)
	for {
		resp, cb := o.describe(ctx, t)
		if len(seen) == 0 || seen[len(seen)-1] != cb.GetState() {
			seen = append(seen, cb.GetState())
		}
		if cb.GetState() == state {
			return resp, cb, seen
		}
		require.Truef(t, time.Now().Before(deadline),
			"callback did not reach %v; states seen: %v; last: %v", state, seen, cb)
		time.Sleep(50 * time.Millisecond)
	}
}

func TestCheckWorkerCallbackDescription(t *testing.T) {
	t.Parallel()

	want := &commonpb.Callback_NexusHandler{
		TaskQueueName: "tq",
		Service:       ks.WorkerCallbackServiceName,
		Operation:     ks.WorkerCallbackOperationName,
		SourceContext: ks.ConvertToPayload(ks.WorkerCallbackSourceContext{OperationID: "op", Expect: ks.WorkerCallbackExpectSuccess}),
	}
	now := timestamppb.Now()
	retryable := &failurepb.Failure{
		Message:     "request timeout",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{}},
	}
	nonRetryable := &failurepb.Failure{
		Message: "handler error (BAD_REQUEST)",
		FailureInfo: &failurepb.Failure_ApplicationFailureInfo{
			ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{NonRetryable: true},
		},
	}

	// callback returns a valid CallbackInfo for state, which each case then breaks (or not).
	callback := func(state enumspb.CallbackState) *callbackpb.CallbackInfo {
		info := &callbackpb.CallbackInfo{
			Callback:         &commonpb.Callback{Variant: &commonpb.Callback_NexusHandler_{NexusHandler: proto.Clone(want).(*commonpb.Callback_NexusHandler)}},
			RegistrationTime: now,
			State:            state,
		}
		switch state {
		case enumspb.CALLBACK_STATE_BLOCKED:
			info.BlockedReason = "The circuit breaker is open."
		case enumspb.CALLBACK_STATE_BACKING_OFF:
			info.Attempt, info.LastAttemptCompleteTime = 1, now
			info.LastAttemptFailure, info.NextAttemptScheduleTime = retryable, now
		case enumspb.CALLBACK_STATE_SUCCEEDED:
			info.Attempt, info.LastAttemptCompleteTime = 2, now
		case enumspb.CALLBACK_STATE_FAILED:
			info.Attempt, info.LastAttemptCompleteTime, info.LastAttemptFailure = 1, now, nonRetryable
		}
		return info
	}
	describe := func(status enumspb.NexusOperationExecutionStatus, callbacks ...*callbackpb.CallbackInfo) *workflowservice.DescribeNexusOperationExecutionResponse {
		resp := &workflowservice.DescribeNexusOperationExecutionResponse{
			Info: &nexuspb.NexusOperationExecutionInfo{Status: status},
		}
		for _, cb := range callbacks {
			resp.CompletionCallbacks = append(resp.CompletionCallbacks, &nexusoperationpb.CallbackInfo{
				Trigger: &nexusoperationpb.CallbackInfo_Trigger{Variant: &nexusoperationpb.CallbackInfo_Trigger_OperationCompleted{
					OperationCompleted: &nexusoperationpb.CallbackInfo_OperationCompleted{},
				}},
				Info: cb,
			})
		}
		return resp
	}
	modified := func(state enumspb.CallbackState, modify func(*callbackpb.CallbackInfo)) *callbackpb.CallbackInfo {
		cb := callback(state)
		modify(cb)
		return cb
	}

	const (
		running   = enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING
		completed = enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED
	)
	cases := []struct {
		name    string
		resp    *workflowservice.DescribeNexusOperationExecutionResponse
		wantErr string
	}{
		{name: "standby while running", resp: describe(running, callback(enumspb.CALLBACK_STATE_STANDBY))},
		{name: "scheduled after close", resp: describe(completed, callback(enumspb.CALLBACK_STATE_SCHEDULED))},
		{name: "blocked after close", resp: describe(completed, callback(enumspb.CALLBACK_STATE_BLOCKED))},
		{name: "backing off after close", resp: describe(completed, callback(enumspb.CALLBACK_STATE_BACKING_OFF))},
		{name: "succeeded after close", resp: describe(completed, callback(enumspb.CALLBACK_STATE_SUCCEEDED))},
		{name: "failed after close", resp: describe(enumspb.NEXUS_OPERATION_EXECUTION_STATUS_TIMED_OUT, callback(enumspb.CALLBACK_STATE_FAILED))},

		{name: "no callbacks", resp: describe(running), wantErr: "expected 1 completion callback, got 0"},
		{
			name:    "two callbacks",
			resp:    describe(running, callback(enumspb.CALLBACK_STATE_STANDBY), callback(enumspb.CALLBACK_STATE_STANDBY)),
			wantErr: "expected 1 completion callback, got 2",
		},
		{
			name: "missing trigger",
			resp: func() *workflowservice.DescribeNexusOperationExecutionResponse {
				resp := describe(running, callback(enumspb.CALLBACK_STATE_STANDBY))
				resp.CompletionCallbacks[0].Trigger = nil
				return resp
			}(),
			wantErr: "want operation completed",
		},
		{
			name: "not a worker callback",
			resp: describe(running, modified(enumspb.CALLBACK_STATE_STANDBY, func(cb *callbackpb.CallbackInfo) {
				cb.Callback = &commonpb.Callback{Variant: &commonpb.Callback_Nexus_{Nexus: &commonpb.Callback_Nexus{Url: "http://x"}}}
			})),
			wantErr: "not a worker callback",
		},
		{
			name: "wrong task queue",
			resp: describe(running, modified(enumspb.CALLBACK_STATE_STANDBY, func(cb *callbackpb.CallbackInfo) {
				cb.GetCallback().GetNexusHandler().TaskQueueName = "other"
			})),
			wantErr: "worker callback reported as",
		},
		{
			name: "wrong source context",
			resp: describe(running, modified(enumspb.CALLBACK_STATE_STANDBY, func(cb *callbackpb.CallbackInfo) {
				cb.GetCallback().GetNexusHandler().SourceContext = ks.ConvertToPayload("other")
			})),
			wantErr: "worker callback reported as",
		},
		{
			name: "no registration time",
			resp: describe(running, modified(enumspb.CALLBACK_STATE_STANDBY, func(cb *callbackpb.CallbackInfo) {
				cb.RegistrationTime = nil
			})),
			wantErr: "no registration time",
		},
		{
			name:    "scheduled while running",
			resp:    describe(running, callback(enumspb.CALLBACK_STATE_SCHEDULED)),
			wantErr: "must be in STANDBY while the operation is running",
		},
		{
			name:    "standby after close",
			resp:    describe(completed, callback(enumspb.CALLBACK_STATE_STANDBY)),
			wantErr: "must leave STANDBY",
		},
		{
			name: "standby with an attempt",
			resp: describe(running, modified(enumspb.CALLBACK_STATE_STANDBY, func(cb *callbackpb.CallbackInfo) {
				cb.Attempt = 1
			})),
			wantErr: "must not have attempted delivery yet",
		},
		{
			name: "blocked reason outside BLOCKED",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_SCHEDULED, func(cb *callbackpb.CallbackInfo) {
				cb.BlockedReason = "The circuit breaker is open."
			})),
			wantErr: "only BLOCKED callbacks have one",
		},
		{
			name: "blocked without a reason",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_BLOCKED, func(cb *callbackpb.CallbackInfo) {
				cb.BlockedReason = ""
			})),
			wantErr: "only BLOCKED callbacks have one",
		},
		{
			name: "scheduled with a next attempt time",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_SCHEDULED, func(cb *callbackpb.CallbackInfo) {
				cb.NextAttemptScheduleTime = now
			})),
			wantErr: "must not have a next attempt time",
		},
		{
			name: "backing off without a next attempt time",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_BACKING_OFF, func(cb *callbackpb.CallbackInfo) {
				cb.NextAttemptScheduleTime = nil
			})),
			wantErr: "when the next one is due",
		},
		{
			name: "backing off without a failure",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_BACKING_OFF, func(cb *callbackpb.CallbackInfo) {
				cb.LastAttemptFailure = nil
			})),
			wantErr: "must record the failed attempt",
		},
		{
			name: "backing off after a non-retryable failure",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_BACKING_OFF, func(cb *callbackpb.CallbackInfo) {
				cb.LastAttemptFailure = nonRetryable
			})),
			wantErr: "only back off after a retryable failure",
		},
		{
			name: "succeeded with a failure",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_SUCCEEDED, func(cb *callbackpb.CallbackInfo) {
				cb.LastAttemptFailure = retryable
			})),
			wantErr: "must record the successful attempt",
		},
		{
			name: "succeeded without an attempt",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_SUCCEEDED, func(cb *callbackpb.CallbackInfo) {
				cb.Attempt, cb.LastAttemptCompleteTime = 0, nil
			})),
			wantErr: "must record the successful attempt",
		},
		{
			name: "failed with a retryable failure",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_FAILED, func(cb *callbackpb.CallbackInfo) {
				cb.LastAttemptFailure = retryable
			})),
			wantErr: "must record a non-retryable failure",
		},
		{
			name: "failed with a next attempt time",
			resp: describe(completed, modified(enumspb.CALLBACK_STATE_FAILED, func(cb *callbackpb.CallbackInfo) {
				cb.NextAttemptScheduleTime = now
			})),
			wantErr: "no next attempt",
		},
		{
			name:    "unspecified state",
			resp:    describe(completed, callback(enumspb.CALLBACK_STATE_UNSPECIFIED)),
			wantErr: "not a known callback state",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cb, err := checkWorkerCallbackDescription(tc.resp, want)
			if tc.wantErr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.resp.GetCompletionCallbacks()[0].GetInfo(), cb)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}
