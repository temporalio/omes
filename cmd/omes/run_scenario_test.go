package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
			name:    "invalid iteration failure policy",
			mutate:  func(r *scenarioRunner) { r.iterationFailurePolicy = "sometimes" },
			wantMsg: "--iteration-failure-policy must be",
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

func TestIterationFailurePolicyConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		policy string
		want   bool
	}{
		{name: "zero value defaults to continue", want: true},
		{name: "continue", policy: iterationFailurePolicyContinue, want: true},
		{name: "fail fast", policy: iterationFailurePolicyFailFast, want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := scenarioRunConfig{iterationFailurePolicy: test.policy}.loadgenConfiguration()
			if config.ContinueOnIterationFailure != test.want {
				t.Fatalf("ContinueOnIterationFailure = %v, want %v", config.ContinueOnIterationFailure, test.want)
			}
		})
	}
}

func TestIterationFailuresOnly(t *testing.T) {
	degraded := &loadgen.IterationFailuresError{Attempted: 2, Succeeded: 1, Failed: 1}

	found, ok := iterationFailuresOnly(fmt.Errorf("scenario wrapper: %w", degraded))
	if !ok || found != degraded {
		t.Fatalf("expected wrapped degraded completion to be recognized, got %v, %v", found, ok)
	}

	if _, ok := iterationFailuresOnly(errors.Join(degraded, errors.New("cleanup failed"))); ok {
		t.Fatal("must not treat a joined run-level error as only iteration failures")
	}
	if _, ok := iterationFailuresOnly(errors.New("run failed")); ok {
		t.Fatal("must not treat an ordinary run error as degraded completion")
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
