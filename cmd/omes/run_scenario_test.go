package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen"
)

func TestLoadFlags(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scenario string
		flags    []string
		want     string
	}{
		{"legacy", "workflow_with_single_noop_activity", []string{"--iterations=2", "--max-iterations-per-second=2"}, ""},
		{"steady", "workflow_with_single_noop_activity", []string{"--load=steady", "--peak=120/s", "--duration=1m"}, ""},
		{"ramp", "workflow_with_single_noop_activity", []string{"--load=ramp-hold-ramp", "--peak=2", "--duration=1m", "--load-param=up=20%", "--load-param=down=10s"}, ""},
		{"sweep", "throughput_stress", []string{"--load=sweep", "--peak=10", "--duration=1m", "--max-concurrent=3"}, ""},
		{"spikes", "scheduler_stress", []string{"--load=spike-train", "--peak=10", "--duration=1h"}, ""},
		{"needs duration", "workflow_with_single_noop_activity", []string{"--load=steady", "--peak=10"}, "--duration"},
		{"needs peak", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m"}, "--peak"},
		{"iterations", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--iterations=2"}, "use --duration"},
		{"explicit zero iterations", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--iterations=0"}, "use --duration"},
		{"rate", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--max-iterations-per-second=2"}, "use --peak"},
		{"explicit zero rate", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--max-iterations-per-second=0"}, "use --peak"},
		{"unknown pattern", "workflow_with_single_noop_activity", []string{"--load=other", "--duration=1m", "--peak=2"}, "valid patterns: steady, ramp-hold-ramp, sweep, spike-train"},
		{"unknown param", "workflow_with_single_noop_activity", []string{"--load=sweep", "--duration=1m", "--peak=2", "--load-param=unknown=2"}, "valid parameters: from, to, steps"},
		{"malformed param", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--peak=2", "--load-param=wrong"}, "key=value"},
		{"duplicate param", "workflow_with_single_noop_activity", []string{"--load=sweep", "--duration=1m", "--peak=2", "--load-param=steps=2", "--load-param=steps=3"}, "duplicate"},
		{"bad peak", "workflow_with_single_noop_activity", []string{"--load=steady", "--duration=1m", "--peak=NaN"}, "--peak"},
		{"bad fraction", "workflow_with_single_noop_activity", []string{"--load=sweep", "--duration=1m", "--peak=2", "--load-param=from=-1"}, "from"},
		{"peak alone", "workflow_with_single_noop_activity", []string{"--peak=2"}, "requires --load"},
		{"params alone", "workflow_with_single_noop_activity", []string{"--load-param=up=10%"}, "requires --load"},
		{"custom scheduler", "ebb_and_flow", []string{"--load=steady", "--duration=1m", "--peak=2"}, "does not support --load"},
		{"own rate control", "state_transitions_steady", []string{"--load=steady", "--duration=1m", "--peak=2"}, "does not support --load"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRunner(tc.scenario)
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			r.scenarioRunConfig.addCLIFlags(fs)
			require.NoError(t, fs.Parse(tc.flags))

			_, _, err := r.validateInput()
			if tc.want == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.want)
			var usage loadgen.UsageError
			require.ErrorAs(t, err, &usage)
		})
	}
}

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
