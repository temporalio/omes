package workers

import (
	"slices"
	"testing"
)

func TestWithEnvClearsInheritedEnvVar(t *testing.T) {
	const name = "TEST_ENV_VAR"

	environ := []string{
		"PATH=/bin",
		name + "=ambient",
		"HOME=/tmp",
	}

	got := withEnv(environ, name, "")

	want := []string{
		"PATH=/bin",
		"HOME=/tmp",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestWithEnvSetsExplicitEnvVar(t *testing.T) {
	const name = "TEST_ENV_VAR"

	environ := []string{
		"PATH=/bin",
		"HOME=/tmp",
	}

	got := withEnv(environ, name, "explicit")

	want := []string{
		"PATH=/bin",
		"HOME=/tmp",
		name + "=explicit",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestWithEnvExplicitEnvVarOverridesInheritedEnvVar(t *testing.T) {
	const name = "TEST_ENV_VAR"

	environ := []string{
		"PATH=/bin",
		name + "=ambient",
		"HOME=/tmp",
	}

	got := withEnv(environ, name, "explicit")

	want := []string{
		"PATH=/bin",
		"HOME=/tmp",
		name + "=explicit",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
