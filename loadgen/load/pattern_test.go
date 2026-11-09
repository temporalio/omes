package load

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompilePatterns(t *testing.T) {
	for _, tc := range []struct {
		name     string
		pattern  string
		duration time.Duration
		params   map[string]string
		want     []Phase
	}{
		{"steady defaults", "steady", 100 * time.Second, nil, []Phase{{"hold", 100 * time.Second, 100, 100}}},
		{"ramp defaults", "ramp-hold-ramp", 100 * time.Second, nil, []Phase{
			{"up", 10 * time.Second, 0, 100}, {"hold", 80 * time.Second, 100, 100}, {"down", 10 * time.Second, 100, 0},
		}},
		{"ramp params", "ramp-hold-ramp", 100 * time.Second, map[string]string{"up": "20%", "down": "30s"}, []Phase{
			{"up", 20 * time.Second, 0, 100}, {"hold", 50 * time.Second, 100, 100}, {"down", 30 * time.Second, 100, 0},
		}},
		{"sweep defaults", "sweep", 100 * time.Second, nil, []Phase{
			{"step-1", 25 * time.Second, 25, 25}, {"step-2", 25 * time.Second, 50, 50}, {"step-3", 25 * time.Second, 75, 75}, {"step-4", 25 * time.Second, 100, 100},
		}},
		{"sweep down", "sweep", 3 * time.Second, map[string]string{"from": "100%", "to": "0.5", "steps": "3"}, []Phase{
			{"step-1", time.Second, 100, 100}, {"step-2", time.Second, 75, 75}, {"step-3", time.Second, 50, 50},
		}},
		{"spike defaults", "spike-train", 21 * time.Minute, nil, []Phase{
			{"base", 10 * time.Minute, 40, 40}, {"spike", time.Minute, 100, 100}, {"base", 9 * time.Minute, 40, 40}, {"spike", time.Minute, 100, 100},
		}},
		{"spike params", "spike-train", 10 * time.Second, map[string]string{"base": "0", "every": "30%", "for": "1s"}, []Phase{
			{"base", 3 * time.Second, 0, 0}, {"spike", time.Second, 100, 100}, {"base", 2 * time.Second, 0, 0}, {"spike", time.Second, 100, 100}, {"base", 2 * time.Second, 0, 0}, {"spike", time.Second, 100, 100},
		}},
		{"truncated spike", "spike-train", 10500 * time.Millisecond, map[string]string{"every": "10s", "for": "1s"}, []Phase{
			{"base", 10 * time.Second, 40, 40}, {"spike", 500 * time.Millisecond, 100, 100},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pattern, err := Compile(tc.pattern, 100, tc.duration, tc.params)
			require.NoError(t, err)
			require.Equal(t, tc.want, pattern.Phases)
		})
	}
}

func TestCompileRejectsInvalidParameters(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		params  map[string]string
		want    string
	}{
		{"other", nil, "valid patterns: steady, ramp-hold-ramp, sweep, spike-train"},
		{"steady", map[string]string{"up": "1s"}, "valid parameters: none"},
		{"sweep", map[string]string{"up": "1s"}, "valid parameters: from, to, steps"},
		{"sweep", map[string]string{"steps": "1025"}, "steps"},
		{"sweep", map[string]string{"steps": "1"}, "steps"},
		{"sweep", map[string]string{"from": "NaN"}, "from"},
		{"sweep", map[string]string{"to": "101%"}, "to"},
		{"ramp-hold-ramp", map[string]string{"up": "60%", "down": "50%"}, "exceed"},
		{"ramp-hold-ramp", map[string]string{"up": "bad"}, "up"},
		{"ramp-hold-ramp", map[string]string{"down": "-1s"}, "down"},
		{"spike-train", map[string]string{"every": "0s"}, "every"},
		{"spike-train", map[string]string{"for": "11m"}, "for"},
		{"spike-train", map[string]string{"every": "1ns", "for": "1ns"}, "1024 phases"},
	} {
		t.Run(tc.pattern+tc.want, func(t *testing.T) {
			_, err := Compile(tc.pattern, 100, time.Hour, tc.params)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestParsePeak(t *testing.T) {
	for _, value := range []string{"120", "120/s", "0.5/s"} {
		peak, err := ParsePeak(value)
		require.NoError(t, err)
		if value == "0.5/s" {
			require.Equal(t, 0.5, peak)
		} else {
			require.Equal(t, float64(120), peak)
		}
	}

	for _, value := range []string{"", "0", "-1", "NaN", "Inf", "1/m", "1e10"} {
		_, err := ParsePeak(value)
		require.ErrorContains(t, err, "--peak")
	}
}
