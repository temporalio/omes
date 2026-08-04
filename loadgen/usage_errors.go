package loadgen

import (
	"fmt"
	"sort"
	"strings"
)

// UsageError marks an error as caused by bad input rather than by a failure
// during a run.
//
// The distinction matters at the point where omes exits. A stack trace taken
// there describes omes's own call stack, not the mistake the user made, so
// printing one for a bad flag buries the message that would fix it. Callers
// detect this interface with errors.As and print the message alone.
type UsageError interface {
	error
	isUsageError()
}

// NewUsageError builds a [UsageError] with a formatted message, for input
// problems that need no structure beyond what they say.
func NewUsageError(format string, args ...any) UsageError {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

type usageError struct{ msg string }

func (e *usageError) Error() string { return e.msg }

func (e *usageError) isUsageError() {}

// ScenarioNotFoundError reports a scenario name that is not registered, and
// lists the names that are.
type ScenarioNotFoundError struct {
	Name string
	// Available holds the registered scenario names, sorted.
	Available []string
}

// NewScenarioNotFoundError builds the error for name from the global registry.
func NewScenarioNotFoundError(name string) *ScenarioNotFoundError {
	available := make([]string, 0, len(registeredScenarios))
	for scenarioName := range registeredScenarios {
		available = append(available, scenarioName)
	}
	sort.Strings(available)
	return &ScenarioNotFoundError{Name: name, Available: available}
}

func (e *ScenarioNotFoundError) Error() string {
	if e.Name == "" {
		return fmt.Sprintf("no scenario given; available scenarios: %s",
			strings.Join(e.Available, ", "))
	}
	return fmt.Sprintf("unknown scenario %q; available scenarios: %s",
		e.Name, strings.Join(e.Available, ", "))
}

func (e *ScenarioNotFoundError) isUsageError() {}
