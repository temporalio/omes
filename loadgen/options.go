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
// declarations and returns the values the run should use.
//
// Every problem is reported at once rather than one per run attempt: unknown
// option names, values that do not parse as their declared type, and missing
// required options.
//
// Declared defaults describe and validate; they are deliberately not written
// into the returned map. Scenarios read their own defaults through the
// ScenarioOption* accessors, and some treat an option's *absence* as meaningful
// — injecting defaults here would silently change what those scenarios do.
//
// A scenario that declares no options is passed through untouched, which keeps
// scenarios outside this repository working unchanged.
func (s *Scenario) ResolveOptions(provided map[string]string) (map[string]string, error) {
	set := s.optionSet()
	if set == nil {
		return provided, nil
	}

	var errs []error
	for _, name := range sortedKeys(provided) {
		flag := set.Lookup(name)
		if flag == nil {
			errs = append(errs, fmt.Errorf("unknown option %q; this scenario accepts: %s",
				name, strings.Join(set.names(), ", ")))
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

	resolved := make(map[string]string, len(provided))
	set.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			resolved[f.Name] = f.Value.String()
		}
	})
	return resolved, nil
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
