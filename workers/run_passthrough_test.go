package workers

import (
	"strings"
	"testing"

	"github.com/temporalio/omes/cmd/clioptions"
)

// The prepared worker runs as a child process and registers the same WorkerOptions flag
// set this process forwards from, so every forwarded flag must parse against that set.
//
// This drives Runner.workerArgs rather than calling passthrough directly: the bug was the
// prefix argument at the call site, not the helper, so a test that hardcodes the prefix
// proves nothing. Forwarding --max-concurrent-workflow-pollers to a binary accepting only
// --worker-max-concurrent-workflow-pollers killed the worker at startup, silently, for
// every worker.* knob except --build-id.
func TestWorkerArgsParseAgainstChildFlagSet(t *testing.T) {
	var r Runner
	r.SdkOptions.Language = clioptions.LangGo
	r.TaskQueueName = "omes-test"

	want := map[string]string{
		"worker-max-concurrent-workflow-pollers": "32",
		"worker-max-concurrent-workflow-tasks":   "500",
		"worker-max-concurrent-activity-pollers": "4",
		"worker-max-concurrent-activities":       "8",
		"worker-workflow-poller-autoscale-max":   "64",
		"build-id":                               "abc123",
	}
	for name, val := range want {
		if err := r.WorkerOptions.FlagSet().Set(name, val); err != nil {
			t.Fatalf("Set(%q): %v", name, err)
		}
	}

	args := r.workerArgs()

	// Keep only the flag-shaped tail; --task-queue and its value are positional-ish here.
	var flags []string
	for _, a := range args {
		if strings.HasPrefix(a, "--") && strings.Contains(a, "=") {
			flags = append(flags, a)
		}
	}
	if len(flags) == 0 {
		t.Fatalf("no flags forwarded, got args %v", args)
	}

	// A fresh WorkerOptions stands in for the child, which registers the same flag set.
	var child clioptions.WorkerOptions
	childFS := child.FlagSet()
	for _, f := range flags {
		name := strings.SplitN(strings.TrimPrefix(f, "--"), "=", 2)[0]
		if _, ok := want[name]; !ok {
			continue // flags from other option groups; not this test's contract
		}
		if childFS.Lookup(name) == nil {
			t.Errorf("forwarded %q is not a flag the prepared worker accepts", f)
		}
	}

	// Guard the exact regression shape.
	for _, f := range flags {
		if strings.HasPrefix(f, "--max-concurrent") || strings.HasPrefix(f, "--activities-per-second") {
			t.Errorf("forwarded flag %q lost its worker- prefix", f)
		}
	}
}
