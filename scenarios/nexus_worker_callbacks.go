package scenarios

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	callbackpb "go.temporal.io/api/callback/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Worker callbacks are NexusHandler-variant completion callbacks attached to standalone Nexus
// operations (SANOs). When the SANO completes, the server delivers its outcome to a Nexus operation
// on a worker task queue in the same namespace. See loadgen/kitchensink/worker_callback.go for the
// contract between this client and the Go worker's handler.
//
// Each iteration starts a SANO with a worker callback attached, and waits for the server to report
// the callback as succeeded. The handler only succeeds when the delivered outcome matches the
// expectation in the callback's source context, so a SUCCEEDED callback means the right outcome was
// delivered. Iterations alternate between a successful operation and one that times out.
//
// Requires the Go worker (the only one implementing the handler), standalone Nexus operations, and
// the nexusHandler callback kind ("nexusoperation.enabledCallbackKinds" dynamic config).

const (
	// IncludeNexusWorkerCallbacksFlag runs worker callbacks on standalone Nexus operations as an
	// independent load in throughput_stress. Opt-in: the server reports no capability for the
	// nexusHandler callback kind, and only the Go worker implements the handler.
	IncludeNexusWorkerCallbacksFlag = "include-nexus-worker-callbacks"
	// NexusWorkerCallbacksMaxConcurrentFlag limits concurrent worker callback tests. Default is 3.
	NexusWorkerCallbacksMaxConcurrentFlag = "nexus-worker-callbacks-max-concurrent"
	// NexusWorkerCallbacksMaxIterationsPerSecondFlag limits worker callback test starts per
	// second. Default is 0, meaning no rate cap.
	NexusWorkerCallbacksMaxIterationsPerSecondFlag = "nexus-worker-callbacks-max-iterations-per-second"
	// NexusWorkerCallbacksDrainTimeoutFlag sets how long running worker callback tests may finish
	// after the main load ends before they are canceled. Default is 1m.
	NexusWorkerCallbacksDrainTimeoutFlag = "nexus-worker-callbacks-drain-timeout"
)

const (
	workerCallbackIdentity = "omes-nexus-worker-callbacks"
	// workerCallbackSuccessScheduleToClose bounds the echo operation, which completes synchronously.
	workerCallbackSuccessScheduleToClose = time.Minute
	// workerCallbackTimeoutScheduleToClose is how long the operation that is meant to time out gets.
	// Its handler workflow sleeps for workerCallbackTimeoutWorkflowSleep, then completes on its own,
	// so the workflows these operations start don't linger.
	workerCallbackTimeoutScheduleToClose = 2 * time.Second
	workerCallbackTimeoutWorkflowSleep   = 10 * time.Second
	// workerCallbackWaitTimeout bounds how long an iteration waits for the callback to resolve.
	workerCallbackWaitTimeout  = 2 * time.Minute
	workerCallbackPollInterval = 500 * time.Millisecond
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Worker callbacks (NexusHandler completion callbacks) on standalone Nexus operations, " +
			"driven directly by the client. Iterations alternate between an operation that succeeds and one " +
			"that times out, and verify the worker's handler received the outcome. Requires the Go worker " +
			"and the server's nexusHandler callback kind.",
		Options: func(o *loadgen.OptionSet) {
			loadgen.DeclareNexusOptions(o)
		},
		ExecutorFn: func() loadgen.Executor {
			return loadgen.ExecutorFunc(runNexusWorkerCallbacksScenario)
		},
	})
}

func runNexusWorkerCallbacksScenario(ctx context.Context, info loadgen.ScenarioInfo) error {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	namespace, err := info.Client.WorkflowService().DescribeNamespace(probeCtx,
		&workflowservice.DescribeNamespaceRequest{Namespace: info.Namespace})
	cancel()
	if err != nil {
		return fmt.Errorf("check standalone Nexus operation support: %w", err)
	}
	if !namespace.GetNamespaceInfo().GetCapabilities().GetStandaloneNexusOperation() {
		return fmt.Errorf("namespace %q does not report standalone Nexus operation support", info.Namespace)
	}

	nexus, err := info.ResolveNexusConfig(ctx)
	if err != nil {
		return err
	}
	if !nexus.Enabled {
		return fmt.Errorf("worker callbacks need Nexus, but %s resolved to false", loadgen.NexusEnabledOption)
	}
	return runNexusWorkerCallbacks(ctx, info, nexus.Endpoint, nil)
}

