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
	"github.com/temporalio/omes/loadgen/load"
)

func TestLoadValues(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  load.Pattern
	}{
		{"steady,peak=120/s", load.Pattern{Name: "steady", Peak: 120, Duration: time.Minute, Phases: []load.Phase{{Name: "hold", Duration: time.Minute, From: 120, To: 120}}}},
		{"ramp-hold-ramp,peak=2,up=0.2,down=10s", load.Pattern{Name: "ramp-hold-ramp", Peak: 2, Duration: time.Minute, Phases: []load.Phase{
			{Name: "up", Duration: 12 * time.Second, From: 0, To: 2}, {Name: "hold", Duration: 38 * time.Second, From: 2, To: 2}, {Name: "down", Duration: 10 * time.Second, From: 2, To: 0},
		}}},
		{"sweep,peak=10,from=0,to=1,steps=2", load.Pattern{Name: "sweep", Peak: 10, Duration: time.Minute, Phases: []load.Phase{
			{Name: "step-1", Duration: 30 * time.Second, From: 0, To: 0}, {Name: "step-2", Duration: 30 * time.Second, From: 10, To: 10},
		}}},
		{"spike-train,peak=10,base=0.5,every=0.5,for=0.1", load.Pattern{Name: "spike-train", Peak: 10, Duration: time.Minute, Phases: []load.Phase{
			{Name: "base", Duration: 30 * time.Second, From: 5, To: 5}, {Name: "spike", Duration: 6 * time.Second, From: 10, To: 10}, {Name: "base", Duration: 24 * time.Second, From: 5, To: 5},
		}}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			r := newRunner("workflow_with_single_noop_activity")
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			r.scenarioRunConfig.addCLIFlags(fs)
			require.NoError(t, fs.Parse([]string{"--load=" + tc.value, "--duration=1m"}))

			_, _, err := r.validateInput()
			require.NoError(t, err)
			require.Equal(t, &tc.want, r.loadPattern)
		})
	}
}

func TestLoadFlags(t *testing.T) {
	for _, tc := range []struct {
		name     string
		scenario string
		flags    []string
		want     string
	}{
		{"legacy", "workflow_with_single_noop_activity", []string{"--iterations=2", "--max-iterations-per-second=2"}, ""},
		{"sweep defaults", "throughput_stress", []string{"--load=sweep,peak=10", "--duration=1m", "--max-concurrent=3"}, ""},
		{"spike defaults", "scheduler_stress", []string{"--load=spike-train,peak=10", "--duration=1h"}, ""},

		{"needs duration", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10"}, "--duration"},
		{"zero duration", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=0"}, "--duration"},
		{"iterations", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=1m", "--iterations=2"}, "use --duration"},
		{"explicit zero iterations", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=1m", "--iterations=0"}, "use --duration"},
		{"rate", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=1m", "--max-iterations-per-second=2"}, "set peak= in --load"},
		{"explicit zero rate", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=1m", "--max-iterations-per-second=0"}, "set peak= in --load"},
		{"negative concurrency", "workflow_with_single_noop_activity", []string{"--load=steady,peak=10", "--duration=1m", "--max-concurrent=-1"}, "--max-concurrent"},

		{"custom scheduler", "ebb_and_flow", []string{"--load=steady,peak=2", "--duration=1m"}, "does not support --load"},
		{"own rate control", "state_transitions_steady", []string{"--load=steady,peak=2", "--duration=1m"}, "does not support --load"},
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

func TestLoadValuesRejectInvalidInput(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  []string
	}{
		{"", []string{"valid patterns: steady, ramp-hold-ramp, sweep, spike-train"}},
		{"other,peak=2", []string{"valid patterns: steady, ramp-hold-ramp, sweep, spike-train"}},
		{"steady", []string{"peak", "valid parameters: peak"}},
		{"ramp-hold-ramp,up=0.2", []string{"peak", "valid parameters: peak, up, down"}},
		{"sweep,steps=2", []string{"peak", "valid parameters: peak, from, to, steps"}},
		{"spike-train,every=10s", []string{"peak", "valid parameters: peak, base, every, for"}},

		{"steady,peak=0", []string{"peak"}},
		{"steady,peak=-1", []string{"peak"}},
		{"steady,peak=NaN", []string{"peak", "valid parameters: peak"}},
		{"steady,peak=Inf", []string{"peak"}},
		{"steady,peak=1000000001", []string{"peak"}},
		{"steady,peak=10%", []string{"fraction"}},
		{"steady,peak=1/m", []string{"peak"}},

		{"steady,peak=2,up=1s", []string{"valid parameters: peak"}},
		{"ramp-hold-ramp,peak=2,unknown=1", []string{"valid parameters: peak, up, down"}},
		{"sweep,peak=2,unknown=2", []string{"valid parameters: peak, from, to, steps"}},
		{"spike-train,peak=2,unknown=1", []string{"valid parameters: peak, base, every, for"}},

		{"steady,peak=2,peak=3", []string{"duplicate", "valid parameters: peak"}},
		{"sweep,peak=2,steps=2,steps=3", []string{"duplicate", "valid parameters: peak, from, to, steps"}},

		{"steady,peak=2,wrong", []string{"key=value", "valid parameters: peak"}},
		{"steady,peak=2,=2", []string{"key=value", "valid parameters: peak"}},
		{"steady,peak=2=3", []string{"key=value", "valid parameters: peak"}},
		{"steady,peak=", []string{"key=value", "valid parameters: peak"}},
		{"steady,peak=2,", []string{"key=value", "valid parameters: peak"}},
		{"steady,,peak=2", []string{"key=value", "valid parameters: peak"}},
		{"steady,peak=2, ", []string{"key=value", "valid parameters: peak"}},

		{"sweep,peak=2,from=-1", []string{"from"}},
		{"sweep,peak=2,to=1.1", []string{"to"}},
		{"sweep,peak=2,from=25%", []string{"fraction"}},
		{"sweep,peak=2,to=100%", []string{"fraction"}},
		{"spike-train,peak=2,base=40%", []string{"fraction"}},

		{"ramp-hold-ramp,peak=2,up=1.01", []string{"up must be a fraction from 0 to 1", "valid parameters: peak, up, down"}},
		{"ramp-hold-ramp,peak=2,down=-0.1", []string{"down must be a fraction from 0 to 1"}},
		{"spike-train,peak=2,every=NaN", []string{"every must be a fraction from 0 to 1"}},

		{"ramp-hold-ramp,peak=2,up=10%", []string{"fraction"}},
		{"ramp-hold-ramp,peak=2,down=10%", []string{"fraction"}},
		{"spike-train,peak=2,every=10%", []string{"fraction"}},
		{"spike-train,peak=2,for=10%", []string{"fraction"}},
	} {
		t.Run(tc.value, func(t *testing.T) {
			r := newRunner("workflow_with_single_noop_activity")
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			r.scenarioRunConfig.addCLIFlags(fs)
			require.NoError(t, fs.Parse([]string{"--load=" + tc.value, "--duration=1m"}))

			_, _, err := r.validateInput()
			for _, want := range tc.want {
				require.ErrorContains(t, err, want)
			}
			var usage loadgen.UsageError
			require.ErrorAs(t, err, &usage)
		})
	}
}

func TestLoadFlagsRejectRemovedFlags(t *testing.T) {
	for _, flag := range []string{"--peak=2", "--load-param=up=0.1"} {
		t.Run(flag, func(t *testing.T) {
			fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
			newRunner("workflow_with_single_noop_activity").scenarioRunConfig.addCLIFlags(fs)

			require.ErrorContains(t, fs.Parse([]string{flag}), "unknown flag")
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
