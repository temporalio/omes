package workers

import (
	"slices"
	"strings"
	"testing"
)

func TestWithWorkerProfileEnvSetsProfile(t *testing.T) {
	env := withWorkerProfileEnv([]string{"PATH=/bin", "HOME=/tmp"}, "resource-based-default")

	if !slices.Contains(env, "OMES_WORKER_PROFILE=resource-based-default") {
		t.Fatalf("expected worker profile env, got %#v", env)
	}
}

func TestWithWorkerProfileEnvReplacesExistingProfile(t *testing.T) {
	env := withWorkerProfileEnv(
		[]string{"OMES_WORKER_PROFILE=old", "PATH=/bin", "OMES_WORKER_PROFILE=older"},
		"resource-based-default",
	)

	var matches []string
	for _, item := range env {
		if strings.HasPrefix(item, "OMES_WORKER_PROFILE=") {
			matches = append(matches, item)
		}
	}
	if len(matches) != 1 || matches[0] != "OMES_WORKER_PROFILE=resource-based-default" {
		t.Fatalf("expected one replaced worker profile env var, got %#v", matches)
	}
}

func TestWithWorkerProfileEnvClearsAmbientProfileWhenUnset(t *testing.T) {
	env := withWorkerProfileEnv([]string{"OMES_WORKER_PROFILE=old", "PATH=/bin"}, "")

	for _, item := range env {
		if strings.HasPrefix(item, "OMES_WORKER_PROFILE=") {
			t.Fatalf("expected no worker profile env var, got %#v", env)
		}
	}
}
