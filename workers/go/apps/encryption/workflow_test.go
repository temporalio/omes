package encryption

import (
	"errors"
	"testing"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestFailingWorkflowFailsWithNestedDetails(t *testing.T) {
	// FailingWorkflow touches no workflow APIs, so it needs no test environment.
	_, err := FailingWorkflow(nil, FailingInput{Message: testMessage})
	if err == nil {
		t.Fatal("FailingWorkflow succeeded; it exists to fail")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is %T, want *temporal.ApplicationError", err)
	}
	if appErr.Type() != workflowFailureErrorType {
		t.Errorf("error type = %q, want %q", appErr.Type(), workflowFailureErrorType)
	}

	var details FailureDetails
	if err := appErr.Details(&details); err != nil {
		t.Fatalf("failed to read details: %v", err)
	}
	if details.Nested.Message != testMessage {
		t.Errorf("nested detail message = %q, want %q", details.Nested.Message, testMessage)
	}

	// The cause carries details of its own, which is what makes the encoded failure
	// nest rather than being one flat failure.
	var causeErr *temporal.ApplicationError
	if !errors.As(appErr.Unwrap(), &causeErr) {
		t.Fatalf("cause is %T, want *temporal.ApplicationError", appErr.Unwrap())
	}
	if causeErr.Type() != nestedFailureErrorType {
		t.Errorf("cause type = %q, want %q", causeErr.Type(), nestedFailureErrorType)
	}
	if !causeErr.HasDetails() {
		t.Error("cause carries no details")
	}
}

func TestFailingLocalActivityFails(t *testing.T) {
	_, err := FailingLocalActivity(nil, ActivityInput{Step: "failing-local-activity", Message: testMessage})
	if err == nil {
		t.Fatal("FailingLocalActivity succeeded; it exists to fail")
	}
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is %T, want *temporal.ApplicationError", err)
	}
	if appErr.Type() != localActivityFailureType {
		t.Errorf("error type = %q, want %q", appErr.Type(), localActivityFailureType)
	}
}

func TestFailActivityFailsWithNestedDetails(t *testing.T) {
	// This one heartbeats, so it needs an activity context.
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(FailActivity)

	_, err := env.ExecuteActivity(FailActivity, ActivityInput{Step: "fail", Message: testMessage})
	if err == nil {
		t.Fatal("FailActivity succeeded; it exists to fail")
	}

	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is %T, want *temporal.ApplicationError", err)
	}
	if appErr.Type() != activityFailureErrorType {
		t.Errorf("error type = %q, want %q", appErr.Type(), activityFailureErrorType)
	}
	var causeErr *temporal.ApplicationError
	if !errors.As(appErr.Unwrap(), &causeErr) {
		t.Fatalf("cause is %T, want *temporal.ApplicationError", appErr.Unwrap())
	}
	if causeErr.Type() != nestedFailureErrorType {
		t.Errorf("cause type = %q, want %q", causeErr.Type(), nestedFailureErrorType)
	}
}

func TestFailingWorkflowConvertsThroughTheRealConverters(t *testing.T) {
	dataConverter, err := newDataConverter()
	if err != nil {
		t.Fatalf("newDataConverter failed: %v", err)
	}

	suite := &testsuite.WorkflowTestSuite{}
	suite.SetContextPropagators([]workflow.ContextPropagator{plaintextPropagator{}})
	env := suite.NewTestWorkflowEnvironment()
	env.SetDataConverter(dataConverter)
	env.SetFailureConverter(temporal.NewDefaultFailureConverter(temporal.DefaultFailureConverterOptions{
		DataConverter:          dataConverter,
		EncodeCommonAttributes: true,
	}))
	env.RegisterWorkflowWithOptions(FailingWorkflow, workflow.RegisterOptions{Name: failingWorkflowName})

	env.ExecuteWorkflow(failingWorkflowName, FailingInput{Message: testMessage})

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	err = env.GetWorkflowError()
	if err == nil {
		t.Fatal("FailingWorkflow succeeded; it exists to fail")
	}

	// The failure survives a round trip through the encrypting converters.
	var appErr *temporal.ApplicationError
	if !errors.As(err, &appErr) {
		t.Fatalf("error is %T, want *temporal.ApplicationError: %v", err, err)
	}
	if appErr.Type() != workflowFailureErrorType {
		t.Errorf("error type = %q, want %q", appErr.Type(), workflowFailureErrorType)
	}
	var details FailureDetails
	if err := appErr.Details(&details); err != nil {
		t.Fatalf("failed to read details back through the codec: %v", err)
	}
	if details.Nested.Message != testMessage {
		t.Errorf("nested detail message = %q, want %q", details.Nested.Message, testMessage)
	}
}
