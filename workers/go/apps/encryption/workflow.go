package encryption

import (
	"fmt"
	"strings"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// Registered workflow names.
const (
	coverageWorkflowName  = "EncryptionCoverageWorkflow"
	echoChildWorkflowName = "EncryptionEchoChildWorkflow"
	failingWorkflowName   = "EncryptionFailingWorkflow"
	targetWorkflowName    = "EncryptionTargetWorkflow"
	retryWorkflowName     = "EncryptionRetryWorkflow"

	coverageSignalName = "encryption-coverage-signal"
	coverageUpdateName = "encryption-coverage-update"
)

// Change IDs for the version markers. These are the one marker kind generated
// by the SDK for replay rather than by user code, so they never go through the
// data converter and are the only markers a payload check is expected to skip.
// The app emits them so that rule has something to act on.
const (
	versionMarkerChangeID      = "encryption-version-marker"
	otherVersionMarkerChangeID = "encryption-version-marker-2"
)

// Search attributes. Both must already exist in the namespace, or the first upsert
// fails the workflow task.
const (
	keywordSearchAttributeName = "EncryptionKeyword"
	intSearchAttributeName     = "EncryptionInt"
)

var (
	keywordSearchAttribute = temporal.NewSearchAttributeKeyKeyword(keywordSearchAttributeName)
	intSearchAttribute     = temporal.NewSearchAttributeKeyInt64(intSearchAttributeName)
)

const (
	activityStartToCloseTimeout = 30 * time.Second
	activityHeartbeatTimeout    = 5 * time.Second

	// The heartbeat-timeout activity heartbeats once and then sleeps past this
	// timeout, so the server times it out and records the last heartbeat details.
	// The sleep only has to outlast the timeout plus the server's detection
	// slack. It holds a worker activity slot for its whole duration — unlike a
	// workflow waiting on a timer, which holds nothing — so it is no longer than
	// it needs to be.
	shortHeartbeatTimeout         = 500 * time.Millisecond
	heartbeatTimeoutActivitySleep = 1500 * time.Millisecond

	// How long to let the cancel activity run before cancelling it, so that it
	// has started and heartbeated first. Under heavy load an activity may not be
	// dispatched inside this window, in which case it is cancelled before it
	// starts and records no canceled details; raise this if that shows up.
	cancelActivityLeadTime = 500 * time.Millisecond

	// The retry chain exists for ContinuedFailure, not for the backoff. Left
	// unset the SDK defaults to a one second initial interval, which would be a
	// large share of an iteration for a wait that buys nothing.
	retryInitialInterval = 100 * time.Millisecond

	localActivityStartToCloseTimeout = 5 * time.Second
	nexusScheduleToCloseTimeout      = time.Minute
	childWorkflowExecutionTimeout    = 2 * time.Minute
)

// makeFiller builds the padding carried by every payload body. strings.Repeat is
// deterministic, so a workflow may call it directly.
func makeFiller(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("x", n)
}

// CoverageInput is the workflow input, and is also the continue-as-new argument.
//
// The counts on it are the per-iteration fan-out. Raising one fattens a single
// RespondWorkflowTaskCompleted rather than adding requests: the workflow issues
// every command it can before blocking on any of them, so the SDK batches them
// into one workflow task completion. They are separate counts rather than one
// multiplier because the paths are not interchangeable — memo fields are
// map<string, Payload>, one payload per field, while activity input is a single
// Payloads holding many.
//
// The driver fills these in from its config and normalizes them first, so they
// arrive already defaulted; withDefaults is a guard for a replayed or
// hand-written input, not the primary path.
type CoverageInput struct {
	// Message is the payload body carried through every step of the iteration.
	Message          string `json:"message"`
	Iteration        int64  `json:"iteration"`
	TargetWorkflowID string `json:"targetWorkflowId"`
	NexusEndpoint    string `json:"nexusEndpoint"`
	// Filler is the padding every payload body carries, already materialized by
	// the driver. The byte count that produced it is config the workflow has no
	// use for, so only the string travels.
	Filler string `json:"filler,omitempty"`

	// MemoEntries is how many fields each upserted memo carries.
	MemoEntries int `json:"memoEntries"`
	// ActivityCount fans out the echo activity and the failing activity.
	ActivityCount int `json:"activityCount"`
	// MarkerCount fans out side effects and failing local activities.
	MarkerCount int `json:"markerCount"`
	// ChildCount fans out the echo child workflow. Each child is a real
	// execution, so this is the expensive knob; the failing child stays at one.
	ChildCount int `json:"childCount"`
	// SignalCount is how many signals reach the target workflow, which waits for
	// exactly this many before completing.
	SignalCount int `json:"signalCount"`
	// NexusCount fans out the Nexus operation, when an endpoint is configured.
	NexusCount int `json:"nexusCount"`
	// FailureDepth is how many levels deep a failure's cause chain goes. Each
	// level is its own Failure proto with its own details and, because the
	// failure converter sets EncodeCommonAttributes, its own encoded_attributes.
	// Depth therefore adds proto nodes rather than only enlarging one payload.
	FailureDepth int `json:"failureDepth"`
}

// withDefaults clamps every count to at least one, so a zero never silently
// turns a path off. It is pure, so a workflow may call it.
func (c CoverageInput) withDefaults() CoverageInput {
	for _, count := range []*int{
		&c.MemoEntries, &c.ActivityCount, &c.MarkerCount, &c.ChildCount,
		&c.SignalCount, &c.NexusCount, &c.FailureDepth,
	} {
		if *count < 1 {
			*count = 1
		}
	}
	return c
}

// CoverageOutput is the workflow output, and the child workflow's output.
type CoverageOutput struct {
	Message string `json:"message"`
}

// ChildInput is the child workflow input.
type ChildInput struct {
	Message string `json:"message"`
	Filler  string `json:"filler,omitempty"`
}

// FailingInput is the input to the workflow that always fails.
type FailingInput struct {
	Message      string `json:"message"`
	Filler       string `json:"filler,omitempty"`
	FailureDepth int    `json:"failureDepth,omitempty"`
}

// TargetInput is the input to the workflow the iteration messages from outside.
type TargetInput struct {
	// ExpectedSignals is how many signals the workflow waits for before
	// completing. The coverage workflow sends exactly this many.
	ExpectedSignals int `json:"expectedSignals"`
}

// SignalInput is the signal payload sent to another workflow.
type SignalInput struct {
	Step    string `json:"step"`
	Message string `json:"message"`
	Filler  string `json:"filler,omitempty"`
}

// UpdateInput and UpdateResult are the update payloads. Reject drives the
// validator into rejecting, which produces a failure payload instead of a result.
type UpdateInput struct {
	Step         string `json:"step"`
	Message      string `json:"message"`
	Reject       bool   `json:"reject"`
	Filler       string `json:"filler,omitempty"`
	FailureDepth int    `json:"failureDepth,omitempty"`
}

type UpdateResult struct {
	Step   string `json:"step"`
	Echoed string `json:"echoed"`
	Filler string `json:"filler,omitempty"`
}

// RetryInput and RetryResult belong to RetryWorkflow.
type RetryInput struct {
	Message      string `json:"message"`
	Filler       string `json:"filler,omitempty"`
	FailureDepth int    `json:"failureDepth,omitempty"`
}

type RetryResult struct {
	Message   string `json:"message"`
	LastError string `json:"lastError,omitempty"`
}

// ActivityInput, ActivityResult, and the detail types are what the activities
// exchange; every one of them ends up as an encrypted payload.
type ActivityInput struct {
	Step         string `json:"step"`
	Message      string `json:"message"`
	Filler       string `json:"filler,omitempty"`
	FailureDepth int    `json:"failureDepth,omitempty"`
}

type ActivityResult struct {
	Step   string `json:"step"`
	Echoed string `json:"echoed"`
	Filler string `json:"filler,omitempty"`
}

type HeartbeatDetails struct {
	Step    string `json:"step"`
	Attempt int32  `json:"attempt"`
	Message string `json:"message"`
	Filler  string `json:"filler,omitempty"`
}

type FailureDetails struct {
	Step    string               `json:"step"`
	Message string               `json:"message"`
	Filler  string               `json:"filler,omitempty"`
	Level   int                  `json:"level,omitempty"`
	Nested  NestedFailureDetails `json:"nested"`
}

type NestedFailureDetails struct {
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// MarkerData is what SideEffect records into a marker's details.
type MarkerData struct {
	Step    string `json:"step"`
	Message string `json:"message"`
	Filler  string `json:"filler,omitempty"`
}

// NexusInput and NexusOutput are the Nexus operation's payloads.
type NexusInput struct {
	Message string `json:"message"`
	Filler  string `json:"filler,omitempty"`
}

type NexusOutput struct {
	Echoed string `json:"echoed"`
}

// newFailureChain builds a cause chain depth levels deep. Level 1 is the
// outermost cause and level depth the innermost, so the numbering reads
// outward-in as a reader unwraps.
func newFailureChain(step, message, filler string, depth int) error {
	if depth < 1 {
		depth = 1
	}
	var cause error
	for level := depth; level >= 1; level-- {
		cause = temporal.NewApplicationErrorWithCause(
			fmt.Sprintf("cause level %d of the failure", level),
			nestedFailureErrorType,
			cause,
			FailureDetails{
				Step:    step,
				Message: message,
				Filler:  filler,
				Level:   level,
				Nested:  NestedFailureDetails{Reason: "cause detail", Message: message},
			},
		)
	}
	return cause
}

// CoverageWorkflow walks every payload path this project covers, then
// continues-as-new into a run that does nothing but complete, so that both the
// continue-as-new and the complete-workflow payloads appear.
//
// It runs in three phases. Phase A issues the commands that never block, phase B
// starts everything that does without waiting on it, and phase C waits. The
// split is what produces a fat RespondWorkflowTaskCompleted: every command from
// A and B lands in the first workflow task completion, because the workflow does
// not yield until C. It also runs the steps concurrently rather than in
// sequence, which is most of why an iteration is quicker than the sum of its
// waits.
func CoverageWorkflow(ctx workflow.Context, input CoverageInput) (CoverageOutput, error) {
	// Check if coming from continue-as-new. Complete the workflow if so.
	if workflow.GetInfo(ctx).ContinuedExecutionRunID != "" {
		return CoverageOutput{Message: input.Message}, nil
	}

	input = input.withDefaults()
	filler := input.Filler

	// ---- Phase A: commands that do not block ----

	memoFields := make(map[string]any, input.MemoEntries)
	for i := 0; i < input.MemoEntries; i++ {
		memoFields[fmt.Sprintf("encryptionUpsertedMemo-%d", i)] = MarkerData{
			Step: "upsert-memo", Message: input.Message, Filler: filler,
		}
	}
	if err := workflow.UpsertMemo(ctx, memoFields); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to upsert memo: %w", err)
	}

	if err := workflow.UpsertTypedSearchAttributes(
		ctx,
		keywordSearchAttribute.ValueSet(input.Message),
		intSearchAttribute.ValueSet(input.Iteration),
	); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to upsert search attributes: %w", err)
	}

	for i := 0; i < input.MarkerCount; i++ {
		step := fmt.Sprintf("side-effect-%d", i)
		var sideEffect MarkerData
		if err := workflow.SideEffect(
			ctx,
			func(workflow.Context) any {
				return MarkerData{Step: step, Message: input.Message, Filler: filler}
			},
		).Get(&sideEffect); err != nil {
			return CoverageOutput{}, fmt.Errorf("failed to read side effect: %w", err)
		}
	}

	// Version markers. Recorded by the SDK for replay rather than by user code,
	// and never routed through the data converter.
	_ = workflow.GetVersion(ctx, versionMarkerChangeID, workflow.DefaultVersion, 1)
	_ = workflow.GetVersion(ctx, otherVersionMarkerChangeID, workflow.DefaultVersion, 1)

	// ---- Phase B: start everything, wait for nothing ----

	// A failing local activity: its LocalActivity marker carries both details and
	// a Failure, which no other marker in the Go SDK does.
	localCtx := workflow.WithLocalActivityOptions(
		ctx,
		workflow.LocalActivityOptions{
			StartToCloseTimeout: localActivityStartToCloseTimeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		},
	)
	localFutures := make([]workflow.Future, 0, input.MarkerCount)
	for i := 0; i < input.MarkerCount; i++ {
		localFutures = append(localFutures, workflow.ExecuteLocalActivity(
			localCtx,
			FailingLocalActivity,
			ActivityInput{
				Step:         fmt.Sprintf("failing-local-activity-%d", i),
				Message:      input.Message,
				Filler:       filler,
				FailureDepth: input.FailureDepth,
			},
		))
	}

	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    activityHeartbeatTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	echoFutures := make([]workflow.Future, 0, input.ActivityCount)
	failFutures := make([]workflow.Future, 0, input.ActivityCount)
	for i := 0; i < input.ActivityCount; i++ {
		echoFutures = append(echoFutures, workflow.ExecuteActivity(
			activityCtx,
			echoActivityName,
			ActivityInput{Step: fmt.Sprintf("echo-%d", i), Message: input.Message, Filler: filler},
		))
		failFutures = append(failFutures, workflow.ExecuteActivity(
			activityCtx,
			failActivityName,
			ActivityInput{
				Step:         fmt.Sprintf("fail-%d", i),
				Message:      input.Message,
				Filler:       filler,
				FailureDepth: input.FailureDepth,
			},
		))
	}

	// Last heartbeat details, via a heartbeat timeout. Not fanned out: it holds a
	// worker activity slot for the whole of its sleep.
	timeoutCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    shortHeartbeatTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})
	heartbeatTimeoutFuture := workflow.ExecuteActivity(
		timeoutCtx,
		heartbeatTimeoutActivityName,
		ActivityInput{Step: "heartbeat-timeout", Message: input.Message, Filler: filler},
	)

	// Cancellation details. WaitForCancellation makes the workflow wait for the
	// activity's canceled error, which is what carries the details into history.
	cancelCtx, cancelActivity := workflow.WithCancel(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    activityHeartbeatTimeout,
		WaitForCancellation: true,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}))
	cancelFuture := workflow.ExecuteActivity(cancelCtx, cancelActivityName,
		ActivityInput{
			Step:         "cancel",
			Message:      input.Message,
			Filler:       filler,
			FailureDepth: input.FailureDepth,
		})

	parentID := workflow.GetInfo(ctx).WorkflowExecution.ID
	childFutures := make([]workflow.ChildWorkflowFuture, 0, input.ChildCount)
	for i := 0; i < input.ChildCount; i++ {
		childCtx := workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				WorkflowID:               fmt.Sprintf("%s-child-%d", parentID, i),
				WorkflowExecutionTimeout: childWorkflowExecutionTimeout,
				Memo:                     childMemo(input.MemoEntries, input.Message, filler),
				TypedSearchAttributes: temporal.NewSearchAttributes(
					keywordSearchAttribute.ValueSet(input.Message),
				),
			},
		)
		childFutures = append(childFutures, workflow.ExecuteChildWorkflow(
			childCtx,
			echoChildWorkflowName,
			ChildInput{Message: input.Message, Filler: filler},
		))
	}

	// A child workflow that fails, so the fail-workflow payloads appear in a
	// history this workflow is responsible for as well as in the standalone one
	// the driver starts. One is enough: it is a distinct shape, not a density
	// knob, and every child is a real execution.
	failingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               parentID + "-failing-child",
		WorkflowExecutionTimeout: childWorkflowExecutionTimeout,
	})
	failingChildFuture := workflow.ExecuteChildWorkflow(
		failingChildCtx,
		failingWorkflowName,
		FailingInput{Message: input.Message, Filler: filler, FailureDepth: input.FailureDepth},
	)

	// Signals to an external workflow: input and header. The target waits for
	// exactly SignalCount of them.
	signalFutures := make([]workflow.Future, 0, input.SignalCount)
	for i := 0; i < input.SignalCount; i++ {
		signalFutures = append(signalFutures, workflow.SignalExternalWorkflow(
			ctx,
			input.TargetWorkflowID,
			"",
			coverageSignalName,
			SignalInput{Step: fmt.Sprintf("signal-external-%d", i), Message: input.Message, Filler: filler},
		))
	}

	// Nexus is optional. A run that names no endpoint generates no Nexus load
	// rather than failing every iteration on an operation it cannot route.
	nexusFutures := make([]workflow.Future, 0, input.NexusCount)
	if input.NexusEndpoint != "" {
		nexusClient := workflow.NewNexusClient(input.NexusEndpoint, nexusServiceName)
		for i := 0; i < input.NexusCount; i++ {
			nexusFutures = append(nexusFutures, nexusClient.ExecuteOperation(
				ctx,
				echoOperation,
				NexusInput{Message: input.Message, Filler: filler},
				workflow.NexusOperationOptions{ScheduleToCloseTimeout: nexusScheduleToCloseTimeout},
			))
		}
	}

	// ---- Phase C: wait ----

	// Cancelling is a wait of its own: the activity has to have started and
	// heartbeated for its canceled details to mean anything.
	if err := workflow.Sleep(ctx, cancelActivityLeadTime); err != nil {
		cancelActivity()
		return CoverageOutput{}, fmt.Errorf("failed waiting before cancelling: %w", err)
	}
	cancelActivity()

	for i, future := range echoFutures {
		var echoed ActivityResult
		if err := future.Get(ctx, &echoed); err != nil {
			return CoverageOutput{}, fmt.Errorf("echo activity %d failed: %w", i, err)
		}
	}
	for i, future := range childFutures {
		var childResult CoverageOutput
		if err := future.Get(ctx, &childResult); err != nil {
			return CoverageOutput{}, fmt.Errorf("child workflow %d failed: %w", i, err)
		}
	}
	for i, future := range signalFutures {
		if err := future.Get(ctx, nil); err != nil {
			return CoverageOutput{}, fmt.Errorf("failed to signal external workflow (%d): %w", i, err)
		}
	}
	for i, future := range nexusFutures {
		var output NexusOutput
		if err := future.Get(ctx, &output); err != nil {
			return CoverageOutput{}, fmt.Errorf("nexus operation %d failed: %w", i, err)
		}
	}

	// These are supposed to fail. Their errors are the payloads they exist to
	// produce, so they are awaited and dropped rather than checked; that they
	// really do fail is covered by the unit tests.
	for _, future := range localFutures {
		_ = future.Get(ctx, nil)
	}
	for _, future := range failFutures {
		_ = future.Get(ctx, nil)
	}
	_ = heartbeatTimeoutFuture.Get(ctx, nil)
	_ = cancelFuture.Get(ctx, nil)
	_ = failingChildFuture.Get(ctx, nil)

	// Continue-as-new
	return CoverageOutput{}, workflow.NewContinueAsNewError(ctx, coverageWorkflowName, input)
}

