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

	scenarioName := "throughput_stress_test"
	runID := fmt.Sprintf("tps-%d", time.Now().Unix())
	taskQueueName := loadgen.TaskQueueForRun(scenarioName, runID)

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(2*time.Minute),
		workers.WithNexusEndpoint(taskQueueName),
	)

	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName: scenarioName,
		RunID:        runID,
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
		ScenarioOptions: map[string]string{
			IterFlag:                          "2",
			ContinueAsNewAfterIterFlag:        "1",
			NexusEndpointFlag:                 env.NexusEndpointName(),
			SleepTimeFlag:                     "1ms", // reduce to safe time
			VisibilityVerificationTimeoutFlag: "10s", // lower timeout to fail fast
		},
	}

	executor := &tpsExecutor{state: &tpsState{}}
	err := env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	state := executor.Snapshot().(tpsState)
	require.Equal(t, state.CompletedIterations, 1)
}
