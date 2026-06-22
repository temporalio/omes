package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
)

func TestThroughputStress(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-%d", time.Now().Unix())

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute),
		workertest.WithNexusEndpoint(runID))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: runID,
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		ScenarioOptions: map[string]string{
			IterFlag:                          "2",
			ContinueAsNewAfterIterFlag:        "1",
			NexusEndpointFlag:                 env.NexusEndpointName(),
			SleepTimeFlag:                     "1ms", // reduce to safe time
			VisibilityVerificationTimeoutFlag: "10s", // lower timeout to fail fast
		},
	}

	t.Run("Run executor", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state := executor.Snapshot().(tpsState)
		require.Equal(t, state.CompletedIterations, 2)
	})

	t.Run("Run executor again, resuming from middle", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		err := executor.LoadState(func(v any) error {
			s := v.(*tpsState)
			s.CompletedIterations = 0 // execution will start from iteration 1
			return nil
		})
		require.NoError(t, err)

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully when resuming from middle")
	})

	t.Run("Run executor again, resuming from end", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		err := executor.LoadState(func(v any) error {
			s := v.(*tpsState)
			s.CompletedIterations = s.CompletedIterations
			return nil
		})
		require.NoError(t, err)

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully when resuming from end")
	})
}

func TestThroughputStressScheduleConfig(t *testing.T) {
	t.Parallel()

	baseInfo := loadgen.ScenarioInfo{
		RunID:       "schedule-config",
		ExecutionID: "schedule-config-execution",
		ScenarioOptions: map[string]string{
			IncludeSchedulesFlag: "true",
		},
	}

	t.Run("disabled by default", func(t *testing.T) {
		info := baseInfo
		info.ScenarioOptions = nil
		executor := newThroughputStressExecutor()
		require.NoError(t, executor.Configure(info))
		require.False(t, executor.config.Schedules.Enabled)
		require.Equal(t, 0, executor.config.Schedules.IterationSchedulesPerIteration)
	})

	t.Run("valid enabled defaults", func(t *testing.T) {
		executor := newThroughputStressExecutor()
		require.NoError(t, executor.Configure(baseInfo))
		require.True(t, executor.config.Schedules.Enabled)
		require.Equal(t, 3, executor.config.Schedules.StableCount)
		require.Equal(t, 1, executor.config.Schedules.IterationSchedulesPerIteration)
		require.True(t, hasRequiredStablePolicies(executor.config.Schedules.OverlapPolicies))
	})

	t.Run("invalid overlap policy", func(t *testing.T) {
		info := baseInfo
		info.ScenarioOptions = map[string]string{
			IncludeSchedulesFlag:        "true",
			ScheduleOverlapPoliciesFlag: "skip,buffer_one",
		}
		executor := newThroughputStressExecutor()
		require.ErrorContains(t, executor.Configure(info), ScheduleOverlapPoliciesFlag)
	})

	t.Run("invalid completion mode", func(t *testing.T) {
		info := baseInfo
		info.ScenarioOptions = map[string]string{
			IncludeSchedulesFlag:             "true",
			StableScheduleCompletionModeFlag: "wait-forever",
		}
		executor := newThroughputStressExecutor()
		require.ErrorContains(t, executor.Configure(info), StableScheduleCompletionModeFlag)
	})

	t.Run("invalid schedule duration", func(t *testing.T) {
		info := baseInfo
		info.ScenarioOptions = map[string]string{
			IncludeSchedulesFlag:               "true",
			StableScheduleWorkflowDurationFlag: "0s",
		}
		executor := newThroughputStressExecutor()
		require.ErrorContains(t, executor.Configure(info), StableScheduleWorkflowDurationFlag)
	})

	t.Run("invalid schedule interval", func(t *testing.T) {
		info := baseInfo
		info.ScenarioOptions = map[string]string{
			IncludeSchedulesFlag:       "true",
			StableScheduleIntervalFlag: "500ms",
		}
		executor := newThroughputStressExecutor()
		require.ErrorContains(t, executor.Configure(info), StableScheduleIntervalFlag)
	})
}

func TestThroughputStressWithSchedules(t *testing.T) {
	runID := fmt.Sprintf("tps-schedules-%d", time.Now().Unix())

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(90*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID:       runID,
		ExecutionID: "schedule-execution",
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
		ScenarioOptions: map[string]string{
			IterFlag:                              "1",
			ContinueAsNewAfterIterFlag:            "0",
			SleepTimeFlag:                         "1ms",
			VisibilityVerificationTimeoutFlag:     "20s",
			IncludeSchedulesFlag:                  "true",
			StableScheduleIntervalFlag:            "1s",
			StableScheduleWorkflowDurationFlag:    "100ms",
			StableScheduleWindowFlag:              "4s",
			IterationScheduleWorkflowDurationFlag: "100ms",
			ScheduleVisibilityTimeoutFlag:         "20s",
			ScheduleAPIOperationIntervalFlag:      "1ms",
			IterationSchedulesPerIterationFlag:    "1",
			StableScheduleCompletionModeFlag:      ScheduleCompletionModeRelease,
			ScheduleOverlapPoliciesFlag:           "skip,buffer_one,buffer_all",
		},
	}

	executor := newThroughputStressExecutor()

	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully with schedules")

	state := executor.Snapshot().(tpsState)
	require.Equal(t, 1, state.CompletedIterations)
	require.True(t, state.StableSchedulesCreated)
	require.True(t, state.StableSchedulesFinalized)
	require.True(t, state.MatchingTimesVerified)
	require.Equal(t, 1, state.CompletedIterationScheduledWorkflows)
	require.GreaterOrEqual(t, state.CompletedStableScheduledWorkflows, 3)
}
