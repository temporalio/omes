package encryption

import (
	"fmt"
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

// Search attributes. The app does not register these: it targets cloud namespaces,
// where custom attributes are provisioned out of band rather than through the
// operator service. Both must already exist in the namespace, or the first upsert
// fails the workflow task.
//
// They carry no encryption load — the server indexes search attributes, so no SDK
// routes them through a payload codec — and exist only so the start-child and
// continue-as-new paths carry the same field shape a real workflow would.
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

	// The heartbeat-timeout activity heartbeats once and then sleeps well past this
	// timeout, so the server times it out and records the last heartbeat details.
	shortHeartbeatTimeout         = time.Second
	heartbeatTimeoutActivitySleep = 3 * time.Second

	// How long to let the cancel activity run before cancelling it, so that it has
	// started and heartbeated first.
	cancelActivityLeadTime = time.Second

	localActivityStartToCloseTimeout = 5 * time.Second
	nexusScheduleToCloseTimeout      = time.Minute
	childWorkflowExecutionTimeout    = 2 * time.Minute
)

// CoverageInput is the workflow input, and is also the continue-as-new argument.
type CoverageInput struct {
	// Message is the payload body carried through every step of the iteration.
	Message          string `json:"message"`
	Iteration        int64  `json:"iteration"`
	TargetWorkflowID string `json:"targetWorkflowId"`
	NexusEndpoint    string `json:"nexusEndpoint"`
}

// CoverageOutput is the workflow output, and the child workflow's output.
type CoverageOutput struct {
	Message string `json:"message"`
}

// ChildInput is the child workflow input.
type ChildInput struct {
	Message string `json:"message"`
}

// FailingInput is the input to the workflow that always fails.
type FailingInput struct {
	Message string `json:"message"`
}

// SignalInput is the signal payload sent to another workflow.
type SignalInput struct {
	Step    string `json:"step"`
	Message string `json:"message"`
}

// UpdateInput and UpdateResult are the update payloads. Reject drives the
// validator into rejecting, which produces a failure payload instead of a result.
type UpdateInput struct {
	Step    string `json:"step"`
	Message string `json:"message"`
	Reject  bool   `json:"reject"`
}

type UpdateResult struct {
	Step   string `json:"step"`
	Echoed string `json:"echoed"`
}

// RetryInput and RetryResult belong to RetryWorkflow.
type RetryInput struct {
	Message string `json:"message"`
}

type RetryResult struct {
	Message   string `json:"message"`
	LastError string `json:"lastError,omitempty"`
}

// ActivityInput, ActivityResult, and the detail types are what the activities
// exchange; every one of them ends up as an encrypted payload.
type ActivityInput struct {
	Step    string `json:"step"`
	Message string `json:"message"`
}

type ActivityResult struct {
	Step   string `json:"step"`
	Echoed string `json:"echoed"`
}

type HeartbeatDetails struct {
	Step    string `json:"step"`
	Attempt int32  `json:"attempt"`
	Message string `json:"message"`
}

type FailureDetails struct {
	Step    string               `json:"step"`
	Message string               `json:"message"`
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
}

// NexusInput and NexusOutput are the Nexus operation's payloads.
type NexusInput struct {
	Message string `json:"message"`
}

type NexusOutput struct {
	Echoed string `json:"echoed"`
}

