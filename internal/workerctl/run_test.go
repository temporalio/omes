package workerctl

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/temporalio/omes/clioptions"
)

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

func TestPassthroughPreservesRepeatedServerAddresses(t *testing.T) {
	var opts clioptions.ClientOptions
	fs := opts.FlagSet()
	if err := fs.Set("server-address", "127.0.0.1:7234"); err != nil {
		t.Fatal(err)
	}
	if err := fs.Set("server-address", "127.0.0.1:8234"); err != nil {
		t.Fatal(err)
	}

	got := passthrough(fs, "")
	want := []string{
		"--server-address=127.0.0.1:7234",
		"--server-address=127.0.0.1:8234",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestRunnerRejectsMultipleServerAddressesForNonGoWorkers(t *testing.T) {
	r := Runner{
		Builder: Builder{
			SdkOptions: clioptions.SdkOptions{Language: clioptions.LangTypeScript},
		},
		TaskQueueName: "omes",
		ClientOptions: clioptions.ClientOptions{
			Addresses: []string{"127.0.0.1:7234", "127.0.0.1:8234"},
		},
	}

	err := r.Run(context.Background(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "multiple server addresses") {
		t.Fatalf("expected multiple server addresses error, got %v", err)
	}
}
