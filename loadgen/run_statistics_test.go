package loadgen

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

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
		require.Equal(t, &IterationCounts{Started: 3, Succeeded: 2, Failed: 1}, snapshot.Iterations)
		require.Equal(t, []IterationFailure{{Iteration: 2, Attempt: 2, Error: "iteration 2 failed: iteration error"}}, snapshot.Failures)
		require.Equal(t, 1, snapshot.Configuration.MaxConcurrent)
	})
}

func TestGenericExecutorRunStatisticsCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		stats := &RunStatistics{}
		err := executeContext(ctx, &GenericExecutor{Execute: func(ctx context.Context, _ *Run) error {
			cancel()
			return ctx.Err()
		}}, RunConfiguration{Iterations: 1, Statistics: stats})
		require.ErrorIs(t, err, context.Canceled)
		require.Equal(t, &IterationCounts{Started: 1}, stats.Snapshot().Iterations)
		require.Empty(t, stats.Snapshot().Failures)
	})
}

func TestGenericExecutorRunStatisticsFailureLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		stats := &RunStatistics{}
		err := execute(&GenericExecutor{Execute: func(context.Context, *Run) error { return errors.New("failed") }},
			RunConfiguration{Iterations: 101, ContinueOnIterationFailure: true, Statistics: stats})
		require.Error(t, err)
		require.Equal(t, &IterationCounts{Started: 101, Failed: 101}, stats.Snapshot().Iterations)
		require.Len(t, stats.Snapshot().Failures, 100)
	})
}

func TestGenericExecutorRunStatisticsEffectiveDefaults(t *testing.T) {
	stats := &RunStatistics{}
	require.Nil(t, stats.Snapshot().Iterations)
	require.NoError(t, execute(&GenericExecutor{Execute: func(context.Context, *Run) error { return nil }}, RunConfiguration{Statistics: stats}))
	snapshot := stats.Snapshot()
	require.Equal(t, DefaultIterations, snapshot.Configuration.Iterations)
	require.Equal(t, DefaultMaxConcurrentIterations, snapshot.Configuration.MaxConcurrent)
	require.Equal(t, DefaultMaxIterationAttempts, snapshot.Configuration.MaxIterationAttempts)
	require.Equal(t, time.Duration(0), snapshot.Configuration.Duration)
}

func TestGenericExecutorRunStatisticsAbandonment(t *testing.T) {
	for _, tc := range []struct {
		name  string
		retry bool
	}{
		{name: "timeout"},
		{name: "interrupted retry", retry: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				stats := &RunStatistics{}
				if tc.retry {
					go func() {
						time.Sleep(time.Millisecond)
						cancel()
					}()
				}
				err := executeContext(ctx, &GenericExecutor{Execute: func(ctx context.Context, _ *Run) error {
					if tc.retry {
						return errors.New("transient failure")
					}
					<-ctx.Done()
					return ctx.Err()
				}}, RunConfiguration{Iterations: 1, Timeout: time.Second, MaxIterationAttempts: 2, Statistics: stats})
				require.Error(t, err)
				require.Equal(t, &IterationCounts{Started: 1}, stats.Snapshot().Iterations)
				require.Empty(t, stats.Snapshot().Failures)
			})
		})
	}
}