func declareNexusWorkerCallbacksOptions(o *loadgen.OptionSet) {
	o.Bool(IncludeNexusWorkerCallbacksFlag, false,
		"Run worker callbacks on standalone Nexus operations in an independent driver loop "+
			"(Go worker only; requires the server's nexusHandler callback kind and "+IncludeStandaloneNexusFlag+").")
	o.Int(NexusWorkerCallbacksMaxConcurrentFlag, 3, "Maximum concurrent worker callback tests.")
	o.Float64(NexusWorkerCallbacksMaxIterationsPerSecondFlag, 0, "Maximum worker callback test starts per second (0 means unlimited).")
	o.Duration(NexusWorkerCallbacksDrainTimeoutFlag, time.Minute, "Time for running worker callback tests to finish after the main load ends before cancellation.")
}

// configureNexusWorkerCallbacks reads the worker callback options. It must run after
// IncludeStandaloneNexus is settled, since worker callbacks are attached to standalone operations.
func (config *tpsConfig) configureNexusWorkerCallbacks(info loadgen.ScenarioInfo) error {
	config.IncludeNexusWorkerCallbacks = info.OptionBool(IncludeNexusWorkerCallbacksFlag)
	config.NexusWorkerCallbacks = independentLoadConfig{
		MaxConcurrent:          info.OptionInt(NexusWorkerCallbacksMaxConcurrentFlag),
		MaxIterationsPerSecond: info.OptionFloat64(NexusWorkerCallbacksMaxIterationsPerSecondFlag),
		DrainTimeout:           info.OptionDuration(NexusWorkerCallbacksDrainTimeoutFlag),
	}
	if !config.IncludeNexusWorkerCallbacks {
		return nil
	}

	if !config.IncludeStandaloneNexus {
		return fmt.Errorf("%s requires %s", IncludeNexusWorkerCallbacksFlag, IncludeStandaloneNexusFlag)
	}
	if config.NexusWorkerCallbacks.MaxConcurrent <= 0 {
		return fmt.Errorf("%s must be positive", NexusWorkerCallbacksMaxConcurrentFlag)
	}
	loadConfig := loadgen.RunConfiguration{
		MaxConcurrent:          config.NexusWorkerCallbacks.MaxConcurrent,
		MaxIterationsPerSecond: config.NexusWorkerCallbacks.MaxIterationsPerSecond,
	}
	if err := loadConfig.Validate(); err != nil {
		return fmt.Errorf("%s: %w", NexusWorkerCallbacksMaxIterationsPerSecondFlag, err)
	}
	if config.NexusWorkerCallbacks.DrainTimeout <= 0 {
		return fmt.Errorf("%s must be positive", NexusWorkerCallbacksDrainTimeoutFlag)
	}
	return nil
}

func (t *tpsExecutor) nexusWorkerCallbacksLoad(info loadgen.ScenarioInfo) independentLoad {
	loadInfo := info
	loadInfo.ScenarioName = "throughput_stress_nexus_worker_callbacks"
	loadInfo.Logger = info.Logger.With("load_stream", "nexus_worker_callbacks")
	// Use separate scheduling limits and progress tracking for worker callback tests.
	loadInfo.Configuration = loadgen.RunConfiguration{
		MaxConcurrent:                 t.config.NexusWorkerCallbacks.MaxConcurrent,
		MaxIterationsPerSecond:        t.config.NexusWorkerCallbacks.MaxIterationsPerSecond,
		MaxIterationAttempts:          info.Configuration.MaxIterationAttempts,
		ContinueOnIterationFailure:    info.Configuration.ContinueOnIterationFailure,
		DoNotRegisterSearchAttributes: true,
	}
	endpoint := t.config.NexusEndpoint
	return independentLoad{
		Name:         "nexus worker callbacks",
		DrainTimeout: t.config.NexusWorkerCallbacks.DrainTimeout,
		Run: func(ctx context.Context, stopNewIterations <-chan struct{}) error {
			return runNexusWorkerCallbacks(ctx, loadInfo, endpoint, stopNewIterations)
		},
	}
}

