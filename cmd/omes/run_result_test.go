package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
)

func TestRunResultEncoding(t *testing.T) {
	result := runResult{
		Outcome: "failed",
		Error:   "iteration failed",
		Iterations: &loadgen.IterationCounts{
			Started: 2, Succeeded: 1, Failed: 1,
		},
		Failures:      []loadgen.IterationFailure{{Iteration: 2, Attempt: 6, Error: "failure"}},
		ScenarioState: map[string]int{"completed": 1},
	}

	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.JSONEq(t, `{
  "outcome": "failed",
  "error": "iteration failed",
  "iterations": {"started": 2, "succeeded": 1, "failed": 1},
  "failures": [{"iteration": 2, "attempt": 6, "error": "failure"}],
  "scenarioState": {"completed": 1}
}`, string(encoded))
}

func TestRunResultOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name           string
		fail           bool
		iterationError error
		timeout        time.Duration
		cancel         bool
		want           string
		wantIterations loadgen.IterationCounts
		wantFailure    string
	}{
		{
			name:           "pass",
			want:           "passed",
			wantIterations: loadgen.IterationCounts{Started: 1, Succeeded: 1},
		},
		{
			name:           "iteration failure",
			fail:           true,
			want:           "failed",
			wantIterations: loadgen.IterationCounts{Started: 1, Failed: 1},
			wantFailure:    "iteration 1 failed: failed iteration",
		},
		{
			name:           "iteration deadline",
			iterationError: context.DeadlineExceeded,
			want:           "failed",
			wantIterations: loadgen.IterationCounts{Started: 1, Failed: 1},
			wantFailure:    "iteration 1 failed: context deadline exceeded",
		},
		{
			name:           "iteration cancellation",
			iterationError: context.Canceled,
			want:           "failed",
			wantIterations: loadgen.IterationCounts{Started: 1, Failed: 1},
			wantFailure:    "iteration 1 failed: context canceled",
		},
		{
			name:           "timeout",
			timeout:        time.Second,
			want:           "timedOut",
			wantIterations: loadgen.IterationCounts{Started: 1, Failed: 1},
			wantFailure:    "iteration 1 failed: context deadline exceeded",
		},
		{
			name:           "interruption",
			cancel:         true,
			want:           "interrupted",
			wantIterations: loadgen.IterationCounts{Started: 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				path := filepath.Join(t.TempDir(), "result.json")
				runner := &scenarioRunner{
					scenarioRunConfig: scenarioRunConfig{exportResultFile: path},
					statistics:        &loadgen.RunStatistics{},
				}
				executor := &loadgen.GenericExecutor{Execute: func(ctx context.Context, _ *loadgen.Run) error {
					if tc.cancel {
						cancel()
					}

					if tc.timeout > 0 || tc.cancel {
						<-ctx.Done()
						return ctx.Err()
					}

					if tc.fail {
						return errors.New("failed iteration")
					}

					return tc.iterationError
				}}

				err := executor.Run(ctx, loadgen.ScenarioInfo{
					Logger:         zap.NewNop().Sugar(),
					MetricsHandler: client.MetricsNopHandler,
					Configuration: loadgen.RunConfiguration{
						Iterations: 1,
						Timeout:    tc.timeout,
						Statistics: runner.statistics,
					},
				})
				runner.finish(ctx, &err)

				if tc.want == "passed" {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}

				data, readErr := os.ReadFile(path)
				require.NoError(t, readErr)
				var result runResult
				require.NoError(t, json.Unmarshal(data, &result))
				require.Equal(t, tc.want, result.Outcome)
				require.Equal(t, &tc.wantIterations, result.Iterations)
				failures := []loadgen.IterationFailure{}
				if tc.wantFailure != "" {
					failures = append(failures, loadgen.IterationFailure{Iteration: 1, Attempt: 1, Error: tc.wantFailure})
				}
				require.Equal(t, failures, result.Failures)
			})
		})
	}
}

func TestRunResultOnInvalidInput(t *testing.T) {
	for _, worker := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := newRunner("no_such_scenario")
		runner.exportResultFile = path

		var err error
		if worker {
			r := workerWithScenarioRunner{scenarioRunConfig: runner.scenarioRunConfig}
			r.ScenarioID = runner.scenario
			err = r.run(context.Background())
		} else {
			err = runner.run(context.Background())
		}

		require.Error(t, err)

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		var result runResult
		require.NoError(t, json.Unmarshal(data, &result))
		require.Equal(t, "failed", result.Outcome)
		require.Equal(t, err.Error(), result.Error)
	}
}

func TestRunResultWriteErrorPreservesRunError(t *testing.T) {
	for _, runErr := range []error{nil, errors.New("run failed")} {
		path := filepath.Join(t.TempDir(), "missing", "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
		err := runErr
		runner.finish(context.Background(), &err)

		require.ErrorContains(t, err, "writing result file")
		if runErr != nil {
			require.ErrorIs(t, err, runErr)
		}
	}
}

func TestRunResultRenameFailureCleansUp(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "result.json")
	require.NoError(t, os.Mkdir(path, 0700))
	runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
	var err error
	runner.finish(context.Background(), &err)

	require.ErrorContains(t, err, "writing result file")

	files, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.True(t, files[0].IsDir())
}

