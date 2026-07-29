package loadgen

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/pflag"
)

// OptionSet declares the options a scenario accepts via `--option <name>=<value>`.
// It embeds a [pflag.FlagSet], so options are declared with the same typed
// registrars as omes's own CLI flags and pflag does the parsing, type checking,
// and default rendering.
type OptionSet struct {
	*pflag.FlagSet
	required map[string]struct{}
}

// MarkRequired fails any run that does not explicitly supply the named options.
func (o *OptionSet) MarkRequired(names ...string) {
	for _, name := range names {
		o.required[name] = struct{}{}
	}
}

func newOptionSet(name string) *OptionSet {
	fs := pflag.NewFlagSet(name+" options", pflag.ContinueOnError)
	// Options are surfaced by `list-scenarios` and validation errors, never by
	// pflag printing its own usage.
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	return &OptionSet{FlagSet: fs, required: make(map[string]struct{})}
}

func (o *OptionSet) names() []string {
	var names []string
	o.VisitAll(func(f *pflag.Flag) { names = append(names, f.Name) })
	sort.Strings(names)
	return names
}

// DeclaredOption describes one declared option for display.
type DeclaredOption struct {
	Name     string
	Type     string
	Default  string
	Usage    string
	Required bool
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
		out = append(out, DeclaredOption{
			Name:     f.Name,
			Type:     f.Value.Type(),
			Default:  f.DefValue,
			Usage:    f.Usage,
			Required: required,
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
		}
	}

	for _, name := range sortedRequired(set.required) {
		flag := set.Lookup(name)
		if flag == nil {
			errs = append(errs, fmt.Errorf("option %q is marked required but not declared", name))
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
