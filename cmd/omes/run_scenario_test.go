package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// newRunner builds a runner with the minimum valid input, so each case below
// changes only the one thing it is about.
func newRunner(scenario string) *scenarioRunner {
	return &scenarioRunner{
		scenario: clioptions.ScenarioID{Scenario: scenario, RunID: "test-run"},
	}
}

// Bad input must be reported as a usage error rather than as a failure during a
// run: that is what keeps a stack trace off the screen for a typo.
func TestValidateInputClassifiesBadInputAsUsageErrors(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*scenarioRunner)
		wantMsg string
	}{
		{
			name:    "unknown scenario",
			mutate:  func(r *scenarioRunner) { r.scenario.Scenario = "no_such_scenario" },
			wantMsg: `unknown scenario "no_such_scenario"`,
		},
		{
			name:    "empty run id",
			mutate:  func(r *scenarioRunner) { r.scenario.RunID = "" },
			wantMsg: "--run-id must not be empty",
		},
		{
			name: "iterations with duration",
			mutate: func(r *scenarioRunner) {
				r.iterations = 5
				r.duration = time.Minute
			},
			wantMsg: "cannot be combined",
		},
		{
			name:    "result interval without file",
			mutate:  func(r *scenarioRunner) { r.exportResultInterval = time.Second },
			wantMsg: "--export-result-interval requires --export-result-file",
		},
		{
			name: "negative result interval",
			mutate: func(r *scenarioRunner) {
				r.exportResultFile = "result.json"
				r.exportResultInterval = -time.Second
			},
			wantMsg: "--export-result-interval must not be negative",
		},
		{
			name:    "option without equals",
			mutate:  func(r *scenarioRunner) { r.scenarioOptions = []string{"novalue"} },
			wantMsg: `--option "novalue" is not in key=value format`,
		},
		{
			name:    "unknown option name",
			mutate:  func(r *scenarioRunner) { r.scenarioOptions = []string{"not-an-option=1"} },
			wantMsg: `unknown option "not-an-option"`,
		},
		{
			name: "unreadable option file",
			mutate: func(r *scenarioRunner) {
				missing := filepath.Join(t.TempDir(), "absent.json")
				r.scenarioOptions = []string{"sleep-activity-json=@" + missing}
			},
			wantMsg: "could not be read",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner("throughput_stress")
			tc.mutate(r)

			_, _, err := r.validateInput()
			if err == nil {
				t.Fatal("expected validation to reject this input")
			}
			var usage loadgen.UsageError
			if !errors.As(err, &usage) {
				t.Errorf("want a usage error so it prints without a stack trace, got %T: %v", err, err)
			}
			if !strings.Contains(err.Error(), tc.wantMsg) {
				t.Errorf("error should mention %q, got: %v", tc.wantMsg, err)
			}
		})
	}
}

func TestValidateInputAcceptsGoodInput(t *testing.T) {
	r := newRunner("throughput_stress")
	r.scenarioOptions = []string{"sleep-time=3s"}

	scenario, options, err := r.validateInput()
	if err != nil {
		t.Fatalf("expected valid input to pass, got %v", err)
	}
	if scenario == nil || options == nil {
		t.Fatal("valid input should yield both the scenario and its resolved options")
	}
	info := &loadgen.ScenarioInfo{ScenarioName: "throughput_stress", Options: options}
	if got := info.OptionDuration("sleep-time"); got != 3*time.Second {
		t.Errorf("sleep-time: want 3s, got %v", got)
	}
}

func TestResultFileFlags(t *testing.T) {
	for _, cmd := range []*cobra.Command{
		runScenarioCmd(),
		runScenarioWithWorkerCmd(),
	} {
		require.NotNil(t, cmd.Flags().Lookup("export-result-file"))
		require.NotNil(t, cmd.Flags().Lookup("export-result-interval"))
		require.NoError(t, cmd.ParseFlags([]string{"--export-result-file", "result.json", "--export-result-interval", "1s"}))
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
			err := tc.run(t.Context(), path)
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
	for _, tc := range []struct {
		name   string
		runErr error
	}{
		{name: "successful run"},
		{name: "failed run", runErr: errors.New("run failed")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing", "result.json")
			runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}

			err := tc.runErr
			runner.finish(t.Context(), &err)

			require.ErrorContains(t, err, "writing result file")
			if tc.runErr != nil {
				require.ErrorIs(t, err, tc.runErr)
			}
		})
	}
}

func TestRunResultOnPanic(t *testing.T) {
	for _, tc := range []struct {
		name        string
		interrupted bool
	}{
		{name: "panic"},
		{name: "panic after interruption", interrupted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: time.Second}}

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.interrupted {
				cancel()
			}

			require.PanicsWithValue(t, "scenario panic", func() {
				var runErr error
				defer runner.finish(ctx, &runErr)
				executor := &scenarioStateExecutor{}
				executor.ExecutorFunc = func(context.Context, loadgen.ScenarioInfo) error {
					executor.state = 1
					panic("scenario panic")
				}
				_ = runner.runWithSnapshots(ctx, executor, loadgen.ScenarioInfo{})
			})

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, `{
  "outcome": "failed",
  "error": "panic: scenario panic",
  "iterations": null,
  "failures": [],
  "scenarioState": 1
}`, string(data))
		})
	}
}

type scenarioStateExecutor struct {
	loadgen.ExecutorFunc
	state int
}

func (e *scenarioStateExecutor) Snapshot() any { return e.state }