func TestRunResultScenarioState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(path, []byte("old result"), 0600))
	runner := &scenarioRunner{
		scenarioRunConfig: scenarioRunConfig{exportResultFile: path},
		scenarioState:     map[string]int{"completed": 12},
	}
	var runErr error
	runner.finish(context.Background(), &runErr)
	require.NoError(t, runErr)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{
  "outcome": "passed",
  "iterations": null,
  "failures": [],
  "scenarioState": {"completed": 12}
}`, string(data))

	files, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestResultFileFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{runScenarioCmd(), runScenarioWithWorkerCmd()} {
		require.NotNil(t, cmd.Flags().Lookup("export-result-file"))

		require.NoError(t, cmd.ParseFlags([]string{"--export-result-file", "result.json", "--export-result-interval", "1s"}))
	}
}

func TestRunResultOnConnectionFailure(t *testing.T) {
	runner := newRunner("workflow_with_single_noop_activity")
	runner.logger = zap.NewNop().Sugar()
	runner.exportResultFile = filepath.Join(t.TempDir(), "result.json")
	runner.clientOptions.EnableTLS = true
	runner.clientOptions.ClientCertPath = filepath.Join(t.TempDir(), "missing-cert")
	runner.clientOptions.ClientKeyPath = filepath.Join(t.TempDir(), "missing-key")

	err := runner.run(context.Background())

	require.ErrorContains(t, err, "failed dialing")

	data, readErr := os.ReadFile(runner.exportResultFile)
	require.NoError(t, readErr)
	var result runResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.Equal(t, "failed", result.Outcome)
	require.Equal(t, err.Error(), result.Error)
}

func TestRunResultInvalidRate(t *testing.T) {
	for _, rate := range []float64{math.NaN(), math.Inf(1)} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{
			scenarioRunConfig: scenarioRunConfig{exportResultFile: path},
			statistics:        &loadgen.RunStatistics{},
		}
		executor := &loadgen.GenericExecutor{}

		err := executor.Run(context.Background(), loadgen.ScenarioInfo{Configuration: loadgen.RunConfiguration{
			MaxIterationsPerSecond: rate,
			Statistics:             runner.statistics,
		}})
		runner.finish(context.Background(), &err)

		require.ErrorContains(t, err, "invalid scenario")

		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		var result runResult
		require.NoError(t, json.Unmarshal(data, &result))
		require.Equal(t, "failed", result.Outcome)
		require.Contains(t, result.Error, "MaxIterationsPerSecond")
	}
}

func TestRunResultOnWorkerStartupFailure(t *testing.T) {
	runner := workerWithScenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: filepath.Join(t.TempDir(), "result.json")}}
	runner.ScenarioID = clioptions.ScenarioID{
		Scenario: "workflow_with_single_noop_activity",
		RunID:    "test-run",
	}
	runner.Logger = zap.NewNop().Sugar()
	runner.TaskQueueIndexSuffixStart = 2
	runner.TaskQueueIndexSuffixEnd = 1

	err := runner.run(context.Background())

	require.ErrorContains(t, err, "worker did not start")

	data, readErr := os.ReadFile(runner.exportResultFile)
	require.NoError(t, readErr)
	var result runResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.Equal(t, "failed", result.Outcome)
	require.Equal(t, err.Error(), result.Error)
}

func TestRunResultOnPanic(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second}}
		scenario := loadgen.GetScenario("ebb_and_flow")
		options, err := scenario.ResolveOptions(map[string]string{"control-interval": "0"})
		require.NoError(t, err)
		executor := scenario.ExecutorFn()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if interrupted {
			cancel()
		}

		require.PanicsWithValue(t, "non-positive interval for NewTicker", func() {
			var runErr error
			defer runner.finish(ctx, &runErr)
			runErr = runner.runExecutor(ctx, executor, loadgen.ScenarioInfo{Options: options})
		})

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.JSONEq(t, `{
  "outcome": "failed",
  "error": "panic: non-positive interval for NewTicker",
  "iterations": null,
  "failures": [],
  "scenarioState": {"totalCompletedWorkflows": 0}
}`, string(data))
	}
}

func TestRunResultMinimalSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
	var runErr error
	runner.finish(context.Background(), &runErr)
	require.NoError(t, runErr)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.JSONEq(t, `{"outcome":"passed","iterations":null,"failures":[],"scenarioState":null}`, string(data))
}

func TestRunResultRunningSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{
			scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second},
			statistics:        &loadgen.RunStatistics{},
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		started := make(chan int)
		proceed := make(chan struct{})
		var completed atomic.Int64
		executor := resultSnapshotExecutor{
			Executor: &loadgen.GenericExecutor{Execute: func(ctx context.Context, run *loadgen.Run) error {
				select {
				case started <- run.Iteration:
				case <-ctx.Done():
					return ctx.Err()
				}
				select {
				case <-proceed:
				case <-ctx.Done():
					return ctx.Err()
				}
				completed.Add(1)
				if run.Iteration == 2 {
					return errors.New("iteration failed")
				}
				return nil
			}},
			snapshot: func() any { return map[string]int64{"completed": completed.Load()} },
		}
		done := make(chan error, 1)
		go func() {
			var err error
			func() {
				defer runner.finish(context.Background(), &err)
				err = runner.runExecutor(ctx, executor, loadgen.ScenarioInfo{
					Logger:         zap.NewNop().Sugar(),
					MetricsHandler: client.MetricsNopHandler,
					Configuration: loadgen.RunConfiguration{
						Iterations: 3, MaxConcurrent: 1,
						ContinueOnIterationFailure:    true,
						DoNotRegisterSearchAttributes: true,
						Statistics:                    runner.statistics,
					},
				})
			}()
			done <- err
		}()
		for i, counts := range []loadgen.IterationCounts{
			{Started: 1},
			{Started: 2, Succeeded: 1},
			{Started: 3, Succeeded: 1, Failed: 1},
		} {
			require.Equal(t, i+1, <-started)
			for range 2 {
				<-time.After(time.Second)
				synctest.Wait()
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				var result runResult
				require.NoError(t, json.Unmarshal(data, &result))
				failures := []loadgen.IterationFailure{}
				if i == 2 {
					failures = append(failures, loadgen.IterationFailure{Iteration: 2, Attempt: 1, Error: "iteration 2 failed: iteration failed"})
				}
				require.Equal(t, runResult{
					Outcome: "running", Iterations: &counts, Failures: failures,
					ScenarioState: map[string]any{"completed": float64(i)},
				}, result)
				var fields map[string]any
				require.NoError(t, json.Unmarshal(data, &fields))
				require.Len(t, fields, 4)
			}
			proceed <- struct{}{}
		}
		require.Error(t, <-done)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		var result runResult
		require.NoError(t, json.Unmarshal(data, &result))
		require.Equal(t, "failed", result.Outcome)
		require.Equal(t, &loadgen.IterationCounts{Started: 3, Succeeded: 2, Failed: 1}, result.Iterations)
		require.Equal(t, map[string]any{"completed": float64(3)}, result.ScenarioState)
		<-time.After(10 * time.Second)
		synctest.Wait()
		final, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, data, final)
	})
}

type resultSnapshotExecutor struct {
	loadgen.Executor
	snapshot func() any
}

func (e resultSnapshotExecutor) Snapshot() any { return e.snapshot() }

func TestRunResultFinalWriteWaitsForSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second}}
		entered := make(chan struct{})
		release := make(chan struct{})
		defer close(release)
		finishRun := make(chan struct{})
		blocked := false
		executor := resultSnapshotExecutor{
			Executor: loadgen.ExecutorFunc(func(context.Context, loadgen.ScenarioInfo) error {
				<-finishRun
				return nil
			}),
			snapshot: func() any {
				if !blocked {
					blocked = true
					close(entered)
					<-release
				}
				return map[string]int{"completed": 1}
			},
		}
		done := make(chan error, 1)
		go func() {
			var err error
			func() {
				defer runner.finish(context.Background(), &err)
				err = runner.runExecutor(context.Background(), executor, loadgen.ScenarioInfo{})
			}()
			done <- err
		}()
		<-entered
		close(finishRun)
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("final write did not wait for the running snapshot")
		default:
		}
		_, err := os.Stat(path)
		require.ErrorIs(t, err, os.ErrNotExist)
		release <- struct{}{}
		require.NoError(t, <-done)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.JSONEq(t, `{"outcome":"passed","iterations":null,"failures":[],"scenarioState":{"completed":1}}`, string(data))
	})
}

func TestRunResultSnapshotIntervalDisabled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
		err := runner.runExecutor(context.Background(), loadgen.ExecutorFunc(func(context.Context, loadgen.ScenarioInfo) error {
			<-time.After(time.Hour)
			_, err := os.Stat(path)
			require.ErrorIs(t, err, os.ErrNotExist)
			return nil
		}), loadgen.ScenarioInfo{})
		runner.finish(context.Background(), &err)
		require.NoError(t, err)
		require.FileExists(t, path)
	})
}

func TestRunResultSnapshotErrorPreservesRunError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing", "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second}}
		runErr := errors.New("run failed")
		err := runner.runExecutor(context.Background(), loadgen.ExecutorFunc(func(context.Context, loadgen.ScenarioInfo) error {
			<-time.After(time.Second)
			synctest.Wait()
			require.NoError(t, os.Mkdir(filepath.Dir(path), 0700))
			return runErr
		}), loadgen.ScenarioInfo{})
		runner.finish(context.Background(), &err)
		require.ErrorIs(t, err, runErr)
		require.ErrorContains(t, err, "writing result file")
		require.FileExists(t, path)
	})
}