func runNexusWorkerCallbacks(ctx context.Context, info loadgen.ScenarioInfo, endpoint string, stopNewIterations <-chan struct{}) error {
	var started, completed atomic.Int64
	defer func() {
		info.Logger.Infof("Nexus worker callback load summary: started=%d completed=%d", started.Load(), completed.Load())
	}()

	// No workflows are started directly, so no workflow search attributes are needed.
	info.Configuration.DoNotRegisterSearchAttributes = true
	previousOnCompletion := info.Configuration.OnCompletion
	info.Configuration.OnCompletion = func(ctx context.Context, run *loadgen.Run) {
		completed.Add(1)
		if previousOnCompletion != nil {
			previousOnCompletion(ctx, run)
		}
	}

	executor := loadgen.GenericExecutor{
		StopNewIterations: stopNewIterations,
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			started.Add(1)
			expect := workerCallbackExpectationForIteration(run.Iteration)
			return executeNexusWorkerCallback(ctx, info, run, endpoint, expect)
		},
	}
	return executor.Run(ctx, info)
}

// workerCallbackExpectationForIteration alternates between the success and timeout cases, starting
// with success.
func workerCallbackExpectationForIteration(iteration int) ks.WorkerCallbackExpectation {
	if iteration%2 == 0 {
		return ks.WorkerCallbackExpectTimeout
	}
	return ks.WorkerCallbackExpectSuccess
}

// executeNexusWorkerCallback starts a SANO with a worker callback and waits for the callback to
// resolve. Each attempt uses a fresh operation ID, so a retried iteration starts a new operation.
func executeNexusWorkerCallback(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	run *loadgen.Run,
	endpoint string,
	expect ks.WorkerCallbackExpectation,
) error {
	operationID := fmt.Sprintf("nexus-worker-callback-%s-%d-%s", run.RunID, run.Iteration, uuid.NewString())
	req, err := buildWorkerCallbackStartRequest(info.Namespace, endpoint, run.TaskQueue(), operationID, expect)
	if err != nil {
		return err
	}

	resp, err := info.Client.WorkflowService().StartNexusOperationExecution(ctx, req)
	if err != nil {
		return fmt.Errorf("start standalone Nexus operation %q with worker callback: %w", operationID, err)
	}

	ctx, cancel := context.WithTimeout(ctx, workerCallbackWaitTimeout)
	defer cancel()
	want := req.GetCompletionCallbacks()[0].GetNexusHandler()
	return awaitWorkerCallbackSucceeded(ctx, info, operationID, resp.GetRunId(), want, expect)
}

// buildWorkerCallbackStartRequest builds the StartNexusOperationExecution request for one test.
//
// TODO(https://temporalio.atlassian.net/browse/NEXUS-466): The Go SDK cannot attach worker
// callbacks yet, so the operation is started through the gRPC API directly.
func buildWorkerCallbackStartRequest(
	namespace, endpoint, taskQueue, operationID string,
	expect ks.WorkerCallbackExpectation,
) (*workflowservice.StartNexusOperationExecutionRequest, error) {
	sourceContext := ks.WorkerCallbackSourceContext{OperationID: operationID, Expect: expect}
	var (
		input           *ks.NexusOperationRequest
		scheduleToClose time.Duration
	)
	switch expect {
	case ks.WorkerCallbackExpectSuccess:
		sourceContext.EchoValue = "echo-" + operationID
		input = &ks.NexusOperationRequest{
			Action: &ks.NexusOperationRequest_Echo{Echo: sourceContext.EchoValue},
		}
		scheduleToClose = workerCallbackSuccessScheduleToClose
	case ks.WorkerCallbackExpectTimeout:
		// Start a workflow that outlives the operation's schedule-to-close timeout.
		input = &ks.NexusOperationRequest{
			Action: &ks.NexusOperationRequest_WorkflowAction{
				WorkflowAction: &ks.NexusWorkflowAction{
					StartOptions: &ks.NexusWorkflowStartOptions{
						WorkflowInput: &ks.WorkflowInput{
							InitialActions: ks.ListActionSet(
								ks.NewTimerAction(workerCallbackTimeoutWorkflowSleep),
								ks.NewEmptyReturnResultAction(),
							),
						},
					},
					Action: &ks.NexusWorkflowAction_Start{Start: &emptypb.Empty{}},
				},
			},
		}
		scheduleToClose = workerCallbackTimeoutScheduleToClose
	default:
		return nil, fmt.Errorf("unknown worker callback expectation %q", expect)
	}

	return newWorkerCallbackStartRequest(namespace, endpoint, operationID, input, scheduleToClose,
		taskQueue, sourceContext), nil
}

