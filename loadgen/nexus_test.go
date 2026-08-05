package loadgen

import (
	"strings"
	"testing"

	"go.temporal.io/api/workflowservice/v1"
)

// Capability resolution is exercised here by handing over capabilities directly.
// Fetching them needs a WorkflowService stub, which loadgen has no fake for, so
// the dev-server test in scenarios covers that end to end.

func nexusOptions(t *testing.T, provided map[string]string) *OptionSet {
	t.Helper()
	s := &Scenario{Options: DeclareNexusOptions}
	set, err := s.ResolveOptions(provided)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return set
}

// omes is a testing tool, so the useful default is to exercise Nexus wherever the
// server can. That makes nexus-enabled capability-gated rather than plainly true.
func TestNexusEnabledFollowsServerCapability(t *testing.T) {
	for _, supported := range []bool{true, false} {
		set := nexusOptions(t, nil)
		err := set.ResolveFeaturesFromCapabilities(Capabilities{
			System: &workflowservice.GetSystemInfoResponse_Capabilities{Nexus: supported},
		})
		if err != nil {
			t.Fatalf("resolve features: %v", err)
		}
		info := &ScenarioInfo{Options: set}
		if got := info.OptionBool(NexusEnabledOption); got != supported {
			t.Errorf("server Nexus support %v: want nexus-enabled %v, got %v", supported, supported, got)
		}
	}
}

// Asking for Nexus on a server without it cannot be honored, so it fails rather
// than quietly running with less load than requested.
func TestNexusEnabledExplicitlyOnUnsupportedServerFails(t *testing.T) {
	set := nexusOptions(t, map[string]string{NexusEnabledOption: "true"})
	err := set.ResolveFeaturesFromCapabilities(Capabilities{
		System: &workflowservice.GetSystemInfoResponse_Capabilities{Nexus: false},
	})
	if err == nil {
		t.Fatal("expected asking for Nexus against a server without it to fail")
	}
	if !strings.Contains(err.Error(), NexusEnabledOption) {
		t.Errorf("error should name the option, got: %v", err)
	}
}

func TestNexusEnabledShownAsCapabilityGated(t *testing.T) {
	for _, o := range (&Scenario{Options: DeclareNexusOptions}).DeclaredOptions() {
		if o.Name != NexusEnabledOption {
			continue
		}
		if !o.IsFeature {
			t.Error("nexus-enabled is gated on the server supporting Nexus, so it is a feature option")
		}
		if o.Default != featureDefaultDisplay {
			t.Errorf("list-scenarios should show the dynamic default, got %q", o.Default)
		}
		return
	}
	t.Fatal("nexus-enabled should be declared")
}

func TestResolveNexusConfigRejectsEndpointWithoutNexus(t *testing.T) {
	info := &ScenarioInfo{Options: nexusOptions(t, map[string]string{
		NexusEnabledOption:  "false",
		NexusEndpointOption: "some-endpoint",
	})}
	// Returns before touching the client, so a nil one is fine here.
	_, err := info.ResolveNexusConfig(t.Context())
	if err == nil {
		t.Fatal("naming an endpoint for a run that generates no Nexus load should be rejected")
	}
	if !strings.Contains(err.Error(), NexusEndpointOption) {
		t.Errorf("error should name the offending option, got: %v", err)
	}
}

func TestResolveNexusConfigDisabledYieldsNoEndpoint(t *testing.T) {
	info := &ScenarioInfo{Options: nexusOptions(t, map[string]string{NexusEnabledOption: "false"})}
	cfg, err := info.ResolveNexusConfig(t.Context())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Enabled || cfg.Endpoint != "" {
		t.Errorf("Nexus off should yield an empty config, got %+v", cfg)
	}
}

func TestNexusEndpointNamePrefersTheNamedEndpoint(t *testing.T) {
	named := &ScenarioInfo{
		RunID:   "run-1",
		Options: nexusOptions(t, map[string]string{NexusEndpointOption: "mine"}),
	}
	if got := named.NexusEndpointName(); got != "mine" {
		t.Errorf("want the named endpoint, got %q", got)
	}

	// Unnamed falls back to the run's conventional endpoint, which is what both
	// the endpoint creation and the generated actions have to agree on.
	unnamed := &ScenarioInfo{RunID: "run-1", Options: nexusOptions(t, nil)}
	if got, want := unnamed.NexusEndpointName(), NexusEndpointForRun("run-1"); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestIncludeNexusSubFeature(t *testing.T) {
	const sub = "include-standalone-nexus"
	declare := func(o *OptionSet) {
		DeclareNexusOptions(o)
		o.Bool(sub, false, "")
	}

	cases := []struct {
		name        string
		provided    map[string]string
		nexusOn     bool
		wantInclude bool
		wantErr     string
	}{
		{name: "off stays off", nexusOn: true},
		{name: "on with nexus", provided: map[string]string{sub: "true"}, nexusOn: true, wantInclude: true},
		{
			name:     "explicitly on without nexus is a contradiction",
			provided: map[string]string{sub: "true"},
			wantErr:  "requires",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Scenario{Options: declare}
			set, err := s.ResolveOptions(tc.provided)
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			info := &ScenarioInfo{Options: set}

			include, err := IncludeNexusSubFeature(info, sub, tc.nexusOn)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("want an error containing %q, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if include != tc.wantInclude {
				t.Errorf("include: want %v, got %v", tc.wantInclude, include)
			}
		})
	}
}

// A sub-feature switched on by a capability probe rather than by the user is
// dropped when there is no Nexus load, instead of failing the run.
func TestIncludeNexusSubFeatureDropsAutoEnabledWithoutNexus(t *testing.T) {
	const sub = "include-standalone-nexus"
	s := &Scenario{Options: func(o *OptionSet) {
		DeclareNexusOptions(o)
		o.Bool(sub, false, "")
	}}
	set, err := s.ResolveOptions(nil)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// Stand in for what a probe would have written for an unset feature option.
	if err := set.FlagSet.Set(sub, "true"); err != nil {
		t.Fatalf("set: %v", err)
	}
	info := &ScenarioInfo{Options: set}

	include, err := IncludeNexusSubFeature(info, sub, false)
	if err != nil {
		t.Fatalf("an auto-enabled sub-feature should not fail the run: %v", err)
	}
	if include {
		t.Error("the sub-feature should be dropped when Nexus is off")
	}
}
