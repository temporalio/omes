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

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/load"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestRunLoadPatterns(t *testing.T) {
	for _, tc := range []struct {
		name     string
		peak     float64
		duration time.Duration
		params   map[string]string
		want     []int
	}{
		{"ramp-hold-ramp", 8, 4 * time.Second, map[string]string{"up": "1", "down": "0s"}, []int{0, 3, 5, 7}},
		{"sweep", 4, 3 * time.Second, map[string]string{"from": "0.5", "steps": "3"}, []int{1, 3, 4}},
		{"spike-train", 4, 8 * time.Second, map[string]string{"base": "0", "every": "3s", "for": "1s"}, []int{0, 0, 0, 3, 0, 0, 3, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				pattern, err := load.Compile(tc.name, tc.peak, tc.duration, tc.params)
				require.NoError(t, err)
				start := time.Now()
				var mu sync.Mutex
				windows := make([]int, len(tc.want))
				executor := &GenericExecutor{Execute: func(context.Context, *Run) error {
					mu.Lock()
					defer mu.Unlock()
					windows[int(time.Since(start)/time.Second)]++
					return nil
				}}

				err = execute(executor, RunConfiguration{Duration: tc.duration, LoadPattern: pattern})
				require.NoError(t, err)
				require.Equal(t, tc.want, windows)
				require.Equal(t, tc.duration, time.Since(start))
			})
		})
	}
}

func TestRunLegacyRateWaitsUnderSaturation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.DebugLevel)
		start := time.Now()
		var mu sync.Mutex
		var starts []time.Duration
		executor := &GenericExecutor{Execute: func(_ context.Context, run *Run) error {
			mu.Lock()
			starts = append(starts, time.Since(start))
			mu.Unlock()
			if run.Iteration == 1 {
				time.Sleep(2250 * time.Millisecond)
			}
			return nil
		}}
		config := RunConfiguration{Iterations: 5, MaxConcurrent: 1, MaxIterationsPerSecond: 2, DoNotRegisterSearchAttributes: true}

		err := executor.Run(context.Background(), ScenarioInfo{Configuration: config, Logger: zap.New(core).Sugar(), MetricsHandler: client.MetricsNopHandler})

		require.NoError(t, err)
		require.Equal(t, []time.Duration{500 * time.Millisecond, 2750 * time.Millisecond, 2750 * time.Millisecond, 3 * time.Second, 3500 * time.Millisecond}, starts)
		require.Empty(t, logs.FilterLevelExact(zap.WarnLevel).All())
	})
}

func TestRunLoadDropsWithoutCatchUp(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		core, logs := observer.New(zap.InfoLevel)
		start := time.Now()
		var mu sync.Mutex
		var starts []time.Duration
		executor := &GenericExecutor{Execute: func(_ context.Context, run *Run) error {
			mu.Lock()
			starts = append(starts, time.Since(start))
			mu.Unlock()
			if run.Iteration == 1 {
				time.Sleep(2250 * time.Millisecond)
			}
			return nil
		}}
		pattern, err := load.Compile("steady", 2, 5*time.Second, nil)
		require.NoError(t, err)
		config := RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true}

		err = executor.Run(context.Background(), ScenarioInfo{Configuration: config, Logger: zap.New(core).Sugar(), MetricsHandler: client.MetricsNopHandler})

		require.NoError(t, err)
		require.Equal(t, []time.Duration{500 * time.Millisecond, 3 * time.Second, 3500 * time.Millisecond, 4 * time.Second, 4500 * time.Millisecond}, starts)
		warnings := logs.FilterMessage("Load phase dropped starts; raise --max-concurrent or lower peak= in --load").All()
		require.Len(t, warnings, 1)
		require.Equal(t, map[string]interface{}{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(4)}, warnings[0].ContextMap())
	})
}