// newWorkerCallbackStartRequest builds a request that runs the kitchen sink Nexus operation with
// input as a SANO, with a worker callback targeting the handler on taskQueue.
func newWorkerCallbackStartRequest(
	namespace, endpoint, operationID string,
	input *ks.NexusOperationRequest,
	scheduleToClose time.Duration,
	taskQueue string,
	sourceContext ks.WorkerCallbackSourceContext,
) *workflowservice.StartNexusOperationExecutionRequest {
	return &workflowservice.StartNexusOperationExecutionRequest{
		Namespace:              namespace,
		Identity:               workerCallbackIdentity,
		RequestId:              uuid.NewString(),
		OperationId:            operationID,
		Endpoint:               endpoint,
		Service:                ks.KitchenSinkNexusServiceName,
		Operation:              ks.KitchenSinkNexusOperationName,
		ScheduleToCloseTimeout: durationpb.New(scheduleToClose),
		Input:                  ks.ConvertToPayload(input),
		CompletionCallbacks: []*commonpb.Callback{{
			Variant: &commonpb.Callback_NexusHandler_{
				NexusHandler: &commonpb.Callback_NexusHandler{
					TaskQueueName: taskQueue,
					Service:       ks.WorkerCallbackServiceName,
					Operation:     ks.WorkerCallbackOperationName,
					SourceContext: ks.ConvertToPayload(sourceContext),
				},
			},
		}},
	}
}

// awaitWorkerCallbackSucceeded polls DescribeNexusOperationExecution until the operation's worker
// callback reaches a terminal state. It fails unless the callback succeeded, which the handler only
// allows when the delivered outcome matched the expectation, and the operation closed as expected.
// Every Describe response along the way is checked with checkWorkerCallbackDescription.
func awaitWorkerCallbackSucceeded(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	operationID, runID string,
	want *commonpb.Callback_NexusHandler,
	expect ks.WorkerCallbackExpectation,
) error {
	wantStatus := enumspb.NEXUS_OPERATION_EXECUTION_STATUS_COMPLETED
	if expect == ks.WorkerCallbackExpectTimeout {
		wantStatus = enumspb.NEXUS_OPERATION_EXECUTION_STATUS_TIMED_OUT
	}

	for {
		resp, err := describeNexusOperation(ctx, info, operationID, runID)
		if err != nil {
			return err
		}
		cb, err := checkWorkerCallbackDescription(resp, want)
		if err != nil {
			return fmt.Errorf("standalone Nexus operation %q: %w", operationID, err)
		}

		status := resp.GetInfo().GetStatus()
		switch cb.GetState() {
		case enumspb.CALLBACK_STATE_SUCCEEDED:
			if status != wantStatus {
				return fmt.Errorf("standalone Nexus operation %q: worker callback succeeded, but status is %v, want %v",
					operationID, status, wantStatus)
			}
			return nil
		case enumspb.CALLBACK_STATE_FAILED:
			return fmt.Errorf("standalone Nexus operation %q (status %v): worker callback failed after %d attempt(s): %s",
				operationID, status, cb.GetAttempt(), cb.GetLastAttemptFailure().GetMessage())
		}

		select {
		case <-ctx.Done():
			err := fmt.Errorf("standalone Nexus operation %q: worker callback did not resolve "+
				"(status: %v, callback state: %v, attempt: %d, blocked reason: %q, last failure: %q)",
				operationID, status, cb.GetState(), cb.GetAttempt(), cb.GetBlockedReason(),
				cb.GetLastAttemptFailure().GetMessage())
			return errors.Join(err, context.Cause(ctx))
		case <-time.After(workerCallbackPollInterval):
		}
	}
}

func describeNexusOperation(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	operationID, runID string,
) (*workflowservice.DescribeNexusOperationExecutionResponse, error) {
	resp, err := info.Client.WorkflowService().DescribeNexusOperationExecution(ctx,
		&workflowservice.DescribeNexusOperationExecutionRequest{
			Namespace:   info.Namespace,
			OperationId: operationID,
			RunId:       runID,
		})
	if err != nil {
		return nil, fmt.Errorf("describe standalone Nexus operation %q: %w", operationID, err)
	}
	return resp, nil
}

