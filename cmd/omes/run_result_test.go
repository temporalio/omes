package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
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
			name:   "unfinished takes precedence",
			end:    runEnd{err: errors.New("run failed"), ctxErr: context.Canceled, panicked: true},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}, TimedOut: true},
			want:   "running",
		},
		{
			name:   "panic takes precedence over interruption and timeout",
			end:    runEnd{finished: true, err: errors.New("panic"), ctxErr: context.Canceled, panicked: true},
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}, TimedOut: true},
			want:   "failed",
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
			counts: loadgen.RunCounts{Iterations: &loadgen.IterationCounts{Started: 1, Failed: 1}, TimedOut: true},
			want:   "interrupted",
		},
		{
			name: "context deadline",
			end:  runEnd{finished: true, ctxErr: context.DeadlineExceeded, err: errors.New("run failed")},
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

func TestWriteResultFileRenameFailureCleansUp(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "result.json")
	require.NoError(t, os.Mkdir(path, 0700))

	err := writeResultFile(path, []byte("result"))
	var pathErr *os.LinkError
	require.ErrorAs(t, err, &pathErr)
	require.Equal(t, "rename", pathErr.Op)

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
			result := newRunResult(runEnd{finished: true, err: tc.runErr}, loadgen.RunCounts{}, tc.scenarioState)
			require.NoError(t, result.write(path))
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(data))

			files, err := os.ReadDir(filepath.Dir(path))
			require.NoError(t, err)
			require.Len(t, files, 1)
		})
	}
}

func TestWriteResultFile(t *testing.T) {
	for _, tc := range []struct {
		name     string
		existing bool
	}{
		{name: "create"},
		{name: "replace", existing: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			path := filepath.Join(directory, "result.json")
			var previous os.FileInfo
			if tc.existing {
				require.NoError(t, os.WriteFile(path, []byte("old result"), 0600))
				var err error
				previous, err = os.Stat(path)
				require.NoError(t, err)
			}

			require.NoError(t, writeResultFile(path, []byte("new result")))
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, "new result", string(data))
			if tc.existing {
				current, err := os.Stat(path)
				require.NoError(t, err)
				require.False(t, os.SameFile(previous, current))
			}
			files, err := os.ReadDir(directory)
			require.NoError(t, err)
			require.Len(t, files, 1)
		})
	}
}
