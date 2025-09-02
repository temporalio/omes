package loadgen

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type iterationTracker struct {
	sync.Mutex
	seen []int
}

func newIterationTracker() *iterationTracker {
	return &iterationTracker{seen: make([]int, 0)}
}

func (i *iterationTracker) track(iteration int) {
	i.Lock()
	defer i.Unlock()
	i.seen = append(i.seen, iteration)
}

func (i *iterationTracker) assertSeen(t *testing.T, iterations int) {
	i.Lock()
	defer i.Unlock()
	for iter := 1; iter <= iterations; iter++ {
		require.Contains(t, i.seen, iter)
	}
}

func execute(executor *GenericExecutor, runConfig RunConfiguration) error {
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	info := ScenarioInfo{
		MetricsHandler: client.MetricsNopHandler,
		Logger:         logger.Sugar(),
		Configuration:  runConfig,
	}
	return executor.Run(context.Background(), info)
}

func TestRunHappyPathIterations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				return nil
			}},
			RunConfiguration{Iterations: 5},
		)
		require.NoError(t, err)
		tracker.assertSeen(t, 5)
	})
}

func TestRunFailIterations(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		concurrency := 3
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				if run.Iteration == 2 {
					return errors.New("deliberate fail from test")
				}
				return nil
			}},
			RunConfiguration{MaxConcurrent: concurrency, Iterations: 50},
		)
		require.ErrorContains(t, err, "run finished with error")
		tracker.assertSeen(t, 2)
	})
}

func TestRunHappyPathDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				time.Sleep(time.Millisecond * 20)
				return nil
			}},
			RunConfiguration{Duration: 100 * time.Millisecond},
		)
		require.NoError(t, err)
		tracker.assertSeen(t, DefaultMaxConcurrentIterations*2)
	})
}

func TestRunFailDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				if run.Iteration == 2 {
					return errors.New("deliberate fail from test")
				}
				return nil
			}},
			RunConfiguration{Duration: 200 * time.Millisecond},
		)
		require.ErrorContains(t, err, "run finished with error")
		tracker.assertSeen(t, 2)
	})
}

func TestRunDurationWithTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				<-ctx.Done()
				return nil
			}},
			RunConfiguration{
				Duration: 100 * time.Millisecond,
				Timeout:  10 * time.Millisecond,
			},
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "timed out")
		tracker.assertSeen(t, 5)
	})
}

func TestRunIterationsWithTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				<-ctx.Done()
				return nil
			}},
			RunConfiguration{
				Iterations: 5,
				Timeout:    10 * time.Millisecond,
			},
		)
		require.Error(t, err)
		require.ErrorContains(t, err, "timed out")
		tracker.assertSeen(t, 2)
	})
}

func TestRunDurationWithoutTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		startTime := time.Now()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				time.Sleep(time.Millisecond * 20)
				return nil
			}},
			RunConfiguration{Duration: 1 * time.Millisecond},
		)
		require.Equal(t, time.Millisecond*20, time.Since(startTime))
		require.NoError(t, err)
		tracker.assertSeen(t, DefaultMaxConcurrentIterations)
	})
}

func TestRunIterationsWithoutTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tracker := newIterationTracker()
		startTime := time.Now()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				time.Sleep(time.Millisecond * 20)
				return nil
			}},
			RunConfiguration{Iterations: 5},
		)
		require.Equal(t, time.Millisecond*20, time.Since(startTime))
		require.NoError(t, err)
		tracker.assertSeen(t, 5)
	})
}

func TestRunIterationsWithRateLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		startTime := time.Now()
		tracker := newIterationTracker()
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				tracker.track(run.Iteration)
				return nil
			}},
			RunConfiguration{
				Iterations:             4,
				MaxConcurrent:          1,
				MaxIterationsPerSecond: 4.0,
			},
		)
		require.NoError(t, err)
		require.Equal(t, time.Second, time.Since(startTime))
		tracker.assertSeen(t, 4)
	})
}

func TestExecutorRetries(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalTracker := newIterationTracker()
		successTracker := newIterationTracker()

		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				totalTracker.track(run.Iteration)
				if len(totalTracker.seen) < 3 {
					return errors.New("transient failure")
				}
				successTracker.track(run.Iteration)
				return nil
			}},
			RunConfiguration{
				Iterations:           1,
				MaxIterationAttempts: 5,
			},
		)

		require.NoError(t, err)
		require.Equal(t, []int{1, 1, 1}, totalTracker.seen, "expected 3 attempts before success")
		require.Equal(t, []int{1}, successTracker.seen, "expected 1 success")
	})
}

func TestExecutorRetriesLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		totalTracker := newIterationTracker()

		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				totalTracker.track(run.Iteration)
				return errors.New("persistent failure")
			}},
			RunConfiguration{
				Iterations:           1,
				MaxIterationAttempts: 5,
			},
		)

		require.Error(t, err)
		require.Contains(t, err.Error(), "persistent failure")
		require.Equal(t, []int{1, 1, 1, 1, 1}, totalTracker.seen, "expected 5 attempts")
	})
}
