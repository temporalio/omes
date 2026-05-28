package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

func TestWorkflowWithManyTimers_ParseConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{}}
		cfg, err := parseManyTimersConfig(info)
		require.NoError(t, err)
		require.Equal(t, 30, cfg.concurrentTimers)
		require.Equal(t, 10*time.Second, cfg.timerDuration)
		require.Equal(t, time.Duration(0), cfg.timerJitter)
		require.Equal(t, 1, cfg.iterations)
	})

	t.Run("overrides", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{
			manyTimersConcurrentTimersFlag: "4",
			manyTimersTimerDurationFlag:    "10ms",
			manyTimersTimerJitterFlag:      "5ms",
			manyTimersIterationsFlag:       "3",
		}}
		cfg, err := parseManyTimersConfig(info)
		require.NoError(t, err)
		require.Equal(t, 4, cfg.concurrentTimers)
		require.Equal(t, 10*time.Millisecond, cfg.timerDuration)
		require.Equal(t, 5*time.Millisecond, cfg.timerJitter)
		require.Equal(t, 3, cfg.iterations)
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		cases := map[string]map[string]string{
			"non-positive concurrent-timers": {manyTimersConcurrentTimersFlag: "0"},
			"non-positive timer-duration":    {manyTimersTimerDurationFlag: "0"},
			"negative timer-duration-jitter": {manyTimersTimerJitterFlag: "-1s"},
			"non-positive iterations":        {manyTimersIterationsFlag: "0"},
		}
		for name, opts := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := parseManyTimersConfig(&loadgen.ScenarioInfo{ScenarioOptions: opts})
				require.Error(t, err)
			})
		}
	})
}

func TestWorkflowWithManyTimers_BuildActions(t *testing.T) {
	t.Parallel()

	t.Run("one concurrent batch per iteration plus a terminal return", func(t *testing.T) {
		cfg := &manyTimersConfig{
			concurrentTimers: 4,
			timerDuration:    10 * time.Millisecond,
			iterations:       3,
		}
		sets := buildManyTimersActions(cfg, manyTimersSeededRng("test"))
		require.Len(t, sets, 4, "3 timer batches + 1 terminal return set")

		for i := 0; i < 3; i++ {
			require.True(t, sets[i].Concurrent, "timer batch %d should fire concurrently", i)
			require.Len(t, sets[i].Actions, 4, "batch %d should hold concurrent-timers timers", i)
			for _, a := range sets[i].Actions {
				require.NotNil(t, a.GetTimer(), "batch %d action should be a timer", i)
				require.Equal(t, uint64(10), a.GetTimer().GetMilliseconds(),
					"zero jitter should leave every timer at timer-duration")
			}
		}
		require.NotNil(t, sets[3].Actions[0].GetReturnResult(), "final set should return a result")
	})

	t.Run("jitter spreads durations within [base, base+jitter)", func(t *testing.T) {
		base := 10 * time.Second
		jitter := 5 * time.Second
		cfg := &manyTimersConfig{
			concurrentTimers: 50,
			timerDuration:    base,
			timerJitter:      jitter,
			iterations:       1,
		}
		sets := buildManyTimersActions(cfg, manyTimersSeededRng("run-jitter"))
		timers := sets[0].Actions
		require.Len(t, timers, 50)

		distinct := map[uint64]struct{}{}
		for _, a := range timers {
			ms := a.GetTimer().GetMilliseconds()
			require.GreaterOrEqual(t, ms, uint64(base.Milliseconds()), "below base")
			require.Less(t, ms, uint64((base + jitter).Milliseconds()), "at or above base+jitter")
			distinct[ms] = struct{}{}
		}
		require.Greater(t, len(distinct), 1, "jitter should produce varied durations")
	})

	t.Run("jitter is reproducible for a fixed run ID", func(t *testing.T) {
		cfg := &manyTimersConfig{
			concurrentTimers: 20,
			timerDuration:    time.Second,
			timerJitter:      time.Second,
			iterations:       1,
		}
		durations := func(rng string) []uint64 {
			sets := buildManyTimersActions(cfg, manyTimersSeededRng(rng))
			out := make([]uint64, 0, len(sets[0].Actions))
			for _, a := range sets[0].Actions {
				out = append(out, a.GetTimer().GetMilliseconds())
			}
			return out
		}
		require.Equal(t, durations("run-x"), durations("run-x"), "same run ID must reproduce durations")
		require.NotEqual(t, durations("run-x"), durations("run-y"), "different run ID should diverge")
	})
}

func TestWorkflowWithManyTimers(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(1*time.Minute))

	baseRunID := fmt.Sprintf("wmt-%d", time.Now().Unix())

	t.Run("multiple sequential batches of concurrent timers", func(t *testing.T) {
		executor := loadgen.GetScenario("workflow_with_many_timers").ExecutorFn()

		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-m",
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
			},
			ScenarioOptions: map[string]string{
				manyTimersConcurrentTimersFlag: "5",
				manyTimersTimerDurationFlag:    "10ms",
				manyTimersIterationsFlag:       "2",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
	})
}
