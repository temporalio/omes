package loadgen

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	namespacev1 "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
)

// Capabilities is what a server reports about what it can do, as read by a
// [OptionSet.Feature] predicate. Both fields may be nil for a server that
// reports nothing, so read them through the generated getters.
type Capabilities struct {
	// Namespace holds the capabilities of the namespace under test, from
	// DescribeNamespace. Use it for anything gated per namespace, such as a
	// dynamic config flag.
	Namespace *namespacev1.NamespaceInfo_Capabilities
	// System holds the capabilities of the server as a whole, from GetSystemInfo.
	System *workflowservice.GetSystemInfoResponse_Capabilities
}

// OptionSet declares the options a scenario accepts via `--option <name>=<value>`.
// It embeds a [pflag.FlagSet], so options are declared with the same typed
// registrars as omes's own CLI flags and pflag does the parsing, type checking,
// and default rendering.
type OptionSet struct {
	*pflag.FlagSet
	required map[string]struct{}
	// explicit tracks which names were supplied via `--option`, as distinct from
	// left at their declared default. pflag's own Flag.Changed is unusable here:
	// resolveFeatures writes to the flag for names left unset, which would make
	// Changed true for those too.
	explicit map[string]struct{}
	// features holds the capability predicate for each option declared with
	// Feature, keyed by name.
	features map[string]func(Capabilities) bool
	// featuresResolved records that the set has been reconciled against reported
	// capabilities. Until it is, an unset feature option reads false whether or not
	// it is supported, which is the wrong answer rather than a missing one — so
	// reads before then are reported.
	featuresResolved bool
}

// Set records the value as deliberately chosen, on top of pflag's own
// bookkeeping. Values that arrive this way are authoritative and are left alone
// by capability resolution, so a caller that sets an option directly gets the
// same treatment as a user passing `--option`.
func (o *OptionSet) Set(name, value string) error {
	if err := o.FlagSet.Set(name, value); err != nil {
		return err
	}
	o.explicit[name] = struct{}{}
	return nil
}

// MarkRequired fails any run that does not explicitly supply the named options.
func (o *OptionSet) MarkRequired(names ...string) {
	for _, name := range names {
		o.required[name] = struct{}{}
	}
}

// Feature declares a boolean option gated on a reported capability. It is enabled
// by default when the server reports support, disabled when it does not, and
// always follows an explicit `--option name=<bool>` instead. Explicitly enabling a
// feature that is not reported fails the run.
//
// The predicate reads [Capabilities], so it can gate on what the namespace
// reports, what the server as a whole reports, or both.
//
// Resolution happens after the client dials, via [ResolveFeatureOptions]; until
// then the option reads as its pflag default (false), and reading it is reported
// as a bug in the caller.
func (o *OptionSet) Feature(name, usage string, supported func(Capabilities) bool) {
	o.Bool(name, false, usage)
	o.features[name] = supported
}

// UserSpecified reports whether name was supplied via `--option name=value`, as opposed
// to being read at its declared (or feature-resolved) default.
func (o *OptionSet) UserSpecified(name string) bool {
	_, ok := o.explicit[name]
	return ok
}

// hasFeatures reports whether the set declares any Feature options. A scenario
// that declares none never needs a capability probe.
func (o *OptionSet) hasFeatures() bool {
	return len(o.features) > 0
}