func TestRunLoadSkipsSlotsWhileLoopBlocked(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pattern  string
		duration time.Duration
		message  string
		blocked  time.Duration
		want     []time.Duration
		drops    []map[string]interface{}
	}{
		{
			name: "busy loop", pattern: "steady", duration: 5 * time.Second,
			message: "Running iteration 0", blocked: 2250 * time.Millisecond,
			want:  []time.Duration{2750 * time.Millisecond, 3 * time.Second, 3500 * time.Millisecond, 4 * time.Second, 4500 * time.Millisecond},
			drops: []map[string]interface{}{{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(4)}},
		},
		{
			name: "late timer", pattern: "steady", duration: 5 * time.Second,
			message: "Load phase", blocked: 2250 * time.Millisecond,
			want:  []time.Duration{2750 * time.Millisecond, 3 * time.Second, 3500 * time.Millisecond, 4 * time.Second, 4500 * time.Millisecond},
			drops: []map[string]interface{}{{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(4)}},
		},
		{
			name: "crossed phases", pattern: "sweep", duration: 3 * time.Second,
			message: "Running iteration 0", blocked: 2250 * time.Millisecond,
			want: []time.Duration{2750 * time.Millisecond},
			drops: []map[string]interface{}{
				{"phase": "step-2", "phaseIndex": int64(2), "dropped": int64(3)},
				{"phase": "step-3", "phaseIndex": int64(3), "dropped": int64(4)},
			},
		},
		{
			name: "duration elapsed", pattern: "steady", duration: 5 * time.Second,
			message: "Load phase", blocked: 6 * time.Second,
			drops: []map[string]interface{}{{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(9)}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				core, logs := observer.New(zap.DebugLevel)
				logger := zap.New(core, zap.Hooks(func(entry zapcore.Entry) error {
					if entry.Message == tc.message {
						time.Sleep(tc.blocked)
					}
					return nil
				})).Sugar()
				start := time.Now()
				var mu sync.Mutex
				var starts []time.Duration
				executor := &GenericExecutor{Execute: func(context.Context, *Run) error {
					mu.Lock()
					defer mu.Unlock()
					starts = append(starts, time.Since(start))
					return nil
				}}
				params := map[string]string(nil)
				peak := float64(2)
				if tc.pattern == "sweep" {
					params = map[string]string{"from": "0.5", "steps": "3"}
					peak = 4
				}
				pattern, err := load.Compile(tc.pattern, peak, tc.duration, params)
				require.NoError(t, err)
				config := RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true}

				err = executor.Run(context.Background(), ScenarioInfo{Configuration: config, Logger: logger, MetricsHandler: client.MetricsNopHandler})

				require.NoError(t, err)
				require.Equal(t, tc.want, starts)
				var drops []map[string]interface{}
				for _, entry := range logs.FilterMessage("Load phase dropped starts; scheduling loop was late; lower peak= in --load").All() {
					drops = append(drops, entry.ContextMap())
				}
				require.Equal(t, tc.drops, drops)
			})
		})
	}
}

