package runner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/omes/app"
	"github.com/temporalio/omes/scenario"
	"go.uber.org/zap"
)

var metrics = app.MetricsForTesting()

func TestRunnerIterationsAndDuration(t *testing.T) {
	var err error
	// only iterations
	_, err = NewRunner(Options{Scenario: &scenario.Scenario{Iterations: 3}}, metrics, nil)
	assert.NoError(t, err)
	// only duration
	_, err = NewRunner(Options{Scenario: &scenario.Scenario{Duration: 3 * time.Second}}, metrics, nil)
	assert.NoError(t, err)
	// empty
	_, err = NewRunner(Options{Scenario: &scenario.Scenario{}}, metrics, nil)
	assert.ErrorContains(t, err, "invalid scenario: either iterations or duration is required")
	// both
	_, err = NewRunner(Options{Scenario: &scenario.Scenario{Duration: 3 * time.Second, Iterations: 3}}, metrics, nil)
	assert.ErrorContains(t, err, "invalid scenario: iterations and duration are mutually exclusive")
}

func TestCalcConcurrency(t *testing.T) {
	assert.Equal(t, 3, calcConcurrency(3, &scenario.Scenario{Concurrency: 6}))
	assert.Equal(t, 6, calcConcurrency(0, &scenario.Scenario{Concurrency: 6}))
	assert.Equal(t, DEFAULT_CONCURRENCY, calcConcurrency(0, &scenario.Scenario{}))
	assert.Equal(t, 5, calcConcurrency(5, &scenario.Scenario{}))
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

func run(options Options) error {
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	runner, err := NewRunner(options, metrics, logger.Sugar())
	if err != nil {
		return err
	}
	return runner.Run(context.Background(), nil)
}

func TestRunHappyPathIterations(t *testing.T) {
	tracker := newIterationTracker()
	err := run(Options{
		Scenario: &scenario.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenario.Run) error {
				tracker.track(run.IterationInTest)
				return nil
			},
			Iterations: 5,
		},
	})
	assert.NoError(t, err)
	tracker.assertSeen(t, 5)
}

func TestRunFailIterations(t *testing.T) {
	tracker := newIterationTracker()
	concurrency := 3
	err := run(Options{
		Scenario: &scenario.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenario.Run) error {
				tracker.track(run.IterationInTest)
				if run.IterationInTest == 2 {
					return errors.New("deliberate fail from test")
				}
				// Wait for cancellation
				<-ctx.Done()
				return nil
			},
			Concurrency: concurrency,
			Iterations:  50,
		},
	})
	assert.ErrorContains(t, err, "run finished with errors")
	tracker.assertSeen(t, concurrency)
}

func TestRunHappyPathDuration(t *testing.T) {
	tracker := newIterationTracker()
	err := run(Options{
		Scenario: &scenario.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenario.Run) error {
				tracker.track(run.IterationInTest)
				time.Sleep(time.Millisecond * 50)
				return nil
			},
			Duration: 100 * time.Millisecond,
		},
	})
	assert.NoError(t, err)
	tracker.assertSeen(t, DEFAULT_CONCURRENCY*2)
}

func TestRunFailDuration(t *testing.T) {
	var numIterationSeen atomic.Uint32
	err := run(Options{
		Scenario: &scenario.Scenario{
			Name: "test",
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
		},
	})
	assert.ErrorContains(t, err, "run finished with errors")
	assert.LessOrEqual(t, numIterationSeen.Load(), uint32(DEFAULT_CONCURRENCY))
}
