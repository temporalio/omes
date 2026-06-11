package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

// oooSignalKvs extracts the SetWorkflowState key/value map carried by a single
// shuffled-signal client action.
func oooSignalKvs(t *testing.T, ca *kitchensink.ClientAction) map[string]string {
	t.Helper()
	set := ca.GetDoSignal().GetDoSignalActions().GetDoActions()
	require.NotNil(t, set, "signal should carry a DoActions action set")
	require.Len(t, set.Actions, 1)
	ws := set.Actions[0].GetSetWorkflowState()
	require.NotNil(t, ws, "signal action should set workflow state")
	return ws.Kvs
}

func TestOutOfOrderSignals_SeededRngIsDeterministic(t *testing.T) {
	t.Parallel()

	draw := func(r interface{ Intn(int) int }) []int {
		out := make([]int, 10)
		for i := range out {
			out[i] = r.Intn(1000)
		}
		return out
	}

	a := draw(oooSeededRng("run-a", 0))
	b := draw(oooSeededRng("run-a", 0))
	require.Equal(t, a, b, "same (RunID, iteration) must yield the same sequence")

	require.NotEqual(t, a, draw(oooSeededRng("run-a", 1)),
		"different iteration should diverge")
	require.NotEqual(t, a, draw(oooSeededRng("run-b", 0)),
		"different RunID should diverge")
}

func TestOutOfOrderSignals_BuildShuffledSignals(t *testing.T) {
	t.Parallel()

	const n = 6

	t.Run("zero shuffle keeps event order and grows state cumulatively", func(t *testing.T) {
		seq := buildShuffledSignals(n, 0, oooSeededRng("run", 0))
		require.Len(t, seq.ActionSets, 1)
		signals := seq.ActionSets[0].Actions
		require.Len(t, signals, n)

		for pos, sig := range signals {
			kvs := oooSignalKvs(t, sig)
			// Without shuffling, position p carries events 1..p+1.
			require.Len(t, kvs, pos+1)
			for e := 1; e <= pos+1; e++ {
				require.Equal(t, oooSignalReadyValue, kvs[oooEventKey(e)],
					"position %d should include %s", pos, oooEventKey(e))
			}
		}
	})

	t.Run("full shuffle still delivers every event exactly once with monotonic state", func(t *testing.T) {
		seq := buildShuffledSignals(n, 100, oooSeededRng("run", 0))
		signals := seq.ActionSets[0].Actions
		require.Len(t, signals, n)

		seenNewKey := map[string]bool{}
		prevSize := 0
		for pos, sig := range signals {
			kvs := oooSignalKvs(t, sig)
			// Each signal adds exactly one new key to the cumulative snapshot.
			require.Equal(t, prevSize+1, len(kvs), "snapshot must grow by one at position %d", pos)
			prevSize = len(kvs)
			for k := range kvs {
				seenNewKey[k] = true
			}
		}
		require.Len(t, seenNewKey, n, "every event key should appear")
		for e := 1; e <= n; e++ {
			require.True(t, seenNewKey[oooEventKey(e)], "missing %s", oooEventKey(e))
		}
	})

	t.Run("reproducible for a fixed seed", func(t *testing.T) {
		s1 := buildShuffledSignals(n, 100, oooSeededRng("run", 7))
		s2 := buildShuffledSignals(n, 100, oooSeededRng("run", 7))
		require.True(t, proto.Equal(s1, s2), "same seed must produce identical signal sequences")
	})
}

func TestOutOfOrderSignals_BuildOrderedAwaitActions(t *testing.T) {
	t.Parallel()

	const n = 4

	t.Run("without processing time", func(t *testing.T) {
		sets := buildOrderedAwaitActions(n, 0)
		require.Len(t, sets, 1)
		actions := sets[0].Actions
		require.Len(t, actions, n+1)
		for i := 0; i < n; i++ {
			await := actions[i].GetAwaitWorkflowState()
			require.NotNil(t, await, "action %d should await workflow state", i)
			require.Equal(t, oooEventKey(i+1), await.Key)
			require.Equal(t, oooSignalReadyValue, await.Value)
		}
		require.NotNil(t, actions[n].GetReturnResult())
	})

	t.Run("with processing time inserts an activity per event", func(t *testing.T) {
		sets := buildOrderedAwaitActions(n, time.Millisecond)
		actions := sets[0].Actions
		require.Len(t, actions, 2*n+1)
		for i := 0; i < n; i++ {
			require.NotNil(t, actions[2*i].GetAwaitWorkflowState())
			require.NotNil(t, actions[2*i+1].GetExecActivity())
		}
		require.NotNil(t, actions[2*n].GetReturnResult())
	})
}

func TestOutOfOrderSignals(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute))

	baseRunID := fmt.Sprintf("ooo-%d", time.Now().Unix())

	run := func(t *testing.T, suffix string, opts map[string]string) {
		executor := loadgen.GetScenario("out_of_order_signals").ExecutorFn()
		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + suffix,
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
			},
			ScenarioOptions: opts,
		}, clioptions.LangGo)
		require.NoError(t, err)
	}

	t.Run("shuffled signals processed in order", func(t *testing.T) {
		run(t, "-shuffle", map[string]string{
			oooSignalsCountFlag:             "5",
			oooSignalsShufflePercentageFlag: "100",
		})
	})

	t.Run("in-order signals with per-signal processing", func(t *testing.T) {
		run(t, "-inorder", map[string]string{
			oooSignalsCountFlag:             "4",
			oooSignalsShufflePercentageFlag: "0",
			oooSignalsProcessingTimeFlag:    "1ms",
		})
	})
}

func TestOutOfOrderSignals_ParseConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{}}
		cfg, err := parseOutOfOrderSignalsConfig(info)
		require.NoError(t, err)
		require.Equal(t, 10, cfg.signalsPerWorkflow)
		require.Equal(t, 100, cfg.shufflePercentage)
		require.Equal(t, time.Duration(0), cfg.processingTime)
	})

	t.Run("overrides", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{
			oooSignalsCountFlag:             "5",
			oooSignalsShufflePercentageFlag: "50",
			oooSignalsProcessingTimeFlag:    "250ms",
		}}
		cfg, err := parseOutOfOrderSignalsConfig(info)
		require.NoError(t, err)
		require.Equal(t, 5, cfg.signalsPerWorkflow)
		require.Equal(t, 50, cfg.shufflePercentage)
		require.Equal(t, 250*time.Millisecond, cfg.processingTime)
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		cases := map[string]map[string]string{
			"non-positive count":  {oooSignalsCountFlag: "0"},
			"shuffle below range": {oooSignalsShufflePercentageFlag: "-1"},
			"shuffle above range": {oooSignalsShufflePercentageFlag: "101"},
			"negative processing": {oooSignalsProcessingTimeFlag: "-1s"},
		}
		for name, opts := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := parseOutOfOrderSignalsConfig(&loadgen.ScenarioInfo{ScenarioOptions: opts})
				require.Error(t, err)
			})
		}
	})
}