// checkWorkerCallbackDescription checks how DescribeNexusOperationExecution reports a SANO's worker
// callback, and returns the callback's info.
//
// Exactly one completion callback must be attached: the worker callback that was requested, with its
// target and source context intact. Its state must be consistent with the operation's status and with
// the callback state machine (chasm/lib/callback/statemachine.go in the server):
//
//   - STANDBY: waiting for its trigger, the operation completing. A running operation's callback
//     must be in STANDBY, and a closed operation's must not be.
//   - SCHEDULED: an attempt is queued or in flight. BLOCKED replaces it while the circuit breaker
//     for the target task queue is open, and is the only state with a blocked reason.
//   - BACKING_OFF: an attempt failed with a retryable error, and the next is scheduled.
//   - SUCCEEDED / FAILED: terminal. FAILED records the non-retryable error.
func checkWorkerCallbackDescription(
	resp *workflowservice.DescribeNexusOperationExecutionResponse,
	want *commonpb.Callback_NexusHandler,
) (*callbackpb.CallbackInfo, error) {
	callbacks := resp.GetCompletionCallbacks()
	if len(callbacks) != 1 {
		// Zero means the server dropped the callback, e.g. it doesn't know the field.
		return nil, fmt.Errorf("expected 1 completion callback, got %d", len(callbacks))
	}
	if callbacks[0].GetTrigger().GetOperationCompleted() == nil {
		return nil, fmt.Errorf("completion callback trigger is %v, want operation completed", callbacks[0].GetTrigger())
	}

	cb := callbacks[0].GetInfo()
	got := cb.GetCallback().GetNexusHandler()
	if got == nil {
		return nil, fmt.Errorf("completion callback is not a worker callback: %v", cb.GetCallback())
	}
	if got.GetTaskQueueName() != want.GetTaskQueueName() ||
		got.GetService() != want.GetService() ||
		got.GetOperation() != want.GetOperation() ||
		!proto.Equal(got.GetSourceContext(), want.GetSourceContext()) {
		return nil, fmt.Errorf("worker callback reported as %v, want %v", got, want)
	}
	if cb.GetRegistrationTime() == nil {
		return nil, errors.New("worker callback has no registration time")
	}

	state := cb.GetState()
	status := resp.GetInfo().GetStatus()
	running := status == enumspb.NEXUS_OPERATION_EXECUTION_STATUS_RUNNING
	describe := func(problem string) error {
		return fmt.Errorf("worker callback in state %v (operation status %v, attempt %d) %s",
			state, status, cb.GetAttempt(), problem)
	}

	if running != (state == enumspb.CALLBACK_STATE_STANDBY) {
		if running {
			return nil, describe("must be in STANDBY while the operation is running")
		}
		return nil, describe("must leave STANDBY once the operation has closed")
	}
	if (state == enumspb.CALLBACK_STATE_BLOCKED) != (cb.GetBlockedReason() != "") {
		return nil, describe(fmt.Sprintf("has blocked reason %q; only BLOCKED callbacks have one", cb.GetBlockedReason()))
	}

	attempted := cb.GetAttempt() > 0 && cb.GetLastAttemptCompleteTime() != nil
	switch state {
	case enumspb.CALLBACK_STATE_STANDBY:
		if cb.GetAttempt() != 0 || cb.GetLastAttemptCompleteTime() != nil || cb.GetLastAttemptFailure() != nil ||
			cb.GetNextAttemptScheduleTime() != nil {
			return nil, describe("must not have attempted delivery yet")
		}
	case enumspb.CALLBACK_STATE_SCHEDULED, enumspb.CALLBACK_STATE_BLOCKED:
		if cb.GetNextAttemptScheduleTime() != nil {
			return nil, describe("must not have a next attempt time while an attempt is scheduled")
		}
	case enumspb.CALLBACK_STATE_BACKING_OFF:
		if !attempted || cb.GetLastAttemptFailure() == nil || cb.GetNextAttemptScheduleTime() == nil {
			return nil, describe("must record the failed attempt and when the next one is due")
		}
		if cb.GetLastAttemptFailure().GetApplicationFailureInfo().GetNonRetryable() {
			return nil, describe("must only back off after a retryable failure")
		}
	case enumspb.CALLBACK_STATE_SUCCEEDED:
		if !attempted || cb.GetLastAttemptFailure() != nil || cb.GetNextAttemptScheduleTime() != nil {
			return nil, describe("must record the successful attempt, with no failure or next attempt")
		}
	case enumspb.CALLBACK_STATE_FAILED:
		if !attempted || cb.GetNextAttemptScheduleTime() != nil {
			return nil, describe("must record the final attempt, with no next attempt")
		}
		if !cb.GetLastAttemptFailure().GetApplicationFailureInfo().GetNonRetryable() {
			return nil, describe(fmt.Sprintf("must record a non-retryable failure, got %v", cb.GetLastAttemptFailure()))
		}
	default:
		return nil, describe("is not a known callback state")
	}
	return cb, nil
}
