package loadgen

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestUsageErrorSurvivesWrapping(t *testing.T) {
	// The command layer wraps a scenario error with run context before returning
	// it. A usage error has to stay recognizable through that, or it reverts to
	// being reported as a crash.
	wrapped := fmt.Errorf("scenario failed: %w", NewUsageError("--run-id must not be empty"))

	var usage UsageError
	if !errors.As(wrapped, &usage) {
		t.Fatal("a wrapped usage error should still be detected as one")
	}
	if got := usage.Error(); got != "--run-id must not be empty" {
		t.Errorf("the unwrapped error should carry the message alone, got %q", got)
	}
}

func TestInvalidOptionsErrorIsAUsageError(t *testing.T) {
	err := error(&InvalidOptionsError{ScenarioName: "s", Err: errors.New("boom")})
	var usage UsageError
	if !errors.As(err, &usage) {
		t.Error("a rejected --option is bad input, so it should be a usage error")
	}
}

func TestScenarioNotFoundErrorListsAvailableScenarios(t *testing.T) {
	// MustRegisterScenario names a scenario after its caller's file, so seed the
	// registry directly to get a predictable name to assert on.
	registeredScenarios["zzz_usage_errors_test"] = &Scenario{}
	t.Cleanup(func() { delete(registeredScenarios, "zzz_usage_errors_test") })

	err := NewScenarioNotFoundError("typoed")
	if !strings.Contains(err.Error(), `unknown scenario "typoed"`) {
		t.Errorf("error should name the scenario that was not found, got: %v", err)
	}
	if !strings.Contains(err.Error(), "zzz_usage_errors_test") {
		t.Errorf("error should list registered scenarios, got: %v", err)
	}

	var usage UsageError
	if !errors.As(error(err), &usage) {
		t.Error("an unknown scenario name is bad input, so it should be a usage error")
	}
}

func TestScenarioNotFoundErrorHandlesEmptyName(t *testing.T) {
	err := NewScenarioNotFoundError("")
	if !strings.Contains(err.Error(), "no scenario given") {
		t.Errorf("an empty name should not be quoted back as a name, got: %v", err)
	}
}