// childMemo builds the memo a child workflow is started with, at the same field
// count as the parent's upsert.
func childMemo(entries int, message, filler string) map[string]any {
	memo := make(map[string]any, entries)
	for i := 0; i < entries; i++ {
		memo[fmt.Sprintf("encryptionChildMemo-%d", i)] = MarkerData{
			Step: "child", Message: message, Filler: filler,
		}
	}
	return memo
}

// EchoChildWorkflow completes with a result derived from its input.
func EchoChildWorkflow(_ workflow.Context, input ChildInput) (CoverageOutput, error) {
	return CoverageOutput{Message: input.Message}, nil
}

// FailingWorkflow always fails, with details and a cause chain that carries
// details of its own. With EncodeCommonAttributes set on the failure converter,
// the message and stack trace are encoded too.
func FailingWorkflow(_ workflow.Context, input FailingInput) (CoverageOutput, error) {
	return CoverageOutput{}, temporal.NewNonRetryableApplicationError(
		"workflow failed on purpose",
		workflowFailureErrorType,
		newFailureChain("fail-workflow", input.Message, input.Filler, input.FailureDepth),
		FailureDetails{
			Step:    "fail-workflow",
			Message: input.Message,
			Filler:  input.Filler,
			Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
		},
	)
}

// TargetWorkflow is what the iteration sends messages to from outside: updates,
// then signals from the coverage workflow. It handles both because both are
// external message paths, and one waiting workflow can serve both.
func TargetWorkflow(ctx workflow.Context, input TargetInput) (SignalInput, error) {
	// An update carries three payloads of its own: the argument, the result, and —
	// when the validator rejects it — a failure, which goes through the failure
	// converter rather than the data converter.
	if err := workflow.SetUpdateHandlerWithOptions(ctx, coverageUpdateName,
		func(_ workflow.Context, input UpdateInput) (UpdateResult, error) {
			return UpdateResult{Step: input.Step, Echoed: input.Message, Filler: input.Filler}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(_ workflow.Context, input UpdateInput) error {
				if input.Reject {
					return temporal.NewApplicationErrorWithCause(
						"update rejected on purpose",
						updateRejectedErrorType,
						newFailureChain(input.Step, input.Message, input.Filler, input.FailureDepth),
						FailureDetails{
							Step:    input.Step,
							Message: input.Message,
							Filler:  input.Filler,
							Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
						},
					)
				}
				return nil
			},
		}); err != nil {
		return SignalInput{}, fmt.Errorf("failed to register update handler: %w", err)
	}

	expected := input.ExpectedSignals
	if expected < 1 {
		expected = 1
	}
	channel := workflow.GetSignalChannel(ctx, coverageSignalName)
	var received SignalInput
	for i := 0; i < expected; i++ {
		channel.Receive(ctx, &received)
	}
	return received, nil
}

// RetryWorkflow reaches ContinuedFailure, which nothing else here produces.
// The Go continue-as-new command sets no last-run fields, but the server does
// populate them across a retry chain: this workflow's first attempt fails, and the
// server puts that failure on the second attempt's started event, where the SDK
// decodes it through the failure converter and hands it back as GetLastError.
//
// The error must be retryable for there to be a second attempt.
func RetryWorkflow(ctx workflow.Context, input RetryInput) (RetryResult, error) {
	lastError := workflow.GetLastError(ctx)
	if lastError == nil {
		return RetryResult{}, temporal.NewApplicationErrorWithCause(
			"first attempt failed on purpose",
			retryAttemptFailureErrorType,
			newFailureChain("retry-chain", input.Message, input.Filler, input.FailureDepth),
			FailureDetails{
				Step:    "retry-chain",
				Message: input.Message,
				Filler:  input.Filler,
				Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
			},
		)
	}
	return RetryResult{Message: input.Message, LastError: lastError.Error()}, nil
}
