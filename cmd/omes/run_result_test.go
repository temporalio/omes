package main

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"path/filepath"
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
	rate := 4.5
	result := runResult{
		Version: 1, Scenario: "example", RunID: "run", ExecutionID: "execution", RunFamily: "family",
		Configuration: resultConfiguration{Iterations: 2, MaxConcurrent: 3, MaxIterationsPerSecond: &rate, MaxIterationAttempts: 6, Timeout: "7s", Options: map[string]string{"sleep": "8s"}},
		StartedAt:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), EndedAt: time.Date(2026, 1, 2, 3, 4, 14, 0, time.UTC), Duration: "9s",
		Outcome: "failed", Error: "iteration failed",
		Iterations:        &loadgen.IterationCounts{Started: 2, Succeeded: 1, Failed: 1},
		IterationFailures: []loadgen.IterationFailure{{Iteration: 2, Attempt: 6, Error: "failure"}},
		ScenarioState:     json.RawMessage(`{"completed":1}`),
	}
	encoded, err := json.Marshal(result)
	require.NoError(t, err)
	require.JSONEq(t, `{
  "version": 1,
  "scenario": "example",
  "runId": "run",
  "executionId": "execution",
  "runFamily": "family",
  "configuration": {
    "iterations": 2,
    "maxConcurrent": 3,
    "maxIterationsPerSecond": 4.5,
    "maxIterationAttempts": 6,
    "timeout": "7s",
    "options": {
      "sleep": "8s"
    }
  },
  "startedAt": "2026-01-02T03:04:05Z",
  "endedAt": "2026-01-02T03:04:14Z",
  "duration": "9s",
  "outcome": "failed",
  "error": "iteration failed",
  "iterations": {
    "started": 2,
    "succeeded": 1,
    "failed": 1
  },
  "iterationFailures": [
    {
      "iteration": 2,
      "attempt": 6,
      "error": "failure"
    }
  ],
  "scenarioState": {
    "completed": 1
  }
}`, string(encoded))
	var decoded runResult
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, result, decoded)
	encoded, err = json.Marshal(runResult{Version: 1})
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(encoded, &fields))
	for _, field := range []string{"runFamily", "iterations", "iterationFailures", "scenarioState", "error"} {
		require.NotContains(t, fields, field)
	}
}

func TestRunResultOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name           string
		fail           bool
		iterationError error
		timeout        time.Duration
		cancel         bool
		want           string
	}{
		{name: "pass", want: "passed"},
		{name: "iteration failure", fail: true, want: "failed"},
		{name: "iteration deadline", iterationError: context.DeadlineExceeded, want: "failed"},
		{name: "iteration cancellation", iterationError: context.Canceled, want: "failed"},
		{name: "timeout", timeout: time.Second, want: "timedOut"},
		{name: "interruption", cancel: true, want: "interrupted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				path := filepath.Join(t.TempDir(), "result.json")
				err := withRunResult(ctx, clioptions.ScenarioID{Scenario: "test", RunID: "run"}, scenarioRunConfig{resultFile: path, iterations: 1, timeout: tc.timeout}, func(result *runResult) error {
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
					result.executor = executor
					return executor.Run(ctx, loadgen.ScenarioInfo{Logger: zap.NewNop().Sugar(), MetricsHandler: client.MetricsNopHandler, Configuration: loadgen.RunConfiguration{Iterations: 1, Timeout: tc.timeout, Statistics: result.statistics}})
				})
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
				require.NotEmpty(t, result.ExecutionID)
				require.NotNil(t, result.Iterations)
			})
		})
	}
}

func TestRunResultMissingOnInvalidInput(t *testing.T) {
	for _, worker := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "result.json")
		runner := newRunner("no_such_scenario")
		runner.resultFile = path
		var err error
		if worker {
			r := workerWithScenarioRunner{scenarioRunConfig: runner.scenarioRunConfig}
			r.ScenarioID = runner.scenario
			err = r.run(context.Background())
		} else {
			err = runner.run(context.Background())
		}
		require.Error(t, err)
		_, readErr := os.Stat(path)
		require.ErrorIs(t, readErr, os.ErrNotExist)
	}
}

