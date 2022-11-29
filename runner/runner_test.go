package runner

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/omes/scenarios"
	"go.uber.org/zap"
)

func TestRunnerIterationsAndDuration(t *testing.T) {
	var err error
	// only iterations
	_, err = NewRunner(Options{Scenario: scenarios.Scenario{Iterations: 3}}, nil)
	assert.NoError(t, err)
	// only duration
	_, err = NewRunner(Options{Scenario: scenarios.Scenario{Duration: 3 * time.Second}}, nil)
	assert.NoError(t, err)
	// empty
	_, err = NewRunner(Options{Scenario: scenarios.Scenario{}}, nil)
	assert.ErrorContains(t, err, "invalid scenario: either iterations or duration is required")
	// both
	_, err = NewRunner(Options{Scenario: scenarios.Scenario{Duration: 3 * time.Second, Iterations: 3}}, nil)
	assert.ErrorContains(t, err, "invalid scenario: iterations and duration are mutually exclusive")
}

func TestCalcConcurrency(t *testing.T) {
	assert.Equal(t, 3, calcConcurrency(3, scenarios.Scenario{Concurrency: 6}))
	assert.Equal(t, 6, calcConcurrency(0, scenarios.Scenario{Concurrency: 6}))
	assert.Equal(t, DEFAULT_CONCURRENCY, calcConcurrency(0, scenarios.Scenario{}))
	assert.Equal(t, 5, calcConcurrency(5, scenarios.Scenario{}))
}

type iterationTracker struct {
	sync.Mutex
	seen []uint32
}

func newIterationTracker() *iterationTracker {
	return &iterationTracker{seen: make([]uint32, 0)}
}

func (i *iterationTracker) track(iteration uint32) {
	i.Lock()
	defer i.Unlock()
	i.seen = append(i.seen, iteration)
}

func (i *iterationTracker) assertSeen(t *testing.T, iterations uint32) {
	for iter := uint32(1); iter <= iterations; iter++ {
		assert.Contains(t, i.seen, iter)
	}
}

func run(options Options, failFast bool) error {
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	runner, err := NewRunner(options, logger.Sugar())
	if err != nil {
		return err
	}
	return runner.Run(context.Background(), RunOptions{FailFast: failFast})
}

func TestRunHappyPathIterations(t *testing.T) {
	tracker := newIterationTracker()
	err := run(Options{
		Scenario: scenarios.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenarios.Run) error {
				tracker.track(run.IterationInTest)
				return nil
			},
			Iterations: 5,
		},
	}, false)
	assert.NoError(t, err)
	tracker.assertSeen(t, 5)
}

func TestRunFailFastIterations(t *testing.T) {
	tracker := newIterationTracker()
	concurrency := uint32(3)
	err := run(Options{
		Scenario: scenarios.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenarios.Run) error {
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
	}, true)
	assert.ErrorContains(t, err, "run finished with errors")
	tracker.assertSeen(t, concurrency)
}

func TestRunFailEventuallyIterations(t *testing.T) {
	tracker := newIterationTracker()
	err := run(Options{
		Scenario: scenarios.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenarios.Run) error {
				tracker.track(run.IterationInTest)
				if run.IterationInTest == 2 {
					return errors.New("deliberate fail from test")
				}
				return nil
			},
			Iterations: 50,
		},
	}, false)
	assert.ErrorContains(t, err, "run finished with errors")
	tracker.assertSeen(t, 50)
}

func TestRunHappyPathDuration(t *testing.T) {
	tracker := newIterationTracker()
	err := run(Options{
		Scenario: scenarios.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenarios.Run) error {
				tracker.track(run.IterationInTest)
				time.Sleep(time.Millisecond * 50)
				return nil
			},
			Duration: 100 * time.Millisecond,
		},
	}, false)
	assert.NoError(t, err)
	tracker.assertSeen(t, DEFAULT_CONCURRENCY*2)
}

func TestRunFailFastDuration(t *testing.T) {
	var numIterationSeen atomic.Uint32
	err := run(Options{
		Scenario: scenarios.Scenario{
			Name: "test",
			Execute: func(ctx context.Context, run *scenarios.Run) error {
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
	}, true)
	assert.ErrorContains(t, err, "run finished with errors")
	assert.LessOrEqual(t, numIterationSeen.Load(), uint32(DEFAULT_CONCURRENCY))
}