// CoverageWorkflow walks every payload path this project covers, then
// continues-as-new into a run that does nothing but complete, so that both the
// continue-as-new and the complete-workflow payloads appear.
func CoverageWorkflow(ctx workflow.Context, input CoverageInput) (CoverageOutput, error) {
	// Check if coming from continue-as-new. Complete the workflow if so.
	if workflow.GetInfo(ctx).ContinuedExecutionRunID != "" {
		return CoverageOutput{Message: input.Message}, nil
	}

	if err := workflow.UpsertMemo(
		ctx,
		map[string]any{
			"encryptionUpsertedMemo": MarkerData{Step: "upsert-memo", Message: input.Message},
		},
	); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to upsert memo: %w", err)
	}

	if err := workflow.UpsertTypedSearchAttributes(
		ctx,
		keywordSearchAttribute.ValueSet(input.Message),
		intSearchAttribute.ValueSet(input.Iteration),
	); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to upsert search attributes: %w", err)
	}

	var sideEffect MarkerData
	if err := workflow.SideEffect(
		ctx,
		func(workflow.Context) any {
			return MarkerData{Step: "side-effect", Message: input.Message}
		},
	).Get(&sideEffect); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to read side effect: %w", err)
	}

	// A failing local activity: its LocalActivity marker carries both details and a
	// Failure, which no other marker in the Go SDK does.
	localCtx := workflow.WithLocalActivityOptions(
		ctx,
		workflow.LocalActivityOptions{
			StartToCloseTimeout: localActivityStartToCloseTimeout,
			RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
		},
	)

	// Not handling error since it's expected.
	_ = workflow.ExecuteLocalActivity(
		localCtx,
		FailingLocalActivity,
		ActivityInput{Step: "failing-local-activity", Message: input.Message},
	).Get(localCtx, nil)

	activityCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    activityHeartbeatTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	var echoed ActivityResult
	if err := workflow.ExecuteActivity(
		activityCtx,
		echoActivityName,
		ActivityInput{Step: "echo", Message: input.Message},
	).Get(ctx, &echoed); err != nil {
		return CoverageOutput{}, fmt.Errorf("echo activity failed: %w", err)
	}

	// Not handling error since it's expected.
	_ = workflow.ExecuteActivity(
		activityCtx,
		failActivityName,
		ActivityInput{Step: "fail", Message: input.Message},
	).Get(ctx, nil)

	// Last heartbeat details, via a heartbeat timeout.
	timeoutCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    shortHeartbeatTimeout,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	})

	// Not handling error since it's expected.
	_ = workflow.ExecuteActivity(
		timeoutCtx,
		heartbeatTimeoutActivityName,
		ActivityInput{Step: "heartbeat-timeout", Message: input.Message},
	).Get(ctx, nil)

	// Cancellation details. WaitForCancellation makes the workflow wait for the
	// activity's canceled error, which is what carries the details into history.
	cancelCtx, cancelActivity := workflow.WithCancel(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: activityStartToCloseTimeout,
		HeartbeatTimeout:    activityHeartbeatTimeout,
		WaitForCancellation: true,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}))
	cancelFuture := workflow.ExecuteActivity(cancelCtx, cancelActivityName,
		ActivityInput{Step: "cancel", Message: input.Message})
	if err := workflow.Sleep(ctx, cancelActivityLeadTime); err != nil {
		cancelActivity()
		return CoverageOutput{}, fmt.Errorf("failed waiting before cancelling: %w", err)
	}
	cancelActivity()

	// Not handling error since it's expected.
	cancelFuture.Get(ctx, nil)

	parentID := workflow.GetInfo(ctx).WorkflowExecution.ID
	childCtx := workflow.WithChildOptions(
		ctx,
		workflow.ChildWorkflowOptions{
			WorkflowID:               parentID + "-child",
			WorkflowExecutionTimeout: childWorkflowExecutionTimeout,
			Memo: map[string]any{
				"encryptionChildMemo": MarkerData{Step: "child", Message: input.Message},
			},
			TypedSearchAttributes: temporal.NewSearchAttributes(
				keywordSearchAttribute.ValueSet(input.Message),
			),
		},
	)
	var childResult CoverageOutput
	if err := workflow.ExecuteChildWorkflow(
		childCtx,
		echoChildWorkflowName,
		ChildInput{Message: input.Message},
	).Get(ctx, &childResult); err != nil {
		return CoverageOutput{}, fmt.Errorf("child workflow failed: %w", err)
	}

	// A child workflow that fails, so the fail-workflow payloads appear in a history
	// this workflow is responsible for as well as in the standalone one the driver
	// starts.
	failingChildCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
		WorkflowID:               parentID + "-failing-child",
		WorkflowExecutionTimeout: childWorkflowExecutionTimeout,
	})
	_ = workflow.ExecuteChildWorkflow(
		failingChildCtx,
		failingWorkflowName,
		FailingInput{Message: input.Message},
	).Get(ctx, nil)

	// Signal to an external workflow: input and header.
	if err := workflow.SignalExternalWorkflow(
		ctx,
		input.TargetWorkflowID,
		"",
		coverageSignalName,
		SignalInput{Step: "signal-external", Message: input.Message},
	).Get(ctx, nil); err != nil {
		return CoverageOutput{}, fmt.Errorf("failed to signal external workflow: %w", err)
	}

	// Nexus is optional. A run that names no endpoint generates no Nexus load rather
	// than failing every iteration on an operation it cannot route.
	if input.NexusEndpoint != "" {
		nexusClient := workflow.NewNexusClient(input.NexusEndpoint, nexusServiceName)
		var output NexusOutput
		if err := nexusClient.ExecuteOperation(
			ctx,
			echoOperation,
			NexusInput{Message: input.Message},
			workflow.NexusOperationOptions{ScheduleToCloseTimeout: nexusScheduleToCloseTimeout},
		).Get(ctx, &output); err != nil {
			return CoverageOutput{}, fmt.Errorf("nexus operation failed: %w", err)
		}
	}

	// Continue-as-new
	return CoverageOutput{}, workflow.NewContinueAsNewError(ctx, coverageWorkflowName, input)
}

