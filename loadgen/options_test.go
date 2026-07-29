package loadgen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResolveOptionsUndeclaredPassesThrough(t *testing.T) {
	// Scenarios that declare nothing keep the legacy behavior: anything goes.
	// This is what keeps scenarios outside this repo working.
	s := &Scenario{}
	got, err := s.ResolveOptions(map[string]string{"anything": "goes", "even": "nonsense"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 2 || got["anything"] != "goes" {
		t.Fatalf("expected provided options untouched, got %v", got)
	}
}

func TestResolveOptionsKeepsProvidedValues(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 30, "")
		o.Duration("wait", time.Second, "")
	}}
	got, err := s.ResolveOptions(map[string]string{"count": "7", "wait": "2m"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got["count"] != "7" {
		t.Errorf("count: want 7, got %q", got["count"])
	}
	if got["wait"] != "2m0s" {
		t.Errorf("wait: want 2m0s, got %q", got["wait"])
	}
}

func TestResolveOptionsDoesNotInjectDefaults(t *testing.T) {
	// Declared defaults describe and validate; they must not be written into the
	// map. Some scenarios treat an option's absence as meaningful (a deprecated
	// alias feeding another option's default, or a presence-only flag), and
	// injecting defaults would silently change what those scenarios do.
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 30, "")
		o.Bool("enabled", false, "")
		o.String("label", "", "")
	}}
	got, err := s.ResolveOptions(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no values for unsupplied options, got %v", got)
	}
}

func TestResolveOptionsRejectsUnknownName(t *testing.T) {
	// The typo case that previously ran happily with a silently-ignored option.
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("activities-per-workflow", 30, "")
		o.Int("children-per-workflow", 30, "")
	}}
	_, err := s.ResolveOptions(map[string]string{"activites-per-workflow": "50"})
	if err == nil {
		t.Fatal("expected an error for a misspelled option")
	}
	if !strings.Contains(err.Error(), "activites-per-workflow") {
		t.Errorf("error should name the offending key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "activities-per-workflow") {
		t.Errorf("error should list accepted options, got: %v", err)
	}
}

func TestResolveOptionsRejectsMalformedValues(t *testing.T) {
	for _, tc := range []struct {
		typ     string
		declare func(*OptionSet)
		value   string
	}{
		{"int", func(o *OptionSet) { o.Int("n", 0, "") }, "abc"},
		{"float64", func(o *OptionSet) { o.Float64("n", 0, "") }, "abc"},
		{"bool", func(o *OptionSet) { o.Bool("n", false, "") }, "maybe"},
		{"duration", func(o *OptionSet) { o.Duration("n", 0, "") }, "5 fortnights"},
	} {
		t.Run(tc.typ, func(t *testing.T) {
			s := &Scenario{Options: tc.declare}
			_, err := s.ResolveOptions(map[string]string{"n": tc.value})
			if err == nil {
				t.Fatalf("expected an error for %s value %q", tc.typ, tc.value)
			}
			if !strings.Contains(err.Error(), tc.value) {
				t.Errorf("error should quote the bad value, got: %v", err)
			}
			if !strings.Contains(err.Error(), tc.typ) {
				t.Errorf("error should name the expected type, got: %v", err)
			}
		})
	}
}

func TestResolveOptionsAcceptsBoolForms(t *testing.T) {
	// pflag accepts all the strconv.ParseBool forms, not just the literal "true".
	s := &Scenario{Options: func(o *OptionSet) { o.Bool("on", false, "") }}
	for _, v := range []string{"true", "false", "1", "0", "TRUE", "False"} {
		if _, err := s.ResolveOptions(map[string]string{"on": v}); err != nil {
			t.Errorf("expected %q to be accepted, got %v", v, err)
		}
	}
}

func TestResolveOptionsRequired(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("task-queue-count", 0, "")
		o.MarkRequired("task-queue-count")
	}}

	_, err := s.ResolveOptions(nil)
	if err == nil {
		t.Fatal("expected an error when a required option is missing")
	}
	if !strings.Contains(err.Error(), "task-queue-count") || !strings.Contains(err.Error(), "required") {
		t.Errorf("error should say which option is required, got: %v", err)
	}

	if _, err := s.ResolveOptions(map[string]string{"task-queue-count": "4"}); err != nil {
		t.Errorf("expected supplying the required option to succeed, got %v", err)
	}
}

