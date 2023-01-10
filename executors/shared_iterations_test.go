package executors

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metricsComponent "github.com/temporalio/omes/components/metrics"
	"github.com/temporalio/omes/scenario"
	"go.uber.org/zap"
)

func TestIterationsAndDuration(t *testing.T) {
	metrics := metricsComponent.MustSetup(&metricsComponent.Options{}, nil)
	options := scenario.RunOptions{Metrics: metrics}

	var err error
	{
		// only iterations
		executor := &SharedIterationsExecutor{Iterations: 3}
		_, err = executor.newRun(&options)
		assert.NoError(t, err)
	}
	{
		// only duration
		executor := &SharedIterationsExecutor{Duration: time.Hour}
		_, err = executor.newRun(&options)
		assert.NoError(t, err)
	}
	{
		// empty
		executor := &SharedIterationsExecutor{}
		_, err = executor.newRun(&options)
		assert.ErrorContains(t, err, "invalid scenario: either iterations or duration is required")
	}
	{
		// both
		executor := &SharedIterationsExecutor{Duration: 3 * time.Second, Iterations: 3}
		_, err = executor.newRun(&options)
		assert.ErrorContains(t, err, "invalid scenario: iterations and duration are mutually exclusive")
	}
}

func TestCalcConcurrency(t *testing.T) {
	assert.Equal(t, 3, calcConcurrency(3, 6))
	assert.Equal(t, 6, calcConcurrency(0, 6))
	assert.Equal(t, defaultConcurrency, calcConcurrency(0, 0))
	assert.Equal(t, 5, calcConcurrency(5, 0))
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
		assert.Contains(t, i.seen, iter)
	}
}

func execute(executor *SharedIterationsExecutor) error {
	var metrics = metricsComponent.MustSetup(&metricsComponent.Options{}, nil)
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	options := &scenario.RunOptions{
		Metrics: metrics,
		Logger:  logger.Sugar(),
	}
	return executor.Run(context.Background(), options)
}

func TestRunHappyPathIterations(t *testing.T) {
	tracker := newIterationTracker()
	err := execute(&SharedIterationsExecutor{
		Execute: func(ctx context.Context, run *scenario.Run) error {
			tracker.track(run.IterationInTest)
			return nil
		},
		Iterations: 5,
	})
	assert.NoError(t, err)
	tracker.assertSeen(t, 5)
}

func TestRunFailIterations(t *testing.T) {
	tracker := newIterationTracker()
	concurrency := 3
	err := execute(&SharedIterationsExecutor{
		Execute: func(ctx context.Context, run *scenario.Run) error {
			tracker.track(run.IterationInTest)
			// Start this short timer to allow all concurrent routines to be spawned
			<-time.After(time.Millisecond)
			if run.IterationInTest == 2 {
				return errors.New("deliberate fail from test")
			}
			// Wait for cancellation
			<-ctx.Done()
			return nil
		},
		Concurrency: concurrency,
		Iterations:  50,
	})
	assert.ErrorContains(t, err, "run finished with errors")
	tracker.assertSeen(t, concurrency)
}

func TestRunHappyPathDuration(t *testing.T) {
	tracker := newIterationTracker()
	err := execute(&SharedIterationsExecutor{
		Execute: func(ctx context.Context, run *scenario.Run) error {
			tracker.track(run.IterationInTest)
			time.Sleep(time.Millisecond * 50)
			return nil
		},
		Duration: 100 * time.Millisecond,
	})
	assert.NoError(t, err)
	tracker.assertSeen(t, defaultConcurrency*2)
}

func TestRunFailDuration(t *testing.T) {
	var numIterationSeen atomic.Uint32
	err := execute(&SharedIterationsExecutor{
		Execute: func(ctx context.Context, run *scenario.Run) error {
			numIterationSeen.Add(1)
			if run.IterationInTest == 2 {
				return errors.New("deliberate fail from test")
			}
			// Wait for cancellation
			<-ctx.Done()
			return nil
		},
		Duration: 200 * time.Millisecond,
	})
	assert.ErrorContains(t, err, "run finished with errors")
	assert.LessOrEqual(t, numIterationSeen.Load(), uint32(defaultConcurrency))
}
