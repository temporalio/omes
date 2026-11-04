package loadgen

import (
	"context"
	"errors"
	"strings"
	"testing"
	"testing/synctest"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
)

func TestGenericExecutorRunStatistics(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stats := &RunStatistics{}
		attempts := make(map[int]int)

		err := execute(&GenericExecutor{Execute: func(_ context.Context, run *Run) error {
			attempts[run.Iteration]++
			if run.Iteration == 2 || attempts[run.Iteration] == 1 {
				return errors.New("iteration error")
			}
			return nil
		}}, RunConfiguration{Iterations: 3, MaxConcurrent: 1, MaxIterationAttempts: 2, ContinueOnIterationFailure: true, Statistics: stats})
		require.Error(t, err)

		snapshot := stats.Snapshot()
		require.Equal(t, RunStatisticsSnapshot{
			Configuration: RunConfiguration{
				Iterations: 3, MaxConcurrent: 1, MaxIterationAttempts: 2,
				ContinueOnIterationFailure: true, Statistics: stats,
			},
			Iterations: &IterationCounts{Started: 3, Succeeded: 2, Failed: 1},
			Failures:   []IterationFailure{{Iteration: 2, Attempt: 2, Error: "iteration 2 failed: iteration error"}},
		}, snapshot)
	})
}

func TestGenericExecutorRunStatisticsFailureLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stats := &RunStatistics{}

		err := execute(&GenericExecutor{Execute: func(context.Context, *Run) error { return errors.New("failed") }},
			RunConfiguration{Iterations: 101, ContinueOnIterationFailure: true, Statistics: stats})
		require.Error(t, err)

		snapshot := stats.Snapshot()
		require.Equal(t, &IterationCounts{Started: 101, Failed: 101}, snapshot.Iterations)
		require.Len(t, snapshot.Failures, 100)
	})
}

func TestGenericExecutorRunStatisticsEffectiveDefaults(t *testing.T) {
	stats := &RunStatistics{}
	require.Equal(t, RunStatisticsSnapshot{}, stats.Snapshot())

	require.NoError(t, execute(&GenericExecutor{Execute: func(context.Context, *Run) error { return nil }}, RunConfiguration{Statistics: stats}))

	snapshot := stats.Snapshot()
	require.Equal(t, RunStatisticsSnapshot{
		Configuration: RunConfiguration{
			Iterations: DefaultIterations, MaxConcurrent: DefaultMaxConcurrentIterations,
			MaxIterationAttempts: DefaultMaxIterationAttempts, Statistics: stats,
		},
		Iterations: &IterationCounts{Started: DefaultIterations, Succeeded: DefaultIterations},
	}, snapshot)
}

func TestRunStatisticsTruncatesErrors(t *testing.T) {
	stats := &RunStatistics{}
	stats.configure(RunConfiguration{})

	long := strings.Repeat("é", statisticsMaxErrorBytes)
	stats.iterationFinished(1, 1, errors.New(long), false)

	message := stats.Snapshot().Failures[0].Error
	require.LessOrEqual(t, len(message), statisticsMaxErrorBytes+len("… (truncated)"))
	require.True(t, strings.HasSuffix(message, "… (truncated)"))
	require.True(t, utf8.ValidString(message))
}
