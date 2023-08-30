package loadgen

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestIterationsAndDuration(t *testing.T) {
	info := ScenarioInfo{MetricsHandler: client.MetricsNopHandler}

	var err error
	{
		// only iterations
		executor := &GenericExecutor{DefaultConfiguration: RunConfiguration{Iterations: 3}}
		_, err = executor.newRun(info)
		require.NoError(t, err)
	}
	{
		// only duration
		executor := &GenericExecutor{DefaultConfiguration: RunConfiguration{Duration: time.Hour}}
		_, err = executor.newRun(info)
		require.NoError(t, err)
	}
	{
		// both
		executor := &GenericExecutor{DefaultConfiguration: RunConfiguration{Duration: 3 * time.Second, Iterations: 3}}
		_, err = executor.newRun(info)
		require.ErrorContains(t, err, "invalid scenario: iterations and duration are mutually exclusive")
	}
}

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
	for iter := 1; iter <= iterations; iter++ {
		require.Contains(t, i.seen, iter)
	}
}

func execute(executor *GenericExecutor) error {
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	info := ScenarioInfo{
		MetricsHandler: client.MetricsNopHandler,
		Logger:         logger.Sugar(),
	}
	return executor.Run(context.Background(), info)
}

func TestRunHappyPathIterations(t *testing.T) {
	tracker := newIterationTracker()
	err := execute(&GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			tracker.track(run.Iteration)
			return nil
		},
		DefaultConfiguration: RunConfiguration{Iterations: 5},
	})
	require.NoError(t, err)
	tracker.assertSeen(t, 5)
}

func TestRunFailIterations(t *testing.T) {
	tracker := newIterationTracker()
	concurrency := 3
	err := execute(&GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			tracker.track(run.Iteration)
			// Start this short timer to allow all concurrent routines to be spawned
			<-time.After(time.Millisecond)
			if run.Iteration == 2 {
				return errors.New("deliberate fail from test")
			}
			return nil
		},
		DefaultConfiguration: RunConfiguration{MaxConcurrent: concurrency, Iterations: 50},
	})
	require.ErrorContains(t, err, "run finished with error")
	tracker.assertSeen(t, 2)
}

func TestRunHappyPathDuration(t *testing.T) {
	tracker := newIterationTracker()
	err := execute(&GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			tracker.track(run.Iteration)
			time.Sleep(time.Millisecond * 20)
			return nil
		},
		DefaultConfiguration: RunConfiguration{Duration: 100 * time.Millisecond},
	})
	require.NoError(t, err)
	tracker.assertSeen(t, DefaultMaxConcurrent*2)
}

func TestRunFailDuration(t *testing.T) {
	tracker := newIterationTracker()
	err := execute(&GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			tracker.track(run.Iteration)
			if run.Iteration == 2 {
				return errors.New("deliberate fail from test")
			}
			return nil
		},
		DefaultConfiguration: RunConfiguration{Duration: 200 * time.Millisecond},
	})
	require.ErrorContains(t, err, "run finished with error")
	tracker.assertSeen(t, 2)
}
