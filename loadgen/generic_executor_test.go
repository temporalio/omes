package loadgen

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	omesmetrics "github.com/temporalio/omes/metrics"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	logger := zap.NewNop()
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

func TestRunContinueOnIterationFailure(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var completed, failed []int

		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				if run.Iteration%2 == 0 {
					return errors.New("deliberate fail from test")
				}
				return nil
			}},
			RunConfiguration{
				Iterations:                 6,
				MaxConcurrent:              1,
				ContinueOnIterationFailure: true,
				OnCompletion: func(ctx context.Context, run *Run) {
					mu.Lock()
					defer mu.Unlock()
					completed = append(completed, run.Iteration)
				},
				OnIterationFailure: func(ctx context.Context, run *Run, err error) {
					mu.Lock()
					defer mu.Unlock()
					failed = append(failed, run.Iteration)
				},
			},
		)

		// Every iteration runs (tolerated failures don't abort), while library
		// callers still receive a structured degraded-run verdict.
		var failures *IterationFailuresError
		require.ErrorAs(t, err, &failures)
		require.Equal(t, int64(6), failures.Attempted)
		require.Equal(t, int64(3), failures.Succeeded)
		require.Equal(t, int64(3), failures.Failed)
		mu.Lock()
		defer mu.Unlock()
		require.ElementsMatch(t, []int{1, 3, 5}, completed)
		require.ElementsMatch(t, []int{2, 4, 6}, failed)
	})
}

func TestRunContinueOnIterationFailureDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var attempts atomic.Int64
		err := execute(&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				time.Sleep(10 * time.Millisecond)
				attempts.Add(1)
				if run.Iteration%2 == 0 {
					return errors.New("deliberate fail from test")
				}
				return nil
			}},
			RunConfiguration{
				Duration:                   50 * time.Millisecond,
				MaxConcurrent:              1,
				ContinueOnIterationFailure: true,
			},
		)

		var failures *IterationFailuresError
		require.ErrorAs(t, err, &failures)
		require.Equal(t, attempts.Load(), failures.Attempted)
		require.Positive(t, failures.Succeeded)
		require.Positive(t, failures.Failed)
	})
}

func TestRunCommitsFailureOutcomeBeforeReturning(t *testing.T) {
	for _, continueOnFailure := range []bool{false, true} {
		name := "fail-fast"
		if continueOnFailure {
			name = "continue"
		}
		t.Run(name, func(t *testing.T) {
			callbackStarted := make(chan struct{})
			releaseCallback := make(chan struct{})
			var releaseOnce sync.Once
			t.Cleanup(func() { releaseOnce.Do(func() { close(releaseCallback) }) })

			runDone := make(chan error, 1)
			go func() {
				runDone <- (&GenericExecutor{
					Execute: func(context.Context, *Run) error {
						return errors.New("terminal failure")
					},
				}).Run(context.Background(), ScenarioInfo{
					MetricsHandler: client.MetricsNopHandler,
					Logger:         zap.NewNop().Sugar(),
					Configuration: RunConfiguration{
						Iterations:                 1,
						MaxConcurrent:              1,
						ContinueOnIterationFailure: continueOnFailure,
						OnIterationFailure: func(context.Context, *Run, error) {
							close(callbackStarted)
							<-releaseCallback
						},
					},
				})
			}()

			select {
			case <-callbackStarted:
			case <-time.After(time.Second):
				t.Fatal("iteration failure callback did not start")
			}

			select {
			case err := <-runDone:
				t.Fatalf("run returned before failure bookkeeping completed: %v", err)
			case <-time.After(20 * time.Millisecond):
			}

			releaseOnce.Do(func() { close(releaseCallback) })
			select {
			case err := <-runDone:
				if continueOnFailure {
					var failures *IterationFailuresError
					require.ErrorAs(t, err, &failures)
					require.Equal(t, int64(1), failures.Failed)
				} else {
					require.ErrorContains(t, err, "run finished with error")
				}
			case <-time.After(time.Second):
				t.Fatal("run did not return after failure bookkeeping completed")
			}
		})
	}
}

func TestIterationStatusCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want codes.Code
	}{
		{name: "wrapped canceled context", err: fmt.Errorf("start failed: %w", context.Canceled), want: codes.Canceled},
		{name: "wrapped deadline context", err: fmt.Errorf("start failed: %w", context.DeadlineExceeded), want: codes.DeadlineExceeded},
		{name: "wrapped grpc status", err: fmt.Errorf("start failed: %w", status.Error(codes.ResourceExhausted, "busy")), want: codes.ResourceExhausted},
		{name: "plain error", err: errors.New("plain"), want: codes.Unknown},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, iterationStatusCode(test.err))
		})
	}
}

func TestRunRecordsTerminalIterationOutcomes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		registry := prometheus.NewRegistry()
		metrics := &omesmetrics.Metrics{
			Registry: registry,
			Cache:    make(map[string]any),
		}
		logger := zap.NewNop()
		err := (&GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				if run.Iteration%2 == 0 {
					return fmt.Errorf("workflow start failed: %w", status.Error(codes.Unavailable, "down"))
				}
				return nil
			},
		}).Run(context.Background(), ScenarioInfo{
			ScenarioName:   "metric-test",
			MetricsHandler: metrics.NewHandler(),
			Logger:         logger.Sugar(),
			Configuration: RunConfiguration{
				Iterations:                 4,
				MaxConcurrent:              1,
				ContinueOnIterationFailure: true,
			},
		})

		var failures *IterationFailuresError
		require.ErrorAs(t, err, &failures)
		families, gatherErr := registry.Gather()
		require.NoError(t, gatherErr)

		outcomes := make(map[string]float64)
		for _, family := range families {
			if family.GetName() != iterationsMetricName {
				continue
			}
			for _, metric := range family.Metric {
				labels := make(map[string]string)
				for _, label := range metric.Label {
					labels[label.GetName()] = label.GetValue()
				}
				require.Equal(t, "metric-test", labels["scenario"])
				outcomes[labels["outcome"]+"/"+labels["status_code"]] = metric.Counter.GetValue()
			}
		}

		require.Equal(t, float64(2), outcomes["succeeded/OK"])
		require.Equal(t, float64(2), outcomes["failed/Unavailable"])
	})
}

// TestRunStoppedIterationsAreNotCountedAsFailures pins that iterations abandoned
// by a caller stopping the run are left out of the tallies, so a clean stop is
// not reported as a burst of failures.
func TestRunStoppedIterationsAreNotCountedAsFailures(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var mu sync.Mutex
		var failed, completed []int

		const concurrent = 5
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var inFlight int
		executor := &GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				mu.Lock()
				inFlight++
				full := inFlight == concurrent
				mu.Unlock()

				// Stop the run once every slot is occupied, so every in-flight
				// iteration ends on the stop rather than on its own outcome.
				if full {
					cancel()
				}
				<-ctx.Done()
				return ctx.Err()
			},
		}

		logger := zap.Must(zap.NewDevelopment())
		defer logger.Sync()
		err := executor.Run(ctx, ScenarioInfo{
			MetricsHandler: client.MetricsNopHandler,
			Logger:         logger.Sugar(),
			Configuration: RunConfiguration{
				Iterations:                 100,
				MaxConcurrent:              concurrent,
				ContinueOnIterationFailure: true,
				OnCompletion: func(ctx context.Context, run *Run) {
					mu.Lock()
					defer mu.Unlock()
					completed = append(completed, run.Iteration)
				},
				OnIterationFailure: func(ctx context.Context, run *Run, err error) {
					mu.Lock()
					defer mu.Unlock()
					failed = append(failed, run.Iteration)
				},
			},
		})

		// Stopping a run is still an error for the caller to interpret; what must
		// not happen is the stop being reported as failed iterations.
		require.Error(t, err)
		require.NotContains(t, err.Error(), "iterations failed",
			"a stopped run must not report the end-of-run failure verdict")

		mu.Lock()
		defer mu.Unlock()
		require.Empty(t, failed, "iterations abandoned by the stop must not be counted as failures")
		require.Empty(t, completed, "nor as successes — they did not finish")
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
