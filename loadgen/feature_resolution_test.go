package loadgen

import (
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// featureScenario declares a single capability-gated option, whose capability the
// namespace is taken to support or not per supported.
func featureScenario(supported bool) *Scenario {
	return &Scenario{Options: func(o *OptionSet) {
		o.Feature("gated", "", func(Capabilities) bool { return supported })
	}}
}

// observedInfo returns a ScenarioInfo logging into a buffer, so a test can assert
// on what a scenario author or operator would actually see.
func observedInfo(t *testing.T, set *OptionSet) (*ScenarioInfo, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	return &ScenarioInfo{
		ScenarioName: "s",
		Namespace:    "ns",
		Options:      set,
		Logger:       zap.New(core).Sugar(),
	}, logs
}

func loggedMessages(logs *observer.ObservedLogs) string {
	var b strings.Builder
	for _, entry := range logs.All() {
		b.WriteString(entry.Message)
		b.WriteString("\n")
	}
	return b.String()
}

// A library caller that never probes reads false for every feature, so the run
// quietly omits the load instead of failing. That has to be reported.
func TestUnresolvedFeatureReadIsReported(t *testing.T) {
	set, err := featureScenario(true).ResolveOptions(nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	info, logs := observedInfo(t, set)

	if info.OptionBool("gated") {
		t.Error("an unresolved feature reads false; that is the value being warned about")
	}
	got := loggedMessages(logs)
	if !strings.Contains(got, "before resolving capabilities") {
		t.Errorf("reading an unresolved feature should be reported, got: %q", got)
	}
	if !strings.Contains(got, "ResolveFeatureOptions") {
		t.Errorf("the report should name the call that fixes it, got: %q", got)
	}
}

func TestResolvedFeatureReadIsQuiet(t *testing.T) {
	set, err := featureScenario(true).ResolveOptions(nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := set.resolveFeatures(Capabilities{}); err != nil {
		t.Fatalf("resolveFeatures: %v", err)
	}
	info, logs := observedInfo(t, set)

	if !info.OptionBool("gated") {
		t.Error("a resolved feature should report the capability's value")
	}
	if got := loggedMessages(logs); got != "" {
		t.Errorf("a resolved read should log nothing, got: %q", got)
	}
}

// An explicit value is authoritative with or without a probe, so warning about it
// would be noise.
func TestExplicitFeatureReadIsQuietWithoutResolution(t *testing.T) {
	set, err := featureScenario(true).ResolveOptions(map[string]string{"gated": "true"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	info, logs := observedInfo(t, set)

	if !info.OptionBool("gated") {
		t.Error("an explicitly supplied feature should keep the supplied value")
	}
	if got := loggedMessages(logs); got != "" {
		t.Errorf("an explicit value needs no probe, so it should log nothing, got: %q", got)
	}
}

// Set is how library callers and tests supply a value directly; it has to count as
// deliberate, or resolution would overwrite it and reads would be reported.
func TestSetMarksOptionAsUserSpecified(t *testing.T) {
	set, err := featureScenario(false).ResolveOptions(nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := set.Set("gated", "true"); err != nil {
		t.Fatalf("set: %v", err)
	}
	if !set.UserSpecified("gated") {
		t.Fatal("a value supplied through Set should count as user-specified")
	}

	// Resolution must reject it rather than silently flip it to false.
	err = set.resolveFeatures(Capabilities{})
	if err == nil {
		t.Fatal("enabling a feature the namespace lacks should fail, however it was supplied")
	}
	if !strings.Contains(err.Error(), "explicitly enabled") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUnresolvedFeatureReadFallsBackWithoutLogger(t *testing.T) {
	set, err := featureScenario(true).ResolveOptions(nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// A nil Logger must not panic: library callers build ScenarioInfo by hand.
	info := &ScenarioInfo{ScenarioName: "s", Namespace: "ns", Options: set}
	info.OptionBool("gated")
}

func TestFailedResolutionLeavesFeaturesUnresolved(t *testing.T) {
	// A rejected resolution must not mark the set resolved, or a later read would
	// look trustworthy after the run had already been told it was wrong.
	set, err := featureScenario(false).ResolveOptions(map[string]string{"gated": "true"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if err := set.resolveFeatures(Capabilities{}); err == nil {
		t.Fatal("expected resolution to reject an unsupported explicit enable")
	}
	if set.featuresResolved {
		t.Error("a failed resolution should not mark the set resolved")
	}
}

func TestRequiredFeatureIsRejected(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Feature("gated", "", func(Capabilities) bool { return true })
		o.MarkRequired("gated")
	}}
	_, err := s.ResolveOptions(map[string]string{"gated": "true"})
	if err == nil {
		t.Fatal("required and capability-gated contradict, so declaring both should fail")
	}
	if !strings.Contains(err.Error(), "contradict") {
		t.Errorf("unexpected error: %v", err)
	}
}
