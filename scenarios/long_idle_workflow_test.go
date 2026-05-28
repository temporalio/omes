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

func TestLongIdleWorkflow_ParseConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{}}
		cfg, err := parseLongIdleConfig(info)
		require.NoError(t, err)
		require.Equal(t, time.Minute, cfg.idleDuration)
		require.Equal(t, 1, cfg.cyclesPerRun)
		require.Equal(t, 0, cfg.activitiesPerCycle)
		require.Equal(t, time.Second, cfg.activityDuration)
		require.Equal(t, 0, cfg.continueAsNewIterations)
	})

	t.Run("overrides", func(t *testing.T) {
		info := &loadgen.ScenarioInfo{ScenarioOptions: map[string]string{
			longIdleIdleDurationFlag:            "5ms",
			longIdleCyclesPerRunFlag:            "3",
			longIdleActivitiesPerCycleFlag:      "2",
			longIdleActivityDurationFlag:        "250ms",
			longIdleContinueAsNewIterationsFlag: "4",
		}}
		cfg, err := parseLongIdleConfig(info)
		require.NoError(t, err)
		require.Equal(t, 5*time.Millisecond, cfg.idleDuration)
		require.Equal(t, 3, cfg.cyclesPerRun)
		require.Equal(t, 2, cfg.activitiesPerCycle)
		require.Equal(t, 250*time.Millisecond, cfg.activityDuration)
		require.Equal(t, 4, cfg.continueAsNewIterations)
	})

	t.Run("rejects invalid values", func(t *testing.T) {
		cases := map[string]map[string]string{
			"non-positive idle-duration": {longIdleIdleDurationFlag: "0"},
			"non-positive cycles":        {longIdleCyclesPerRunFlag: "0"},
			"negative activities":        {longIdleActivitiesPerCycleFlag: "-1"},
			"negative activity-duration": {longIdleActivityDurationFlag: "-1s"},
			"negative CAN iterations":    {longIdleContinueAsNewIterationsFlag: "-1"},
		}
		for name, opts := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := parseLongIdleConfig(&loadgen.ScenarioInfo{ScenarioOptions: opts})
				require.Error(t, err)
			})
		}
	})
}

func TestLongIdleWorkflow_BuildActions(t *testing.T) {
	t.Parallel()

	t.Run("no continue-as-new ends with return result", func(t *testing.T) {
		cfg := &longIdleConfig{
			idleDuration:       time.Millisecond,
			cyclesPerRun:       2,
			activitiesPerCycle: 1,
			activityDuration:   time.Millisecond,
		}
		sets := buildLongIdleActions(cfg, 0)
		require.Len(t, sets, 1)
		actions := sets[0].Actions
		// 2 cycles * (1 timer + 1 activity) + 1 terminal return.
		require.Len(t, actions, 5)
		require.NotNil(t, actions[0].GetTimer())
		require.NotNil(t, actions[1].GetExecActivity())
		require.NotNil(t, actions[2].GetTimer())
		require.NotNil(t, actions[3].GetExecActivity())
		require.NotNil(t, actions[4].GetReturnResult())
	})

	t.Run("continue-as-new chains a follow-on run", func(t *testing.T) {
		cfg := &longIdleConfig{
			idleDuration:       time.Millisecond,
			cyclesPerRun:       1,
			activitiesPerCycle: 0,
			activityDuration:   time.Millisecond,
		}
		sets := buildLongIdleActions(cfg, 1)
		require.Len(t, sets, 1)
		actions := sets[0].Actions
		// 1 timer + 1 continue-as-new terminal action.
		require.Len(t, actions, 2)
		require.NotNil(t, actions[0].GetTimer())
		can := actions[len(actions)-1].GetContinueAsNew()
		require.NotNil(t, can, "last action should continue-as-new when iterations remain")
		require.Len(t, can.GetArguments(), 1, "continue-as-new carries the next run's input")
	})
}

func TestLongIdleWorkflow(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute))

	baseRunID := fmt.Sprintf("liw-%d", time.Now().Unix())

	t.Run("idle cycles with activity burst", func(t *testing.T) {
		executor := loadgen.GetScenario("long_idle_workflow").ExecutorFn()

		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-c",
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
			},
			ScenarioOptions: map[string]string{
				longIdleIdleDurationFlag:       "1ms",
				longIdleCyclesPerRunFlag:       "2",
				longIdleActivitiesPerCycleFlag: "1",
				longIdleActivityDurationFlag:   "1ms",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
	})

	t.Run("continue-as-new chain", func(t *testing.T) {
		executor := loadgen.GetScenario("long_idle_workflow").ExecutorFn()

		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-can",
			Configuration: loadgen.RunConfiguration{
				Iterations:    2,
				MaxConcurrent: 2,
			},
			ScenarioOptions: map[string]string{
				longIdleIdleDurationFlag:            "1ms",
				longIdleContinueAsNewIterationsFlag: "2",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
	})
}
