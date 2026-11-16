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

func TestGenericExecutorRunCounter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		configuration := RunConfiguration{
			Iterations: 3, MaxConcurrent: 1, MaxIterationAttempts: 2,
			ContinueOnIterationFailure: true,
		}
		counter := &RunCounter{}
		require.Equal(t, RunCounts{Failures: []IterationFailure{}}, counter.Snapshot())
		configuration.Counter = counter
		attempts := make(map[int]int)

		err := execute(&GenericExecutor{Execute: func(_ context.Context, run *Run) error {
			attempts[run.Iteration]++
			if run.Iteration == 2 || attempts[run.Iteration] == 1 {
				return errors.New("iteration error")
			}
			return nil
		}}, configuration)
		require.Error(t, err)

		counts := counter.Snapshot()
		require.Equal(t, RunCounts{
			Iterations: &IterationCounts{Started: 3, Succeeded: 2, Failed: 1},
			Failures:   []IterationFailure{{Iteration: 2, Attempt: 2, Error: "iteration 2 failed: iteration error"}},
		}, counts)
	})
}

func TestGenericExecutorRunCounterFailureLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		configuration := RunConfiguration{Iterations: 101, ContinueOnIterationFailure: true}
		counter := &RunCounter{}
		configuration.Counter = counter

		err := execute(&GenericExecutor{Execute: func(context.Context, *Run) error { return errors.New("failed") }},
			configuration)
		require.Error(t, err)

		counts := counter.Snapshot()
		require.Equal(t, &IterationCounts{Started: 101, Failed: 101}, counts.Iterations)
		require.Len(t, counts.Failures, 100)
	})
}

func TestGenericExecutorRunCounterEffectiveDefaults(t *testing.T) {
	counter := &RunCounter{}
	require.Equal(t, RunCounts{Failures: []IterationFailure{}}, counter.Snapshot())

	require.NoError(t, execute(
		&GenericExecutor{Execute: func(context.Context, *Run) error { return nil }},
		RunConfiguration{Counter: counter},
	))

	counts := counter.Snapshot()
	require.Equal(t, RunCounts{
		Iterations: &IterationCounts{Started: DefaultIterations, Succeeded: DefaultIterations},
		Failures:   []IterationFailure{},
	}, counts)
}

func TestGenericExecutorRunCounterInitiallyUnavailable(t *testing.T) {
	counter := &RunCounter{}
	require.Nil(t, counter.Snapshot().Iterations)

	require.NoError(t, execute(
		&GenericExecutor{Execute: func(context.Context, *Run) error { return nil }},
		RunConfiguration{Counter: counter},
	))

	require.Equal(t, &IterationCounts{Started: DefaultIterations, Succeeded: DefaultIterations}, counter.Snapshot().Iterations)
}

func TestRunCounterTruncatesErrors(t *testing.T) {
	counter := &RunCounter{}
	counter.runStarted()

	long := strings.Repeat("é", statisticsMaxErrorBytes)
	counter.iterationFinished(1, 1, errors.New(long), false)

	message := counter.Snapshot().Failures[0].Error
	require.LessOrEqual(t, len(message), statisticsMaxErrorBytes+len("… (truncated)"))
	require.True(t, strings.HasSuffix(message, "… (truncated)"))
	require.True(t, utf8.ValidString(message))
}
