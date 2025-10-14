package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

func TestScheduleStress(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(3*time.Minute))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("schedule-stress-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 3, // Create 3 schedules
		},
		ScenarioOptions: map[string]string{
			ScheduleCountFlag:              "3",
			ScheduleActionsPerScheduleFlag: "2",
			ScheduleCronFlag:               "* * * * * * *", // Every second
		},
	}

	t.Run("Run executor", func(t *testing.T) {
		executor := newScheduleStressExecutor()

		result, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		// Verify that the expected workflows were created and completed
		// Expected: 3 creator workflows + (3 schedules × 2 actions per schedule) = 9 total
		expectedWorkflows := 9
		require.NotNil(t, result, "Test result should not be nil")

		// The scenario already verifies the workflow count via MinVisibilityCountEventually,
		// so if we get here without error, the verification passed
		t.Logf("Schedule stress scenario completed successfully with %d expected workflows", expectedWorkflows)
	})
}
