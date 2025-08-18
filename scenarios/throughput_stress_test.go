package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/loadgen"
)

func TestThroughputStress(t *testing.T) {
	t.Parallel()

	env := loadgen.SetupTestEnvironment(t)
	executor := &tpsExecutor{
		state: &tpsState{},
	}

	scenarioInfo := loadgen.ScenarioInfo{
		ScenarioName: "throughput_stress_test",
		RunID:        fmt.Sprintf("test-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		ScenarioOptions: map[string]string{
			IterFlag:                   "4",
			ContinueAsNewAfterIterFlag: "2",
		},
	}

	err := env.RunExecutorTest(t, executor, scenarioInfo, cmdoptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	state := executor.Snapshot().(tpsState)
	require.Equal(t, state.CompletedIterations, 2)
}
