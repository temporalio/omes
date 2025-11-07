package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
	"go.temporal.io/api/enums/v1"
	"go.uber.org/zap/zaptest"
)

func TestVersioningPinnedWorkflows(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("vpw-%d", time.Now().Unix())

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(2*time.Minute))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID:       runID,
		ExecutionID: "test-exec-id",
		Configuration: loadgen.RunConfiguration{
			Iterations: 12, // 0 (start) + 11 iterations, will bump versions at 5 and 10
		},
		ScenarioOptions: map[string]string{
			NumWorkflowsFlag:        "3", // Start 3 workflows
			VersionBumpIntervalFlag: "5", // Bump every 5 iterations
			InitialVersionFlag:      "1", // Start with version 1
		},
	}

	t.Run("Run executor", func(t *testing.T) {
		executor := newVersioningPinnedExecutor()

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		executor.lock.Lock()
		state := *executor.state
		executor.lock.Unlock()

		require.Len(t, state.WorkflowIDs, 3, "Should have started 3 workflows")
		require.Equal(t, "3", state.CurrentVersion, "Should have bumped to 3 (1->2 at iter 5, 2->3 at iter 10)")
		require.Equal(t, []string{"1", "2", "3"}, state.VersionSequence, "Should track all versions")
	})

	t.Run("Verify checks build ID progression", func(t *testing.T) {
		executor := newVersioningPinnedExecutor()

		// Run a simple scenario
		shortScenarioInfo := loadgen.ScenarioInfo{
			RunID:       fmt.Sprintf("vpw-verify-%d", time.Now().Unix()),
			ExecutionID: "test-verify-exec-id",
			Configuration: loadgen.RunConfiguration{
				Iterations: 7, // 0 (start) + 6 iterations, will bump at iteration 5
			},
			ScenarioOptions: map[string]string{
				NumWorkflowsFlag:        "2",
				VersionBumpIntervalFlag: "5",
				InitialVersionFlag:      "1", // Start with version 1
			},
		}

		_, err := env.RunExecutorTest(t, executor, shortScenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		// Now verify the workflows
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Create a new scenario info for verification with proper client setup
		verifyInfo := loadgen.ScenarioInfo{
			RunID:       shortScenarioInfo.RunID,
			ExecutionID: shortScenarioInfo.ExecutionID,
			Client:      env.TemporalClient(),
			Logger:      zaptest.NewLogger(t).Sugar(),
			Namespace:   "default",
		}

		errors := executor.Verify(ctx, verifyInfo)
		require.Empty(t, errors, "Verification should pass with no errors")

		executor.lock.Lock()
		state := *executor.state
		executor.lock.Unlock()

		require.Len(t, state.WorkflowIDs, 2)
		require.Equal(t, "2", state.CurrentVersion, "Should have bumped to 2")

		// Verify we can read the workflow histories
		for _, workflowID := range state.WorkflowIDs {
			iter := env.TemporalClient().GetWorkflowHistory(
				ctx,
				workflowID,
				"",
				false,
				enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
			)

			eventCount := 0
			for iter.HasNext() {
				_, err := iter.Next()
				require.NoError(t, err)
				eventCount++
			}
			require.Greater(t, eventCount, 0, "Should have history events for workflow %s", workflowID)
		}
	})
}
