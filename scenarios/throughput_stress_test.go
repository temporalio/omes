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

func TestThroughputStress(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-%d", time.Now().Unix())
	taskQueueName := loadgen.TaskQueueForRun(runID)

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(1*time.Minute),
		workers.WithNexusEndpoint(taskQueueName))

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

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
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

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
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

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully when resuming from end")
	})
}
