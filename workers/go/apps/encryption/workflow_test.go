package encryption

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestFailingWorkflowFailsWithNestedDetails(t *testing.T) {
	// FailingWorkflow touches no workflow APIs, so it needs no test environment.
	_, err := FailingWorkflow(nil, FailingInput{Message: testMessage})
	require.Error(t, err, "FailingWorkflow succeeded; it exists to fail")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, workflowFailureErrorType, appErr.Type())

	var details FailureDetails
	require.NoError(t, appErr.Details(&details), "failed to read details")
	require.Equal(t, testMessage, details.Nested.Message)

	// The cause carries details of its own, which is what makes the encoded failure
	// nest rather than being one flat failure.
	var causeErr *temporal.ApplicationError
	require.ErrorAs(t, appErr.Unwrap(), &causeErr)
	require.Equal(t, nestedFailureErrorType, causeErr.Type())
	require.True(t, causeErr.HasDetails(), "cause carries no details")
}

func TestFailingLocalActivityFails(t *testing.T) {
	_, err := FailingLocalActivity(nil, ActivityInput{Step: "failing-local-activity", Message: testMessage})
	require.Error(t, err, "FailingLocalActivity succeeded; it exists to fail")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, localActivityFailureType, appErr.Type())
}

func TestFailActivityFailsWithNestedDetails(t *testing.T) {
	// This one heartbeats, so it needs an activity context.
	env := (&testsuite.WorkflowTestSuite{}).NewTestActivityEnvironment()
	env.RegisterActivity(FailActivity)

	_, err := env.ExecuteActivity(FailActivity, ActivityInput{Step: "fail", Message: testMessage})
	require.Error(t, err, "FailActivity succeeded; it exists to fail")

	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, activityFailureErrorType, appErr.Type())

	var causeErr *temporal.ApplicationError
	require.ErrorAs(t, appErr.Unwrap(), &causeErr)
	require.Equal(t, nestedFailureErrorType, causeErr.Type())
}

func TestFailingWorkflowConvertsThroughTheRealConverters(t *testing.T) {
	dataConverter, err := newDataConverter()
	require.NoError(t, err)

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

	require.True(t, env.IsWorkflowCompleted(), "workflow did not complete")
	err = env.GetWorkflowError()
	require.Error(t, err, "FailingWorkflow succeeded; it exists to fail")

	// The failure survives a round trip through the encrypting converters.
	var appErr *temporal.ApplicationError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, workflowFailureErrorType, appErr.Type())

	var details FailureDetails
	require.NoError(t, appErr.Details(&details), "failed to read details back through the codec")
	require.Equal(t, testMessage, details.Nested.Message)
}