func TestFinishScenarioState(t *testing.T) {
	for _, tc := range []struct {
		name       string
		snapshot   bool
		noExecutor bool
		wantState  string
	}{
		{name: "snapshot", snapshot: true, wantState: "1"},
		{name: "no snapshot", wantState: "null"},
		{name: "before executor exists", noExecutor: true, wantState: "null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			runner := &scenarioRunner{scenarioRunConfig: scenarioRunConfig{exportResultFile: path}}
			if !tc.noExecutor {
				var executor scenarioStateExecutor
				executor.ExecutorFunc = func(context.Context, loadgen.ScenarioInfo) error {
					executor.state = 1
					return nil
				}
				var run loadgen.Executor = executor.ExecutorFunc
				if tc.snapshot {
					run = &executor
				}
				runner.state = scenarioState(run)
				require.NoError(t, run.Run(t.Context(), loadgen.ScenarioInfo{}))
			}
			var runErr error
			runner.finish(t.Context(), &runErr)
			require.NoError(t, runErr)

			data, err := os.ReadFile(path)
			require.NoError(t, err)
			require.JSONEq(t, `{
  "outcome": "passed",
  "iterations": null,
  "failures": [],
  "scenarioState": `+tc.wantState+`
}`, string(data))
		})
	}
}

func TestRunningSnapshots(t *testing.T) {
	run := func(t *testing.T, path string, interval time.Duration, executor *steppedExecutor) <-chan error {
		runner := &scenarioRunner{
			scenarioRunConfig: scenarioRunConfig{exportResultFile: path, exportResultInterval: interval},
			counter:           &loadgen.RunCounter{},
		}
		done := make(chan error, 1)
		go func() {
			var err error
			defer func() { done <- err }()
			defer runner.finish(t.Context(), &err)
			info := loadgen.ScenarioInfo{Configuration: loadgen.RunConfiguration{Counter: runner.counter}}
			err = runner.runWithSnapshots(t.Context(), executor, info)
		}()
		return done
	}

	t.Run("rewrites the file on every tick with current counts and state", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			executor := &steppedExecutor{step: make(chan error)}
			done := run(t, path, time.Second, executor)

			failures := []loadgen.IterationFailure{}
			for i, counts := range []loadgen.IterationCounts{
				{Started: 1},
				{Started: 2, Succeeded: 1},
				{Started: 3, Succeeded: 1, Failed: 1},
			} {
				time.Sleep(time.Second)
				synctest.Wait()
				require.Equal(t, runResult{
					Outcome: "running", Iterations: &counts, Failures: failures,
					ScenarioState: map[string]any{"completed": float64(i)},
				}, readResult(t, path))

				var iterationErr error
				if i == 1 {
					iterationErr = errors.New("iteration failed")
					failures = append(failures, loadgen.IterationFailure{Iteration: 2, Attempt: 1, Error: "iteration 2 failed: iteration failed"})
				}
				executor.step <- iterationErr
			}

			err := <-done
			require.Error(t, err)
			final := readResult(t, path)
			require.Equal(t, runResult{
				Outcome: "failed", Error: err.Error(),
				Iterations:    &loadgen.IterationCounts{Started: 3, Succeeded: 2, Failed: 1},
				Failures:      failures,
				ScenarioState: map[string]any{"completed": float64(3)},
			}, final)

			time.Sleep(10 * time.Second)
			synctest.Wait()
			require.Equal(t, final, readResult(t, path))
		})
	})

	t.Run("final write waits for an in-flight snapshot", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			release := make(chan struct{})
			first := true
			executor := &steppedExecutor{
				step: make(chan error),
				snapshot: func() any {
					if first {
						first = false
						<-release
					}
					return "final"
				},
			}
			done := run(t, path, time.Second, executor)

			time.Sleep(time.Second)
			synctest.Wait()
			close(executor.step)
			synctest.Wait()

			close(release)
			require.NoError(t, <-done)
			synctest.Wait()
			require.Equal(t, runResult{
				Outcome:       "passed",
				Iterations:    &loadgen.IterationCounts{Started: 3, Succeeded: 3},
				Failures:      []loadgen.IterationFailure{},
				ScenarioState: "final",
			}, readResult(t, path))
		})
	})

	t.Run("no snapshots without an interval", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "result.json")
			executor := &steppedExecutor{step: make(chan error)}
			done := run(t, path, 0, executor)

			time.Sleep(time.Hour)
			synctest.Wait()
			require.NoFileExists(t, path)

			close(executor.step)
			require.NoError(t, <-done)
			require.FileExists(t, path)
		})
	})

	t.Run("snapshot write error keeps the run error", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "missing", "result.json")
			executor := &steppedExecutor{step: make(chan error)}
			done := run(t, path, time.Second, executor)

			time.Sleep(time.Second)
			synctest.Wait()
			require.NoError(t, os.Mkdir(filepath.Dir(path), 0700))

			runErr := errors.New("run failed")
			executor.step <- runErr
			close(executor.step)
			err := <-done
			require.ErrorIs(t, err, runErr)
			require.ErrorContains(t, err, "writing result file")
			require.FileExists(t, path)
		})
	})
}

type steppedExecutor struct {
	step      chan error
	completed int
	snapshot  func() any
}

func (e *steppedExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	info.Logger = zap.NewNop().Sugar()
	info.MetricsHandler = client.MetricsNopHandler
	info.Configuration = loadgen.RunConfiguration{
		Iterations: 3, MaxConcurrent: 1,
		ContinueOnIterationFailure:    true,
		DoNotRegisterSearchAttributes: true,
		Counter:                       info.Configuration.Counter,
	}
	executor := &loadgen.GenericExecutor{Execute: func(context.Context, *loadgen.Run) error {
		err := <-e.step
		e.completed++
		return err
	}}
	return executor.Run(ctx, info)
}

func (e *steppedExecutor) Snapshot() any {
	if e.snapshot != nil {
		return e.snapshot()
	}
	return map[string]int{"completed": e.completed}
}

func readResult(t *testing.T, path string) runResult {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var result runResult
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}
