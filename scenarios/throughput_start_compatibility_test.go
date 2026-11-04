package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	enumspb "go.temporal.io/api/enums/v1"
)

func TestThroughputLegacyStartAndSnapshotResume(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	id := fmt.Sprintf("legacy-start-%d", time.Now().UnixNano())
	info := loadgen.ScenarioInfo{RunID: id, ExecutionID: id, Configuration: loadgen.RunConfiguration{Iterations: 1, IgnoreAlreadyStarted: true}, Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{IterFlag: "1", ContinueAsNewAfterIterFlag: "0", NexusEnabledFlag: "false", SleepTimeFlag: "1ms", VisibilityVerificationTimeoutFlag: "10s"})}
	first := newThroughputStressExecutor()
	_, err := env.RunExecutorTest(t, first, info, clioptions.LangGo)
	require.NoError(t, err)
	workflowID := fmt.Sprintf("w-%s-%s-1", id, id)
	checkUnmarked := func() {
		history := env.TemporalClient().GetWorkflowHistory(t.Context(), workflowID, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
		event, err := history.Next()
		require.NoError(t, err)
		require.NotContains(t, event.GetWorkflowExecutionStartedEventAttributes().GetMemo().GetFields(), "omes.original-start")
	}
	checkUnmarked()
	resumed := newThroughputStressExecutor()
	require.NoError(t, resumed.LoadState(func(v any) error { v.(*tpsState).CompletedIterations = 0; return nil }))
	info.Configuration.IgnoreAlreadyStarted = false
	_, err = env.RunExecutorTest(t, resumed, info, clioptions.LangGo)
	require.NoError(t, err)
	require.Equal(t, 1, resumed.Snapshot().(tpsState).CompletedIterations)
	checkUnmarked()
}
