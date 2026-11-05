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

func TestBandwidthStressConfiguration(t *testing.T) {
	t.Parallel()

	t.Run("default incident payload", func(t *testing.T) {
		executor := newBandwidthStressExecutor(t, nil)
		actions := executor.testInput(1).GetWorkflowInput().GetInitialActions()[0].GetActions()

		require.Len(t, actions, 2)
		payload := actions[0].GetExecActivity().GetPayload()
		require.EqualValues(t, bandwidthDefaultPayloadSize, payload.GetBytesToReceive())
		require.EqualValues(t, bandwidthDefaultPayloadSize, payload.GetBytesToReturn())
		require.NotNil(t, actions[0].GetExecActivity().GetRemote())
		require.NotNil(t, actions[1].GetReturnResult())
	})

	t.Run("custom distribution and activity count", func(t *testing.T) {
		executor := newBandwidthStressExecutor(t, map[string]string{
			bandwidthActivitiesPerWorkflowFlag: "3",
			bandwidthPayloadDistributionFlag:   `{"size":{"type":"discrete","weights":{"1024":1,"2048":1}}}`,
		})
		actions := executor.testInput(10).GetWorkflowInput().GetInitialActions()[0].GetActions()

		require.Len(t, actions, 4)
		for _, action := range actions[:3] {
			payload := action.GetExecActivity().GetPayload()
			require.Contains(t, []int32{1024, 2048}, payload.GetBytesToReceive())
			require.Equal(t, payload.GetBytesToReceive(), payload.GetBytesToReturn())
		}
	})

	t.Run("invalid activity count", func(t *testing.T) {
		executor := &bandwidthStressExecutor{}
		err := executor.Configure(bandwidthStressScenarioInfo(map[string]string{
			bandwidthActivitiesPerWorkflowFlag: "0",
		}))
		require.ErrorContains(t, err, "activities-per-workflow must be positive")
	})

	t.Run("missing size distribution", func(t *testing.T) {
		executor := &bandwidthStressExecutor{}
		err := executor.Configure(bandwidthStressScenarioInfo(map[string]string{
			bandwidthPayloadDistributionFlag: `{}`,
		}))
		require.ErrorContains(t, err, "must configure a size distribution")
	})

	t.Run("run validates configuration", func(t *testing.T) {
		executor := &bandwidthStressExecutor{}
		err := executor.Run(t.Context(), bandwidthStressScenarioInfo(map[string]string{
			bandwidthActivitiesPerWorkflowFlag: "0",
		}))
		require.ErrorContains(t, err, "failed to parse scenario configuration")
		require.ErrorContains(t, err, "activities-per-workflow must be positive")
	})
}

func TestBandwidthStress(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	executor := loadgen.GetScenario("bandwidth_stress").ExecutorFn()
	info := loadgen.ScenarioInfo{
		ScenarioName: "bandwidth_stress",
		RunID:        fmt.Sprintf("bandwidth-%d", time.Now().UnixNano()),
		Configuration: loadgen.RunConfiguration{
			Iterations:    2,
			MaxConcurrent: 2,
		},
		Options: loadgen.MustResolveScenarioOptions("bandwidth_stress", map[string]string{
			bandwidthPayloadDistributionFlag: `{"size":{"type":"fixed","value":"1024"}}`,
		}),
	}
	_, err := env.RunExecutorTest(t, executor, info, clioptions.LangGo)
	require.NoError(t, err)
}

func newBandwidthStressExecutor(t *testing.T, provided map[string]string) *bandwidthStressExecutor {
	t.Helper()
	executor := &bandwidthStressExecutor{}
	require.NoError(t, executor.Configure(bandwidthStressScenarioInfo(provided)))
	return executor
}

func bandwidthStressScenarioInfo(provided map[string]string) loadgen.ScenarioInfo {
	return loadgen.ScenarioInfo{
		ScenarioName: "bandwidth_stress",
		RunID:        "bandwidth-test",
		Options:      loadgen.MustResolveScenarioOptions("bandwidth_stress", provided),
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
	}
}
