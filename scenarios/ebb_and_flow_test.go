package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

func TestEbbAndFlow(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(2*time.Minute))

	sleepActivityJson := `{
		"count": {
			"type": "fixed",
			"value": "5"
		},
		"groups": {
			"group_a": {
				"weight": 1,
				"sleepDuration": {
					"type": "fixed",
					"value": "1ms"
				},
				"fairnessKeys": {
					"type": "fixed",
					"value": "1"
				}
			},
			"group_b": {
				"weight": 1,
				"sleepDuration": {
					"type": "fixed",
					"value": "1ms"
				},
				"fairnessKeys": {
					"type": "fixed",
					"value": "2"
				}
			}
		}
	}`

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("eaf-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Duration: 10 * time.Second,
		},
		ScenarioOptions: map[string]string{
			MinBacklogFlag:                    "0",
			MaxBacklogFlag:                    "1",
			PhaseTimeFlag:                     "5s",
			FairnessReportIntervalFlag:        "5s",
			BacklogLogIntervalFlag:            "5s",
			VisibilityVerificationTimeoutFlag: "5s",
			SleepActivityJsonFlag:             sleepActivityJson,
		},
	}

	var state ebbAndFlowState

	t.Run("Run executor", func(t *testing.T) {
		executor := &ebbAndFlowExecutor{}

		res, err := env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state = executor.Snapshot().(ebbAndFlowState)
		require.GreaterOrEqual(t, state.TotalCompletedWorkflows, int64(1))

		fairnessStatusLogs := res.ObservedLogs.FilterMessageSnippet("Fairness status").All()
		require.GreaterOrEqual(t, len(fairnessStatusLogs), 1, "Fairness status logs should be present")
	})

	t.Run("Run executor again, resuming from previous state", func(t *testing.T) {
		executor := &ebbAndFlowExecutor{}

		// Load previous state
		previouState := state
		err := executor.LoadState(func(v any) error {
			s := v.(*ebbAndFlowState)
			*s = state
			return nil
		})
		require.NoError(t, err, "Should be able to load previous state")

		resumeScenarioInfo := scenarioInfo

		res, err := env.RunExecutorTest(t, executor, resumeScenarioInfo, cmdoptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state = executor.Snapshot().(ebbAndFlowState)
		require.Greater(t, state.TotalCompletedWorkflows, previouState.TotalCompletedWorkflows)

		fairnessStatusLogs := res.ObservedLogs.FilterMessageSnippet("Fairness status").All()
		require.GreaterOrEqual(t, len(fairnessStatusLogs), 1, "Fairness status logs should be present")
	})

	t.Run("Run executor again, resuming from previous state but without any time left", func(t *testing.T) {
		executor := &ebbAndFlowExecutor{}

		// Load previous state
		err := executor.LoadState(func(v any) error {
			s := v.(*ebbAndFlowState)
			*s = state
			return nil
		})
		require.NoError(t, err, "Should be able to load previous state")

		resumeScenarioInfo := scenarioInfo
		resumeScenarioInfo.Configuration.Duration = 0 // ie no more time left

		_, err = env.RunExecutorTest(t, executor, resumeScenarioInfo, cmdoptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")
	})
}