func TestRunResultWriteErrorPreservesRunError(t *testing.T) {
	for _, runErr := range []error{nil, errors.New("run failed")} {
		path := filepath.Join(t.TempDir(), "missing", "result.json")
		err := withRunResult(context.Background(), clioptions.ScenarioID{}, scenarioRunConfig{resultFile: path}, func(result *runResult) error {
			result.executor = &loadgen.GenericExecutor{}
			return runErr
		})
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
	err := withRunResult(context.Background(), clioptions.ScenarioID{}, scenarioRunConfig{resultFile: path}, func(result *runResult) error {
		result.executor = &loadgen.GenericExecutor{}
		return nil
	})
	require.ErrorContains(t, err, "writing result file")
	files, err := os.ReadDir(directory)
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.True(t, files[0].IsDir())
}

type resultSnapshotExecutor struct{}

func (resultSnapshotExecutor) Run(context.Context, loadgen.ScenarioInfo) error { return nil }
func (resultSnapshotExecutor) Snapshot() any                                   { return map[string]int{"completed": 12} }

func TestRunResultSnapshotAndOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	require.NoError(t, os.WriteFile(path, []byte("old result"), 0600))
	err := withRunResult(context.Background(), clioptions.ScenarioID{Scenario: "example"}, scenarioRunConfig{resultFile: path, duration: 3 * time.Second}, func(result *runResult) error {
		result.executor = resultSnapshotExecutor{}
		result.options = (&loadgen.Scenario{Options: func(o *loadgen.OptionSet) { o.Duration("sleep", 2*time.Second, "Sleep duration") }}).MustResolveOptions(nil)
		return nil
	})
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result runResult
	require.NoError(t, json.Unmarshal(data, &result))
	require.JSONEq(t, `{"completed":12}`, string(result.ScenarioState))
	require.Equal(t, map[string]string{"sleep": "2s"}, result.Configuration.Options)
	require.Equal(t, "3s", result.Configuration.Duration)
	require.Nil(t, result.Iterations)
	files, err := os.ReadDir(filepath.Dir(path))
	require.NoError(t, err)
	require.Len(t, files, 1)
}

func TestResultFileFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{runScenarioCmd(), runScenarioWithWorkerCmd()} {
		require.NotNil(t, cmd.Flags().Lookup("result-file"))
		require.NoError(t, cmd.ParseFlags([]string{"--result-file", "result.json"}))
	}
}

func TestRunResultMissingOnConnectionFailure(t *testing.T) {
	runner := newRunner("workflow_with_single_noop_activity")
	runner.logger = zap.NewNop().Sugar()
	runner.resultFile = filepath.Join(t.TempDir(), "result.json")
	runner.clientOptions.EnableTLS = true
	runner.clientOptions.ClientCertPath = filepath.Join(t.TempDir(), "missing-cert")
	runner.clientOptions.ClientKeyPath = filepath.Join(t.TempDir(), "missing-key")
	err := runner.run(context.Background())
	require.ErrorContains(t, err, "failed dialing")
	_, statErr := os.Stat(runner.resultFile)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}

func TestRunResultInvalidRate(t *testing.T) {
	for _, rate := range []float64{math.NaN(), math.Inf(1)} {
		path := filepath.Join(t.TempDir(), "result.json")
		err := withRunResult(context.Background(), clioptions.ScenarioID{}, scenarioRunConfig{resultFile: path, maxIterationsPerSecond: rate}, func(result *runResult) error {
			executor := &loadgen.GenericExecutor{}
			result.executor = executor
			return executor.Run(context.Background(), loadgen.ScenarioInfo{Configuration: loadgen.RunConfiguration{MaxIterationsPerSecond: rate, Statistics: result.statistics}})
		})
		require.ErrorContains(t, err, "invalid scenario")
		data, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		var result runResult
		require.NoError(t, json.Unmarshal(data, &result))
		require.Equal(t, "failed", result.Outcome)
		require.Contains(t, result.Error, "MaxIterationsPerSecond")
	}
}

func TestRunResultMissingOnWorkerStartupFailure(t *testing.T) {
	runner := workerWithScenarioRunner{scenarioRunConfig: scenarioRunConfig{resultFile: filepath.Join(t.TempDir(), "result.json")}}
	runner.ScenarioID = clioptions.ScenarioID{Scenario: "workflow_with_single_noop_activity", RunID: "test-run"}
	runner.Logger = zap.NewNop().Sugar()
	runner.TaskQueueIndexSuffixStart = 2
	runner.TaskQueueIndexSuffixEnd = 1
	err := runner.run(context.Background())
	require.ErrorContains(t, err, "worker did not start")
	_, statErr := os.Stat(runner.resultFile)
	require.ErrorIs(t, statErr, os.ErrNotExist)
}
