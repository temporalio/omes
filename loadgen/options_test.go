package loadgen

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResolveOptionsRejectsWhenNoneDeclared(t *testing.T) {
	s := &Scenario{}
	_, err := s.ResolveOptions(map[string]string{"anything": "goes"})
	if err == nil {
		t.Fatal("expected an error: a scenario declaring no options accepts none")
	}
	if !strings.Contains(err.Error(), "accepts no options") {
		t.Errorf("error should say the scenario accepts no options, got: %v", err)
	}
}

func TestResolveOptionsKeepsProvidedValues(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 30, "")
		o.Duration("wait", time.Second, "")
	}}
	set, err := s.ResolveOptions(map[string]string{"count": "7", "wait": "2m"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	info := &ScenarioInfo{Options: set}
	if got := info.OptionInt("count"); got != 7 {
		t.Errorf("count: want 7, got %v", got)
	}
	if got := info.OptionDuration("wait"); got != 2*time.Minute {
		t.Errorf("wait: want 2m, got %v", got)
	}
}

func TestOptionAccessorsReturnDeclaredDefaults(t *testing.T) {
	// The declaration is the only place a default lives, so an unsupplied option
	// reads back as whatever was declared.
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 30, "")
		o.Duration("wait", 5*time.Second, "")
		o.Bool("enabled", true, "")
		o.Float64("ratio", 1.5, "")
		o.String("label", "hello", "")
	}}
	set, err := s.ResolveOptions(nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	info := &ScenarioInfo{Options: set}
	if got := info.OptionInt("count"); got != 30 {
		t.Errorf("count: want 30, got %v", got)
	}
	if got := info.OptionDuration("wait"); got != 5*time.Second {
		t.Errorf("wait: want 5s, got %v", got)
	}
	if got := info.OptionBool("enabled"); !got {
		t.Errorf("enabled: want true, got %v", got)
	}
	if got := info.OptionFloat64("ratio"); got != 1.5 {
		t.Errorf("ratio: want 1.5, got %v", got)
	}
	if got := info.OptionString("label"); got != "hello" {
		t.Errorf("label: want hello, got %q", got)
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

func TestOptionAccessorsTolerateUndeclaredReads(t *testing.T) {
	// Reading an option the scenario never declared is a bug in the scenario. It
	// must report rather than take the process down mid-run.
	info := &ScenarioInfo{ScenarioName: "buggy"}
	if got := info.OptionInt("nope"); got != 0 {
		t.Errorf("int: want zero value, got %v", got)
	}
	if got := info.OptionString("nope"); got != "" {
		t.Errorf("string: want empty, got %q", got)
	}
	if got := info.OptionDuration("nope"); got != 0 {
		t.Errorf("duration: want zero, got %v", got)
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