func TestRunLogsPhaseChangesAndDrops(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pattern, err := load.Compile("sweep", 4, 3*time.Second, map[string]string{"from": "0.5", "steps": "3"})
		require.NoError(t, err)
		core, logs := observer.New(zap.InfoLevel)
		executor := &GenericExecutor{Execute: func(_ context.Context, run *Run) error {
			if run.Iteration == 1 {
				<-time.After(1600 * time.Millisecond)
			}
			return nil
		}}

		err = executor.Run(context.Background(), ScenarioInfo{
			Configuration: RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true},
			Logger:        zap.New(core).Sugar(), MetricsHandler: client.MetricsNopHandler,
		})
		require.NoError(t, err)
		var transitions []map[string]interface{}
		for _, entry := range logs.FilterMessage("Load phase").All() {
			transitions = append(transitions, entry.ContextMap())
		}
		require.Equal(t, []map[string]interface{}{
			{"phase": "step-1", "phaseIndex": int64(1), "requestedIterationsPerSecond": float64(2)},
			{"phase": "step-2", "phaseIndex": int64(2), "requestedIterationsPerSecond": float64(3)},
			{"phase": "step-3", "phaseIndex": int64(3), "requestedIterationsPerSecond": float64(4)},
		}, transitions)
		var drops []map[string]interface{}
		for _, entry := range logs.FilterMessage("Load phase dropped starts; raise --max-concurrent or lower peak= in --load").All() {
			drops = append(drops, entry.ContextMap())
		}
		require.Equal(t, []map[string]interface{}{
			{"phase": "step-2", "phaseIndex": int64(2), "dropped": int64(3)},
			{"phase": "step-3", "phaseIndex": int64(3), "dropped": int64(1)},
		}, drops)
		require.Equal(t, map[string]interface{}{
			"pattern": "sweep,peak=4/s,from=0.5,to=1,steps=3", "peak": float64(4), "duration": 3 * time.Second,
			"phases": []interface{}{"step-1 [0s,1s) 2→2 iterations/s", "step-2 [1s,2s) 3→3 iterations/s", "step-3 [2s,3s) 4→4 iterations/s"},
		}, logs.FilterMessage("Load pattern").All()[0].ContextMap())
	})
}

func TestRunZeroLoadWaitsForDuration(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pattern, err := load.Compile("spike-train", 2, 2*time.Second, map[string]string{"base": "0", "every": "3s", "for": "1s"})
		require.NoError(t, err)
		var started atomic.Int64
		start := time.Now()
		executor := &GenericExecutor{Execute: func(context.Context, *Run) error { started.Add(1); return nil }}

		err = execute(executor, RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern})
		require.NoError(t, err)
		require.Equal(t, int64(0), started.Load())
		require.Equal(t, 2*time.Second, time.Since(start))
	})
}

func TestRunReportsDropsBeforeCooldownCompletes(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pattern, err := load.Compile("steady", 2, 2*time.Second, nil)
		require.NoError(t, err)
		core, logs := observer.New(zap.InfoLevel)
		release := make(chan struct{})
		done := make(chan error, 1)
		executor := &GenericExecutor{Execute: func(context.Context, *Run) error { <-release; return nil }}
		go func() {
			done <- executor.Run(context.Background(), ScenarioInfo{
				Configuration: RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true},
				Logger:        zap.New(core).Sugar(), MetricsHandler: client.MetricsNopHandler,
			})
		}()

		<-time.After(2 * time.Second)
		synctest.Wait()
		warnings := logs.FilterMessage("Load phase dropped starts; raise --max-concurrent or lower peak= in --load").All()
		close(release)
		require.Len(t, warnings, 1)
		require.Equal(t, map[string]interface{}{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(2)}, warnings[0].ContextMap())
		require.NoError(t, <-done)
	})
}

