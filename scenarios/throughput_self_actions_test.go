package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.uber.org/zap"
)

func TestThroughputOwningExecutionRunsOnGoWorker(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	for _, chunk := range []string{"0", "1"} {
		t.Run("continue_after_"+chunk, func(t *testing.T) {
			info := loadgen.ScenarioInfo{
				RunID:       fmt.Sprintf("owning-self-%s-%d", chunk, time.Now().UnixNano()),
				ExecutionID: "execution", OwningExecutionWorkers: true,
				Configuration: loadgen.RunConfiguration{Iterations: 1},
				Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
					OwningExecutionSelfActionsFlag: "true", IterFlag: "2", ContinueAsNewAfterIterFlag: chunk,
					NexusEnabledFlag: "false", IncludeStandaloneActivityFlag: "false",
					SleepTimeFlag: "1ms", VisibilityVerificationTimeoutFlag: "10s",
				}),
			}
			executor := newThroughputStressExecutor()
			_, err := env.RunExecutorTest(t, executor, info, clioptions.LangGo)
			require.NoError(t, err)
			require.Equal(t, 1, executor.Snapshot().(tpsState).CompletedIterations)
		})
	}
}

func TestThroughputOwningExecutionRequiresWorkerProtocol(t *testing.T) {
	info := loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
		OwningExecutionSelfActionsFlag: "true",
	})}
	err := newThroughputStressExecutor().Configure(info)
	require.ErrorContains(t, err, "verified Go worker protocol support")
}

func TestThroughputOwningExecutionActions(t *testing.T) {
	info := loadgen.ScenarioInfo{
		RunID: "self-actions", ExecutionID: "execution", Logger: zap.NewNop().Sugar(),
		OwningExecutionWorkers: true,
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			OwningExecutionSelfActionsFlag: "true",
			IterFlag:                       "1", NexusEnabledFlag: "false", IncludeDescribeFlag: "true",
		}),
	}
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	selfActions := 0
	updates := 0
	walkActions(executor.createActions(info.NewRun(1))[0].GetActions(), func(action *ks.Action) {
		self := action.GetExecActivity().GetClient()
		if self == nil {
			return
		}
		selfActions++
		require.Equal(t, ks.ExecuteActivityAction_ClientActivity_OWNING_EXECUTION, self.TargetPolicy)
		for _, set := range self.ClientSequence.ActionSets {
			for _, clientAction := range set.Actions {
				if update := clientAction.GetDoUpdate(); update != nil {
					updates++
					require.False(t, update.WithStart)
				}
			}
		}
	})
	require.Positive(t, selfActions)
	require.Positive(t, updates)
}
