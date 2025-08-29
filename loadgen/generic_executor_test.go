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

func TestScenarioConfigValidation(t *testing.T) {
	tests := []struct {
		name  string
		info  ScenarioInfo
		error string
	}{
		{
			name:  "default",
			info:  ScenarioInfo{Configuration: RunConfiguration{}},
			error: "",
		},
		{
			name:  "only iterations",
			info:  ScenarioInfo{Configuration: RunConfiguration{Iterations: 3}},
			error: "",
		},
		{
			name:  "start iterations smaller than iterations",
			info:  ScenarioInfo{Configuration: RunConfiguration{Iterations: 10, StartFromIteration: 3}},
			error: "",
		},
		{
			name:  "start iterations larger than iterations",
			info:  ScenarioInfo{Configuration: RunConfiguration{Iterations: 3, StartFromIteration: 10}},
			error: "invalid scenario: StartFromIteration 10 is greater than Iterations 3",
		},
		{
			name:  "only duration",
			info:  ScenarioInfo{Configuration: RunConfiguration{Duration: time.Hour}},
			error: "",
		},
		{
			name:  "negative duration",
			info:  ScenarioInfo{Configuration: RunConfiguration{Duration: -time.Second}},
			error: "invalid scenario: Duration cannot be negative",
		},
		{
			name:  "both duration and iterations",
			info:  ScenarioInfo{Configuration: RunConfiguration{Duration: 3 * time.Second, Iterations: 3}},
			error: "invalid scenario: iterations and duration are mutually exclusive",
		},
		{
			name:  "both duration and start iteration",
			info:  ScenarioInfo{Configuration: RunConfiguration{Duration: 3 * time.Second, StartFromIteration: 3}},
			error: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.info.MetricsHandler = client.MetricsNopHandler
			executor := &GenericExecutor{DefaultConfiguration: tt.info.Configuration}
			_, err := executor.newRun(tt.info)
			if tt.error != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.error)
			} else {
				require.NoError(t, err)
			}
		})
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
	i.Lock()
	defer i.Unlock()
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
	synctest.Test(t, func(t *testing.T) {
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
			},
			DefaultConfiguration: RunConfiguration{MaxConcurrent: concurrency, Iterations: 50},
		})
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
			},
			DefaultConfiguration: RunConfiguration{Duration: 100 * time.Millisecond},
		})
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
			},
			DefaultConfiguration: RunConfiguration{Duration: 200 * time.Millisecond},
		})
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
			},
			DefaultConfiguration: RunConfiguration{
				Duration: 100 * time.Millisecond,
				Timeout:  10 * time.Millisecond,
			},
		})
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
			},
			DefaultConfiguration: RunConfiguration{
				Iterations: 5,
				Timeout:    10 * time.Millisecond,
			},
		})
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
			},
			DefaultConfiguration: RunConfiguration{Duration: 1 * time.Millisecond},
		})
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
			},
			DefaultConfiguration: RunConfiguration{Iterations: 5},
		})
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
			},
			DefaultConfiguration: RunConfiguration{
				Iterations:             4,
				MaxConcurrent:          1,
				MaxIterationsPerSecond: 4.0,
			},
		})
		require.NoError(t, err)
		require.Equal(t, time.Second, time.Since(startTime))
		tracker.assertSeen(t, 4)
	})
}