func TestRunLegacyRateKeepsTickerInterval(t *testing.T) {
	for _, tc := range []struct {
		rate       float64
		duration   time.Duration
		iterations int
		want       []time.Duration
	}{
		{3, time.Second, 0, []time.Duration{333333333, 666666666, 999999999}},
		{7, 0, 3, []time.Duration{142857142, 285714284, 428571426}},
		{120, 0, 3, []time.Duration{8333333, 16666666, 24999999}},
		{0.3, 0, 3, []time.Duration{3333333333, 6666666666, 9999999999}},
		{1e9, 0, 3, []time.Duration{1, 2, 3}},
	} {
		t.Run(fmt.Sprint(tc.rate), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				var mu sync.Mutex
				var starts []time.Duration
				executor := &GenericExecutor{Execute: func(context.Context, *Run) error {
					mu.Lock()
					defer mu.Unlock()
					starts = append(starts, time.Since(start))
					return nil
				}}

				err := execute(executor, RunConfiguration{Duration: tc.duration, Iterations: tc.iterations, MaxIterationsPerSecond: tc.rate})
				require.NoError(t, err)
				require.Equal(t, tc.want, starts)
			})
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

func execute(executor *GenericExecutor, runConfig RunConfiguration) error {
	return executeContext(context.Background(), executor, runConfig)
}

func executeContext(ctx context.Context, executor *GenericExecutor, runConfig RunConfiguration) error {
	logger := zap.Must(zap.NewDevelopment())
	defer logger.Sync()
	info := ScenarioInfo{
		MetricsHandler: client.MetricsNopHandler,
		Logger:         logger.Sugar(),
		Configuration:  runConfig,
	}
	return executor.Run(ctx, info)
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

func TestRunWaitsForOnCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		callbackStarted := make(chan struct{})
		releaseCallback := make(chan struct{})
		runDone := make(chan error, 1)

		go func() {
			runDone <- execute(&GenericExecutor{
				Execute: func(ctx context.Context, run *Run) error {
					return nil
				}},
				RunConfiguration{
					Iterations: 1,
					OnCompletion: func(ctx context.Context, run *Run) {
						close(callbackStarted)
						<-releaseCallback
					},
				},
			)
		}()

		<-callbackStarted

		synctest.Wait()

		returned := false
		select {
		case <-runDone:
			returned = true
		default:
		}
		close(releaseCallback)

		require.False(t, returned, "executor returned before OnCompletion finished")
		require.NoError(t, <-runDone)
	})
}

func TestRunCancellationWaitsForCallbacks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		callbackStarted := make(chan struct{})
		releaseCallback := make(chan struct{})
		callbackFinished := make(chan struct{})
		runDone := make(chan error, 1)
		go func() {
			runDone <- executeContext(ctx, &GenericExecutor{
				Execute: func(context.Context, *Run) error { return nil },
			}, RunConfiguration{
				Iterations: 1,
				OnCompletion: func(context.Context, *Run) {
					close(callbackStarted)
					<-releaseCallback
					close(callbackFinished)
				},
			})
		}()
		<-callbackStarted
		cancel()
		synctest.Wait()
		var runErr error
		returned := false
		select {
		case runErr = <-runDone:
			returned = true
		default:
		}
		close(releaseCallback)
		if !returned {
			runErr = <-runDone
		}
		require.False(t, returned, "Run returned while its completion callback was still running")
		require.ErrorIs(t, runErr, context.Canceled)
		select {
		case <-callbackFinished:
		default:
			t.Fatal("Run returned before the completion callback finished")
		}
	})
}

func TestRunCancellationClassifiesIterationErrors(t *testing.T) {
	for _, tc := range []struct {
		name           string
		iterationError error
		stopRun        bool
		wantFailed     int
	}{
		{"context stop", context.Canceled, true, 0},
		{"service stop", fmt.Errorf("waiting for workflow: %w", serviceerror.NewCanceled("stopped")), true, 0},
		{"service cancellation while running", serviceerror.NewCanceled("workflow canceled"), false, 1},
		{"failure during stop", errors.New("workflow failed"), true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var failed atomic.Int64
				err := executeContext(ctx, &GenericExecutor{
					Execute: func(context.Context, *Run) error {
						if tc.stopRun {
							cancel()
						}
						return tc.iterationError
					},
				}, RunConfiguration{
					Iterations:         1,
					OnIterationFailure: func(context.Context, *Run, error) { failed.Add(1) },
				})
				require.Error(t, err)
				require.EqualValues(t, tc.wantFailed, failed.Load())
			})
		})
	}
}