// EchoChildWorkflow completes with a result derived from its input.
func EchoChildWorkflow(_ workflow.Context, input ChildInput) (CoverageOutput, error) {
	return CoverageOutput{Message: input.Message}, nil
}

// FailingWorkflow always fails, with details and a nested cause that carries
// details of its own. With EncodeCommonAttributes set on the failure converter,
// the message and stack trace are encoded too.
func FailingWorkflow(_ workflow.Context, input FailingInput) (CoverageOutput, error) {
	cause := temporal.NewApplicationError(
		"nested cause of the workflow failure",
		nestedFailureErrorType,
		NestedFailureDetails{Reason: "cause detail", Message: input.Message},
	)
	return CoverageOutput{}, temporal.NewNonRetryableApplicationError(
		"workflow failed on purpose",
		workflowFailureErrorType,
		cause,
		FailureDetails{
			Step:    "fail-workflow",
			Message: input.Message,
			Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
		},
	)
}

// TargetWorkflow is what the iteration sends messages to from outside: an update,
// then a signal from the coverage workflow. It handles both because both are
// external message paths, and one waiting workflow can serve both.
func TargetWorkflow(ctx workflow.Context) (SignalInput, error) {
	// An update carries three payloads of its own: the argument, the result, and —
	// when the validator rejects it — a failure, which goes through the failure
	// converter rather than the data converter.
	if err := workflow.SetUpdateHandlerWithOptions(ctx, coverageUpdateName,
		func(_ workflow.Context, input UpdateInput) (UpdateResult, error) {
			return UpdateResult{Step: input.Step, Echoed: input.Message}, nil
		},
		workflow.UpdateHandlerOptions{
			Validator: func(_ workflow.Context, input UpdateInput) error {
				if input.Reject {
					return temporal.NewApplicationError(
						"update rejected on purpose",
						updateRejectedErrorType,
						FailureDetails{
							Step:    input.Step,
							Message: input.Message,
							Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
						},
					)
				}
				return nil
			},
		}); err != nil {
		return SignalInput{}, fmt.Errorf("failed to register update handler: %w", err)
	}

	var received SignalInput
	workflow.GetSignalChannel(ctx, coverageSignalName).Receive(ctx, &received)
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
		return RetryResult{}, temporal.NewApplicationError(
			"first attempt failed on purpose",
			retryAttemptFailureErrorType,
			FailureDetails{
				Step:    "retry-chain",
				Message: input.Message,
				Nested:  NestedFailureDetails{Reason: "nested detail", Message: input.Message},
			},
		)
	}
	return RetryResult{Message: input.Message, LastError: lastError.Error()}, nil
}
