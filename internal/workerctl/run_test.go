package workerctl

import (
	"slices"
	"testing"

	"github.com/temporalio/omes/clioptions"
)

func TestCacheOverridePassthrough(t *testing.T) {
	for _, value := range []string{"50000", "0"} {
		t.Run(value, func(t *testing.T) {
			var options clioptions.WorkerOptions
			flags := options.FlagSet()
			if got := passthrough(flags, "worker-"); len(got) != 0 {
				t.Fatalf("unset options must not override harness defaults: %v", got)
			}
			if err := flags.Set("worker-max-cached-workflows", value); err != nil {
				t.Fatal(err)
			}
			want := []string{"--max-cached-workflows=" + value}
			if got := passthrough(flags, "worker-"); !slices.Equal(got, want) {
				t.Fatalf("expected %v, got %v", want, got)
			}
		})
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