func TestResolveOptionsReportsEveryProblemAtOnce(t *testing.T) {
	// The point of validating up front: one run tells you everything that's wrong.
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 0, "")
		o.Int("needed", 0, "")
		o.MarkRequired("needed")
	}}
	_, err := s.ResolveOptions(map[string]string{"count": "abc", "bogus": "1"})
	if err == nil {
		t.Fatal("expected errors")
	}
	msg := err.Error()
	for _, want := range []string{"count", "bogus", "needed"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected all problems reported together, %q missing from: %v", want, msg)
		}
	}
}

func TestDeclaredOptionsDescribesForDisplay(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Duration("wait", 5*time.Second, "How long to wait.")
		o.Int("needed", 0, "Must be supplied.")
		o.MarkRequired("needed")
	}}
	opts := s.DeclaredOptions()
	if len(opts) != 2 {
		t.Fatalf("want 2 declared options, got %d", len(opts))
	}
	// Sorted by name: needed, wait.
	if opts[0].Name != "needed" || !opts[0].Required {
		t.Errorf("want required option first, got %+v", opts[0])
	}
	if opts[1].Name != "wait" || opts[1].Type != "duration" || opts[1].Default != "5s" {
		t.Errorf("want wait/duration/5s, got %+v", opts[1])
	}
	if opts[1].Usage != "How long to wait." {
		t.Errorf("usage not carried through, got %q", opts[1].Usage)
	}
}

func TestDeclaredOptionsEmptyWhenUndeclared(t *testing.T) {
	if got := (&Scenario{}).DeclaredOptions(); len(got) != 0 {
		t.Errorf("expected no declared options, got %v", got)
	}
}

func TestScenarioOptionAccessorsDoNotPanic(t *testing.T) {
	// Undeclared options are still parsed lazily; a bad value must fall back
	// rather than take the process down.
	info := &ScenarioInfo{ScenarioOptions: map[string]string{
		"i": "abc", "f": "abc", "b": "maybe", "d": "abc",
	}}
	if got := info.ScenarioOptionInt("i", 7); got != 7 {
		t.Errorf("int: want fallback 7, got %v", got)
	}
	if got := info.ScenarioOptionFloat("f", 1.5); got != 1.5 {
		t.Errorf("float: want fallback 1.5, got %v", got)
	}
	if got := info.ScenarioOptionBool("b", true); !got {
		t.Errorf("bool: want fallback true, got %v", got)
	}
	if got := info.ScenarioOptionDuration("d", time.Second); got != time.Second {
		t.Errorf("duration: want fallback 1s, got %v", got)
	}
}

func TestScenarioOptionBoolAcceptsParseBoolForms(t *testing.T) {
	info := &ScenarioInfo{ScenarioOptions: map[string]string{"on": "1", "off": "0"}}
	if !info.ScenarioOptionBool("on", false) {
		t.Error(`expected "1" to read as true`)
	}
	if info.ScenarioOptionBool("off", true) {
		t.Error(`expected "0" to read as false`)
	}
}

func TestGetDefaultConfigurationPrefersScenarioDeclaration(t *testing.T) {
	want := RunConfiguration{Iterations: 42}
	s := &Scenario{DefaultConfiguration: &want, ExecutorFn: func() Executor { return nil }}
	got, ok := s.GetDefaultConfiguration()
	if !ok || got.Iterations != 42 {
		t.Fatalf("want declared configuration, got %+v (ok=%v)", got, ok)
	}

	// No declaration, and an executor that doesn't implement the interface.
	s = &Scenario{ExecutorFn: func() Executor { return ExecutorFunc(nil) }}
	if _, ok := s.GetDefaultConfiguration(); ok {
		t.Error("expected no default configuration to be reported")
	}
}

func TestInvalidOptionsErrorNamesScenarioAndUnwraps(t *testing.T) {
	inner := errors.New("boom")
	err := &InvalidOptionsError{ScenarioName: "my_scenario", Err: inner}
	if !strings.Contains(err.Error(), "my_scenario") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("want scenario name and cause in message, got %q", err.Error())
	}
	if !errors.Is(err, inner) {
		t.Error("want the wrapped error to be unwrappable")
	}
}
