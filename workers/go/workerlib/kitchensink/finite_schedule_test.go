package kitchensink

import (
	"testing"

	"github.com/stretchr/testify/require"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
)

func TestCreateFiniteScheduleRejectsInvalidID(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestActivityEnvironment()
	activities := &ClientActivities{}
	env.RegisterActivity(activities.CreateFiniteSchedule)
	_, err := env.ExecuteActivity(activities.CreateFiniteSchedule, ks.FiniteScheduleInput{ScheduleID: "unowned"})
	var applicationError *temporal.ApplicationError
	require.ErrorAs(t, err, &applicationError)
	require.Equal(t, "InvalidFiniteScheduleInput", applicationError.Type())
	require.True(t, applicationError.NonRetryable())
}