// sortedFeatureNames returns the declared feature names in sorted order, for
// deterministic error and log ordering.
func (o *OptionSet) sortedFeatureNames() []string {
	names := make([]string, 0, len(o.features))
	for name := range o.features {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ResolveFeaturesFromCapabilities finalizes feature options against capabilities
// the caller already holds, so a caller that has already queried the server does
// not pay for it twice. Prefer [ResolveFeatureOptions], which fetches them.
func (o *OptionSet) ResolveFeaturesFromCapabilities(caps Capabilities) error {
	return o.resolveFeatures(caps)
}

// resolveFeatures finalizes every Feature option against the reported
// capabilities: an unset option takes the capability's value, and an explicit
// true against an unsupported capability is collected as an error. Every
// unsupported feature is reported at once.
func (o *OptionSet) resolveFeatures(caps Capabilities) error {
	var errs []error
	for _, name := range o.sortedFeatureNames() {
		supported := o.features[name](caps)
		flag := o.Lookup(name)
		if !o.UserSpecified(name) {
			if err := flag.Value.Set(strconv.FormatBool(supported)); err != nil {
				errs = append(errs, fmt.Errorf("option %q: %w", name, err))
			}
			continue
		}
		if flag.Value.String() == "true" && !supported {
			errs = append(errs, fmt.Errorf(
				"option %q was explicitly enabled, but the server does not report support for it",
				name))
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	o.featuresResolved = true
	return nil
}

// unresolvedFeature reports whether reading name would return a value that has
// not been reconciled against reported capabilities. An explicitly supplied
// value is authoritative on its own, so only an untouched option qualifies.
func (o *OptionSet) unresolvedFeature(name string) bool {
	if o.featuresResolved || o.UserSpecified(name) {
		return false
	}
	_, isFeature := o.features[name]
	return isFeature
}

func newOptionSet(name string) *OptionSet {
	fs := pflag.NewFlagSet(name+" options", pflag.ContinueOnError)
	// Options are surfaced by `list-scenarios` and validation errors, never by
	// pflag printing its own usage.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return &OptionSet{
		FlagSet:  fs,
		required: make(map[string]struct{}),
		explicit: make(map[string]struct{}),
		features: make(map[string]func(Capabilities) bool),
	}
}

func (o *OptionSet) names() []string {
	var names []string
	o.VisitAll(func(f *pflag.Flag) { names = append(names, f.Name) })
	sort.Strings(names)
	return names
}

// featureDefaultDisplay is the [DeclaredOption.Default] shown for a Feature
// option: its real default depends on what the server reports, not a constant.
const featureDefaultDisplay = "auto (enabled where supported)"

// DeclaredOption describes one declared option for display.
type DeclaredOption struct {
	Name     string
	Type     string
	Default  string
	Usage    string
	Required bool
	// IsFeature is true for an option declared with Feature: its default is
	// resolved from reported capabilities rather than fixed at declaration time.
	IsFeature bool
}

// DeclaredOptions returns the scenario's declared options, sorted by name. It is
// empty for a scenario that declares none.
func (s *Scenario) DeclaredOptions() []DeclaredOption {
	set := s.optionSet()
	if set == nil {
		return nil
	}
	var out []DeclaredOption
	set.VisitAll(func(f *pflag.Flag) {
		_, required := set.required[f.Name]
		_, isFeature := set.features[f.Name]
		def := f.DefValue
		if isFeature {
			def = featureDefaultDisplay
		}
		out = append(out, DeclaredOption{
			Name:      f.Name,
			Type:      f.Value.Type(),
			Default:   def,
			Usage:     f.Usage,
			Required:  required,
			IsFeature: isFeature,
		})
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// optionSet builds a fresh set from the scenario's declaration, or nil if it
// declares no options.
func (s *Scenario) optionSet() *OptionSet {
	if s.Options == nil {
		return nil
	}
	set := newOptionSet(s.Description)
	s.Options(set)
	return set
}

// InvalidOptionsError reports `--option` values a scenario rejects. It exists so
// callers can present a bad option as the usage error it is, rather than as a
// crash.
type InvalidOptionsError struct {
	ScenarioName string
	Err          error
}

func (e *InvalidOptionsError) Error() string {
	return fmt.Sprintf("invalid --option values for scenario %q:\n%v", e.ScenarioName, e.Err)
}

func (e *InvalidOptionsError) Unwrap() error { return e.Err }

// ResolveOptions validates user-supplied options against the scenario's
// declarations and returns the set the run should read from.
//
// Every problem is reported at once rather than one per run attempt: unknown
// option names, values that do not parse as their declared type, and missing
// required options. A scenario that declares no options accepts none.
//
// The returned set holds each option's declared default, overwritten by whatever
// the user supplied — so the declaration is the single source of truth for both
// the value and the default.
func (s *Scenario) ResolveOptions(provided map[string]string) (*OptionSet, error) {
	set := s.optionSet()
	if set == nil {
		set = newOptionSet("")
	}

	var errs []error
	for _, name := range sortedKeys(provided) {
		flag := set.Lookup(name)
		if flag == nil {
			accepted := set.names()
			if len(accepted) == 0 {
				errs = append(errs, fmt.Errorf("unknown option %q; this scenario accepts no options", name))
			} else {
				errs = append(errs, fmt.Errorf("unknown option %q; this scenario accepts: %s",
					name, strings.Join(accepted, ", ")))
			}
			continue
		}
		// pflag does the type checking, but phrases its error for a `--flag`;
		// users type `--option name=value`, so report it in those terms.
		if err := set.Set(name, provided[name]); err != nil {
			errs = append(errs, fmt.Errorf("option %q expects type %s, got %q",
				name, flag.Value.Type(), provided[name]))
			continue
		}
	}

	for _, name := range sortedRequired(set.required) {
		flag := set.Lookup(name)
		if flag == nil {
			errs = append(errs, fmt.Errorf("option %q is marked required but not declared", name))
			continue
		}
		if _, isFeature := set.features[name]; isFeature {
			// Required means the user must supply it; a feature supplies itself from
			// the namespace. Declaring both is a scenario bug, not user error.
			errs = append(errs, fmt.Errorf(
				"option %q is declared as both required and capability-gated, which contradict", name))
			continue
		}
		if !flag.Changed {
			errs = append(errs, fmt.Errorf("option %q is required", name))
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return set, nil
}

// MustResolveOptions is [Scenario.ResolveOptions] for tests and other callers that
// treat a bad option as a programming error. It panics instead of returning one.
func (s *Scenario) MustResolveOptions(provided map[string]string) *OptionSet {
	set, err := s.ResolveOptions(provided)
	if err != nil {
		panic(err)
	}
	return set
}

// MustResolveScenarioOptions resolves options for a registered scenario by name,
// for building a [ScenarioInfo] in tests. Resolving through the real declarations
// means a test that misspells an option fails rather than silently testing a
// default.
func MustResolveScenarioOptions(scenarioName string, provided map[string]string) *OptionSet {
	s := GetScenario(scenarioName)
	if s == nil {
		panic(fmt.Errorf("no registered scenario named %q", scenarioName))
	}
	return s.MustResolveOptions(provided)
}

func sortedRequired(m map[string]struct{}) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
