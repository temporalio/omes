package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestRunResultEncoding(t *testing.T) {
	result := newRunResult(runEnd{finished: true, err: errors.New("iteration failed")}, loadgen.RunCounts{
		Iterations: &loadgen.IterationCounts{
			Started: 2, Succeeded: 1, Failed: 1,
		},
		Failures: []loadgen.IterationFailure{{Iteration: 2, Attempt: 6, Error: "failure"}},
	}, map[string]int{"completed": 1})

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
		name   string
		end    runEnd
		counts loadgen.RunCounts
		want   string
	}{
		{
			name: "zero value",
			want: "running",
		},
		{
			name: "pass",
			end:  runEnd{finished: true},
			want: "passed",
		},
		{
			name: "run failure",
			end:  runEnd{finished: true, err: errors.New("run failed")},
			want: "failed",
		},
		{
			name:   "iteration failure",
			end:    runEnd{finished: true},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}},
			want:   "failed",
		},
		{
			name:   "iteration deadline",
			end:    runEnd{finished: true, err: context.DeadlineExceeded},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}},
			want:   "failed",
		},
		{
			name:   "iteration cancellation",
			end:    runEnd{finished: true, err: context.Canceled},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}},
			want:   "failed",
		},
		{
			name:   "timeout",
			end:    runEnd{finished: true, err: context.DeadlineExceeded},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}, TimedOut: true},
			want:   "timedOut",
		},
		{
			name:   "interruption",
			end:    runEnd{finished: true, ctxErr: context.Canceled, err: context.Canceled},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1}},
			want:   "interrupted",
		},
		{
			name: "context deadline",
			end:  runEnd{finished: true, ctxErr: context.DeadlineExceeded},
			want: "timedOut",
		},
		{
			name: "deadline before execution",
			end:  runEnd{finished: true, err: context.DeadlineExceeded},
			want: "timedOut",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := newRunResult(tc.end, tc.counts, nil)
			require.Equal(t, tc.want, result.Outcome)
			if tc.end.err != nil {
				require.Equal(t, tc.end.err.Error(), result.Error)
			} else {
				require.Empty(t, result.Error)
			}
		})
	}
}

func TestRunResultOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name      string
		run       func(context.Context, string) error
		wantError string
	}{
		{
			name: "invalid input",
			run: func(ctx context.Context, path string) error {
				runner := newRunner("no_such_scenario")
				runner.exportResultFile = path
				return runner.run(ctx)
			},
			wantError: "no_such_scenario",
		},
		{
			name: "invalid input with worker",
			run: func(ctx context.Context, path string) error {
				runner := workerWithScenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
				runner.ScenarioID = newRunner("no_such_scenario").scenario
				return runner.run(ctx)
			},
			wantError: "no_such_scenario",
		},
		{
			name: "connection failure",
			run: func(ctx context.Context, path string) error {
				runner := newRunner("workflow_with_single_noop_activity")
				runner.logger = zap.NewNop().Sugar()
				runner.exportResultFile = path
				runner.clientOptions.EnableTLS = true
				runner.clientOptions.ClientCertPath = filepath.Join(filepath.Dir(path), "missing-cert")
				runner.clientOptions.ClientKeyPath = filepath.Join(filepath.Dir(path), "missing-key")
				return runner.run(ctx)
			},
			wantError: "failed dialing",
		},
		{
			name: "worker startup failure",
			run: func(ctx context.Context, path string) error {
				runner := workerWithScenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
				runner.ScenarioID = clioptions.ScenarioID{
					Scenario: "workflow_with_single_noop_activity",
					RunID:    "test-run",
				}
				runner.Logger = zap.NewNop().Sugar()
				runner.TaskQueueIndexSuffixStart = 2
				runner.TaskQueueIndexSuffixEnd = 1
				return runner.run(ctx)
			},
			wantError: "worker did not start",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			err := tc.run(context.Background(), path)
			require.ErrorContains(t, err, tc.wantError)
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			var result runResult
			require.NoError(t, json.Unmarshal(data, &result))
			require.Equal(t, runResult{
				Outcome:  "failed",
				Error:    err.Error(),
				Failures: []loadgen.IterationFailure{},
			}, result)
		})
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

func TestRunResultFile(t *testing.T) {
	for _, tc := range []struct {
		name          string
		runErr        error
		scenarioState any
		want          string
	}{
		{
			name: "minimal schema",
			want: `{"outcome":"passed","iterations":null,"failures":[],"scenarioState":null}`,
		},
		{
			name:          "scenario state",
			scenarioState: map[string]int{"completed": 12},
			want:          `{"outcome":"passed","iterations":null,"failures":[],"scenarioState":{"completed":12}}`,
		},
		{
			name:   "run error",
			runErr: errors.New("run failed"),
			want:   `{"outcome":"failed","error":"run failed","iterations":null,"failures":[],"scenarioState":null}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			require.NoError(t, os.WriteFile(path, []byte("old result"), 0600))
			runner := &scenarioRunner{
				scenarioRunConfig: scenarioRunConfig{exportResultFile: path},
				scenarioState:     tc.scenarioState,
			}
			runErr := tc.runErr
			runner.finish(context.Background(), &runErr)
			require.Equal(t, tc.runErr, runErr)
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))

			files, err := os.ReadDir(filepath.Dir(path))
			require.NoError(t, err)
			require.Len(t, files, 1)
		})
	}
}

func TestResultFileFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{runScenarioCmd(), runScenarioWithWorkerCmd()} {
		require.NotNil(t, cmd.Flags().Lookup("export-result-file"))
		require.NotNil(t, cmd.Flags().Lookup("export-result-interval"))
		require.NoError(t, cmd.ParseFlags([]string{"--export-result-file", "result.json", "--export-result-interval", "1s"}))
	}
}

func TestRunResultOnPanic(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second}}
		executor := resultSnapshotExecutor{
			Executor: loadgen.ExecutorFunc(func(context.Context, loadgen.ScenarioInfo) error {
				panic("scenario panic")
			}),
			snapshot: func() any { return map[string]int{"completed": 1} },
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if interrupted {
			cancel()
		}

		require.PanicsWithValue(t, "scenario panic", func() {
			var runErr error
			defer runner.finish(ctx, &runErr)
			runErr = runner.runExecutor(ctx, executor, loadgen.ScenarioInfo{})
		})

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.JSONEq(t, `{
  "outcome": "failed",
  "error": "panic: scenario panic",
  "iterations": null,
  "failures": [],
  "scenarioState": {"completed": 1}
}`, string(data))
	}
}

func TestRunResultRunningSnapshots(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{
			scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second},
			counter:           &loadgen.RunCounter{},
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
						Counter:                       runner.counter,
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
