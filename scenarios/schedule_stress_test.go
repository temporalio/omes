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

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		// Verify that schedules were created
		require.Equal(t, 3, len(executor.schedulesCreated), "Should have created 3 schedules")
	})
}
