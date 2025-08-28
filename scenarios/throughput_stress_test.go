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

	env := workers.SetupTestEnvironment(t)
	nexusEndpointName, err := env.CreateNexusEndpoint(t.Context(), taskQueueName)
	require.NoError(t, err, "Failed to create Nexus endpoint")

	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName: scenarioName,
		RunID:        runID,
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
		ScenarioOptions: map[string]string{
			IterFlag:                          "4",
			ContinueAsNewAfterIterFlag:        "2",
			NexusEndpointFlag:                 nexusEndpointName,
			VisibilityVerificationTimeoutFlag: "10s", // lower timeout to fail fast
		},
	}

	executor := &tpsExecutor{state: &tpsState{}}
	err = env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	state := executor.Snapshot().(tpsState)
	require.Equal(t, state.CompletedIterations, 1)
}
