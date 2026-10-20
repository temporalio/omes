package workerctl

import (
	"slices"
	"testing"

	"github.com/spf13/pflag"
)

func TestWorkerMetricsArgs(t *testing.T) {
	// Every registered metrics flag must have an explicit routing expectation.
	// In particular, prom-snapshot is unprefixed even in the worker flag set.
	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{"worker-prom-listen-address", "127.0.0.1:9090", []string{"--prom-listen-address=127.0.0.1:9090"}},
		{"worker-prom-handler-path", "/sdk-metrics", []string{"--prom-handler-path=/sdk-metrics"}},
		{"worker-process-metrics-address", "127.0.0.1:9091", nil},
		{"worker-metrics-version-tag", "sdk-ref", nil},
		{"worker-prom-instance-addr", "http://127.0.0.1:9092", nil},
		{"worker-prom-instance-config", "test-prom-config.yml", nil},
		{"prom-snapshot", "true", nil},
		{"worker-prom-export-worker-metrics", "test-export.json", nil},
		{"worker-prom-export-worker-job", "test-worker", nil},
		{"worker-prom-export-process-job", "test-process", nil},
		{"worker-prom-export-metrics-step", "3s", nil},
		{"worker-prom-export-worker-info-address", "127.0.0.1:9091", nil},
	}

	var runner Runner
	all := &runner.MetricsOptions
	fs := all.FlagSet("worker-")
	if fs.NFlag() != 0 || len(workerMetricsArgs(all)) != 0 {
		t.Fatal("unchanged defaults must not be forwarded")
	}
	covered := make(map[string]bool)
	for _, test := range tests {
		covered[test.name] = true
		t.Run(test.name, func(t *testing.T) {
			var runner Runner
			options := &runner.MetricsOptions
			if err := options.FlagSet("worker-").Set(test.name, test.value); err != nil {
				t.Fatal(err)
			}
			if got := workerMetricsArgs(options); !slices.Equal(got, test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
		if err := fs.Set(test.name, test.value); err != nil {
			t.Fatal(err)
		}
	}
	fs.VisitAll(func(flag *pflag.Flag) {
		if !covered[flag.Name] {
			t.Errorf("missing routing expectation for metrics flag %s", flag.Name)
		}
	})
	want := []string{"--prom-handler-path=/sdk-metrics", "--prom-listen-address=127.0.0.1:9090"}
	if got := workerMetricsArgs(all); !slices.Equal(got, want) {
		t.Fatalf("all flags set: expected %v, got %v", want, got)
	}
}

func TestWithEnv(t *testing.T) {
	const name = "TEST_ENV_VAR"

	tests := []struct {
		name    string
		environ []string
		value   string
		want    []string
	}{
		{
			name: "clears inherited var",
			environ: []string{
				"PATH=/bin",
				name + "=ambient",
				"HOME=/tmp",
			},
			want: []string{
				"PATH=/bin",
				"HOME=/tmp",
			},
		},
		{
			name: "sets explicit var",
			environ: []string{
				"PATH=/bin",
				"HOME=/tmp",
			},
			value: "explicit",
			want: []string{
				"PATH=/bin",
				"HOME=/tmp",
				name + "=explicit",
			},
		},
		{
			name: "overrides inherited var",
			environ: []string{
				"PATH=/bin",
				name + "=ambient",
				"HOME=/tmp",
			},
			value: "explicit",
			want: []string{
				"PATH=/bin",
				"HOME=/tmp",
				name + "=explicit",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := withEnv(test.environ, name, test.value)
			if !slices.Equal(got, test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}
