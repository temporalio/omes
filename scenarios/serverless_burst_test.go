package scenarios

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

func TestServerlessBurst(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(1*time.Minute))

	baseRunID := fmt.Sprintf("sb-%d", time.Now().Unix())

	t.Run("defaults", func(t *testing.T) {
		executor := loadgen.GetScenario("serverless_burst").ExecutorFn()

		var completed atomic.Int32
		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-d",
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
				OnCompletion: func(_ context.Context, _ *loadgen.Run) {
					completed.Add(1)
				},
			},
			ScenarioOptions: map[string]string{
				sbFastActivityCountFlag: "5",
				sbSlowActivityCountFlag: "2",
				sbSlowActivityDuration:  "2s",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
		require.EqualValues(t, 3, completed.Load())
	})

	t.Run("custom fast and slow counts with short duration", func(t *testing.T) {
		executor := loadgen.GetScenario("serverless_burst").ExecutorFn()

		var completed atomic.Int32
		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-c",
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
				OnCompletion: func(_ context.Context, _ *loadgen.Run) {
					completed.Add(1)
				},
			},
			ScenarioOptions: map[string]string{
				sbFastActivityCountFlag: "3",
				sbSlowActivityCountFlag: "1",
				sbSlowActivityDuration:  "100ms",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
		require.EqualValues(t, 3, completed.Load())
	})

	t.Run("fast activities only", func(t *testing.T) {
		executor := loadgen.GetScenario("serverless_burst").ExecutorFn()

		var completed atomic.Int32
		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-f",
			Configuration: loadgen.RunConfiguration{
				Iterations:    3,
				MaxConcurrent: 3,
				OnCompletion: func(_ context.Context, _ *loadgen.Run) {
					completed.Add(1)
				},
			},
			ScenarioOptions: map[string]string{
				sbFastActivityCountFlag: "5",
				sbSlowActivityCountFlag: "0",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
		require.EqualValues(t, 3, completed.Load())
	})

	t.Run("slow activities only", func(t *testing.T) {
		executor := loadgen.GetScenario("serverless_burst").ExecutorFn()

		var completed atomic.Int32
		_, err := env.RunExecutorTest(t, executor, loadgen.ScenarioInfo{
			RunID: baseRunID + "-s",
			Configuration: loadgen.RunConfiguration{
				Iterations:    2,
				MaxConcurrent: 2,
				OnCompletion: func(_ context.Context, _ *loadgen.Run) {
					completed.Add(1)
				},
			},
			ScenarioOptions: map[string]string{
				sbFastActivityCountFlag: "0",
				sbSlowActivityCountFlag: "2",
				sbSlowActivityDuration:  "200ms",
			},
		}, clioptions.LangGo)
		require.NoError(t, err)
		require.EqualValues(t, 2, completed.Load())
	})
}
