package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
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
		require.NoError(t, cmd.ParseFlags([]string{"--export-result-file", "result.json"}))
	}
}

func TestRunResultOnPanic(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if interrupted {
			cancel()
		}

		require.PanicsWithValue(t, "scenario panic", func() {
			var runErr error
			defer runner.finish(ctx, &runErr)
			panic("scenario panic")
		})

		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.JSONEq(t, `{
  "outcome": "failed",
  "error": "panic: scenario panic",
  "iterations": null,
  "failures": [],
  "scenarioState": null
}`, string(data))
	}
}
