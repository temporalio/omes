package loadgen

import (
	"context"
	"fmt"
)

const (
	// NexusEnabledOption governs whether a run generates Nexus load at all.
	NexusEnabledOption = "nexus-enabled"
	// NexusEndpointOption names the endpoint to drive Nexus operations through.
	NexusEndpointOption = "nexus-endpoint"
)

// DeclareNexusOptions declares the options shared by every scenario that can
// generate Nexus load, so the names, defaults, and meaning are defined once.
// Scenarios call it from their own Options declaration.
//
// Nexus is gated on the server supporting it, and enabled by default where it
// does: omes is a testing tool, so the useful default is to exercise what the
// server can do. Set the option false to generate no Nexus load regardless, which
// also keeps the run from creating a Nexus endpoint.
func DeclareNexusOptions(o *OptionSet) {
	o.Feature(NexusEnabledOption, "Generate Nexus load.",
		func(c Capabilities) bool { return c.System.GetNexus() })
	declareNexusEndpointOption(o)
}

func declareNexusEndpointOption(o *OptionSet) {
	o.String(NexusEndpointOption, "", "Nexus endpoint to use. Empty creates one for this run.")
}

// NexusConfig is the resolved Nexus setup for a run.
type NexusConfig struct {
	// Enabled reports whether this run should generate Nexus load. It is false
	// when the option is off and when the server does not support Nexus.
	Enabled bool
	// Endpoint is the endpoint to drive operations through, empty when disabled.
	Endpoint string
}

// NexusEndpointName returns the endpoint this run drives Nexus operations
// through: the one the user named, or the conventional name for the run. It does
// not create anything, so it is safe to call while building generator input.
func (s *ScenarioInfo) NexusEndpointName() string {
	if endpoint := s.OptionString(NexusEndpointOption); endpoint != "" {
		return endpoint
	}
	return NexusEndpointForRun(s.RunID)
}

// ResolveNexusConfig settles whether the run generates Nexus load and against
// which endpoint, creating the endpoint if the run needs one. Call it once the
// client is dialed, and after [ResolveFeatureOptions]: nexus-enabled is gated on
// the server supporting Nexus, so it is that call which decides the default.
func (s *ScenarioInfo) ResolveNexusConfig(ctx context.Context) (NexusConfig, error) {
	endpoint := s.OptionString(NexusEndpointOption)
	if !s.OptionBool(NexusEnabledOption) {
		if endpoint != "" {
			return NexusConfig{}, fmt.Errorf("%s was set but %s is false",
				NexusEndpointOption, NexusEnabledOption)
		}
		return NexusConfig{}, nil
	}

	if endpoint == "" {
		var err error
		endpoint, err = s.EnsureNexusEndpoint(ctx)
		if err != nil {
			return NexusConfig{}, fmt.Errorf("failed to create nexus endpoint: %w", err)
		}
	}
	return NexusConfig{Enabled: true, Endpoint: endpoint}, nil
}

// IncludeNexusSubFeature settles an option that adds to Nexus load rather than
// producing it, such as standalone operations. Nexus being off wins over a
// sub-feature a capability probe switched on, since there is no Nexus load to add
// to; asking for one explicitly with Nexus off is a contradiction and an error.
func IncludeNexusSubFeature(info *ScenarioInfo, option string, nexusEnabled bool) (bool, error) {
	include := info.OptionBool(option)
	if !include || nexusEnabled {
		return include, nil
	}
	if info.OptionUserSpecified(option) {
		return false, fmt.Errorf("%s requires %s", option, NexusEnabledOption)
	}
	info.logf("not including %s: the namespace supports it, but %s is off, so this run generates "+
		"no Nexus load", option, NexusEnabledOption)
	return false, nil
}