func TestRunErrorWaitsForInFlightCallbacks(t *testing.T) {
	for _, timeout := range []time.Duration{0, time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				callbackStarted := make(chan struct{})
				releaseCallback := make(chan struct{})
				failIteration := make(chan struct{})
				runDone := make(chan error, 1)
				failure := errors.New("iteration failed")
				go func() {
					runDone <- execute(&GenericExecutor{
						Execute: func(ctx context.Context, run *Run) error {
							if run.Iteration == 1 {
								return nil
							}
							if timeout > 0 {
								<-ctx.Done()
								return ctx.Err()
							}
							<-failIteration
							return failure
						},
					}, RunConfiguration{
						Iterations: 2,
						Timeout:    timeout,
						OnCompletion: func(context.Context, *Run) {
							close(callbackStarted)
							<-releaseCallback
						},
					})
				}()
				<-callbackStarted
				close(failIteration)
				if timeout > 0 {
					time.Sleep(timeout)
				}
				synctest.Wait()
				var runErr error
				returned := false
				select {
				case runErr = <-runDone:
					returned = true
				default:
				}
				close(releaseCallback)
				if !returned {
					runErr = <-runDone
				}
				require.False(t, returned, "Run returned before an in-flight callback finished")
				if timeout > 0 {
					require.ErrorIs(t, runErr, context.DeadlineExceeded)
				} else {
					require.ErrorIs(t, runErr, failure)
				}
			})
		})
	}
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

func TestRunFailureHandling(t *testing.T) {
	t.Run("RateLimitedFailureStopsScheduling", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			failure := errors.New("operator check failed")
			failureTimes := make(chan time.Time, 3)
			var started atomic.Int64

			err := execute(&GenericExecutor{
				StopNewIterations: make(chan struct{}),
				Execute: func(context.Context, *Run) error {
					started.Add(1)
					return failure
				},
			}, RunConfiguration{
				MaxConcurrent:                 3,
				MaxIterationsPerSecond:        1.0 / 60,
				DoNotRegisterSearchAttributes: true,
				OnIterationFailure: func(context.Context, *Run, error) {
					failureTimes <- time.Now()
				},
			})

			require.ErrorIs(t, err, failure)
			require.Equal(t, time.Duration(0), time.Since(<-failureTimes), "must report failure without waiting for another rate tick")
			require.Equal(t, int64(1), started.Load(), "must not launch more tests after terminal failure")
		})
	})

	t.Run("PreservesFailureWhenCallbackCancels", func(t *testing.T) {
		for _, continueOnFailure := range []bool{false, true} {
			name := "stop-on-failure"
			if continueOnFailure {
				name = "continue-on-failure"
			}
			t.Run(name, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					failure := errors.New("cleanup RPC failed")

					err := executeContext(ctx, &GenericExecutor{
						Execute: func(context.Context, *Run) error {
							return failure
						},
					}, RunConfiguration{
						Iterations:                    1,
						MaxConcurrent:                 1,
						ContinueOnIterationFailure:    continueOnFailure,
						DoNotRegisterSearchAttributes: true,
						OnIterationFailure: func(context.Context, *Run, error) {
							cancel()
						},
					})

					require.ErrorIs(t, err, context.Canceled)
					require.ErrorIs(t, err, failure, "cancellation must not replace the iteration error")
				})
			})
		}
	})

	t.Run("PreservesHandledFailureOnCancellation", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			failure := errors.New("operator check failed")
			failed := make(chan struct{})
			done := make(chan error, 1)

			go func() {
				done <- executeContext(ctx, &GenericExecutor{
					StopNewIterations: make(chan struct{}),
					Execute: func(context.Context, *Run) error {
						return failure
					},
				}, RunConfiguration{
					MaxConcurrent:                 1,
					MaxIterationsPerSecond:        1.0 / 60,
					ContinueOnIterationFailure:    true,
					DoNotRegisterSearchAttributes: true,
					OnIterationFailure: func(context.Context, *Run, error) {
						close(failed)
					},
				})
			}()
			<-failed
			// Let the scheduler consume the failure and wait for its next rate tick.
			synctest.Wait()
			cancel()
			err := <-done

			require.ErrorIs(t, err, context.Canceled)
			require.ErrorIs(t, err, failure, "consuming a tolerated failure must not hide it on cancellation")
		})
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

		// Every iteration runs (tolerated failures don't abort), but the verdict is
		// unchanged: the run still fails because iterations failed.
		require.Error(t, err)
		require.Contains(t, err.Error(), "3 of 6 iterations failed")
		mu.Lock()
		defer mu.Unlock()
		require.ElementsMatch(t, []int{1, 3, 5}, completed)
		require.ElementsMatch(t, []int{2, 4, 6}, failed)
	})
}

func TestRunReportsNonCancellationFailureAfterCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		failureReported := make(chan struct{}, 1)
		err := executeContext(ctx, &GenericExecutor{
			Execute: func(ctx context.Context, run *Run) error {
				cancel()
				return errors.New("deliberate fail from test")
			},
		}, RunConfiguration{
			Iterations:                 1,
			ContinueOnIterationFailure: true,
			OnIterationFailure: func(ctx context.Context, run *Run, err error) {
				failureReported <- struct{}{}
			},
		})
		require.Error(t, err)

		synctest.Wait()

		reported := false
		select {
		case <-failureReported:
			reported = true
		default:
		}
		require.True(t, reported, "non-cancellation error should be reported as a failure")
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
		err := executeContext(ctx, &GenericExecutor{
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
		}, RunConfiguration{
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

func TestRunTerminalFailureReporting(t *testing.T) {
	for _, tc := range []struct {
		name               string
		tolerate, callback bool
		wantLogs           int
	}{
		{name: "propagated"},
		{name: "tolerated without callback", tolerate: true, wantLogs: 1},
		{name: "tolerated with callback", tolerate: true, callback: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.DebugLevel)
			cause := errors.New("terminal failure cause")
			config := RunConfiguration{Iterations: 1, MaxConcurrent: 1, MaxIterationAttempts: 1, ContinueOnIterationFailure: tc.tolerate}
			var callbackErr error
			if tc.callback {
				config.OnIterationFailure = func(_ context.Context, _ *Run, err error) { callbackErr = err }
			}
			executor := GenericExecutor{Execute: func(context.Context, *Run) error { return cause }}
			err := executor.Run(t.Context(), ScenarioInfo{Configuration: config, MetricsHandler: client.MetricsNopHandler, Logger: zap.New(core).Sugar()})
			require.Error(t, err)
			if !tc.tolerate {
				require.ErrorIs(t, err, cause)
			}
			if tc.callback {
				require.ErrorIs(t, callbackErr, cause)
			}
			require.Equal(t, tc.wantLogs, logs.FilterMessageSnippet(cause.Error()).Len())
		})
	}
}

func TestRunLoadEarlyStopAccounting(t *testing.T) {
	for _, reason := range []string{"failure", "cancellation", "stop channel"} {
		t.Run(reason, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				core, logs := observer.New(zap.InfoLevel)
				pattern, err := load.Compile("steady", 2, time.Minute, nil)
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				stop := make(chan struct{})
				failure := errors.New("iteration failed")
				executor := &GenericExecutor{StopNewIterations: stop, Execute: func(context.Context, *Run) error {
					time.Sleep(1250 * time.Millisecond)
					switch reason {
					case "cancellation":
						cancel()
					case "stop channel":
						close(stop)
					default:
						return failure
					}
					return nil
				}}

				err = executor.Run(ctx, ScenarioInfo{
					Configuration: RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true},
					Logger:        zap.New(core).Sugar(), MetricsHandler: client.MetricsNopHandler,
				})
				switch reason {
				case "failure":
					require.ErrorIs(t, err, failure)
				case "cancellation":
					require.ErrorIs(t, err, context.Canceled)
				default:
					require.NoError(t, err)
				}
				var drops []map[string]interface{}
				for _, entry := range logs.FilterMessageSnippet("Load phase dropped starts").All() {
					drops = append(drops, entry.ContextMap())
				}
				require.Equal(t, []map[string]interface{}{{"phase": "hold", "phaseIndex": int64(1), "dropped": int64(2)}}, drops)
			})
		})
	}
}
