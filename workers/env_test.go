package workers

import (
	"slices"
	"testing"
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
