package load

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompilePatterns(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pattern   string
		duration  time.Duration
		params    map[string]string
		want      []Phase
		canonical string
	}{
		{"steady defaults", "steady", 100 * time.Second, nil, []Phase{{"hold", 100 * time.Second, 100, 100}}, "steady,peak=100/s"},
		{"ramp defaults", "ramp-hold-ramp", 100 * time.Second, nil, []Phase{
			{"up", 10 * time.Second, 0, 100}, {"hold", 80 * time.Second, 100, 100}, {"down", 10 * time.Second, 100, 0},
		}, "ramp-hold-ramp,peak=100/s,up=0.1,down=0.1"},
		{"ramp params", "ramp-hold-ramp", 100 * time.Second, map[string]string{"up": "0.2", "down": "30s"}, []Phase{
			{"up", 20 * time.Second, 0, 100}, {"hold", 50 * time.Second, 100, 100}, {"down", 30 * time.Second, 100, 0},
		}, "ramp-hold-ramp,peak=100/s,up=0.2,down=30s"},
		{"full ramp", "ramp-hold-ramp", time.Duration(1<<63 - 1), map[string]string{"up": "1", "down": "0"}, []Phase{
			{"up", time.Duration(1<<63 - 1), 0, 100},
		}, "ramp-hold-ramp,peak=100/s,up=1,down=0"},

		{"sweep defaults", "sweep", 100 * time.Second, nil, []Phase{
			{"step-1", 25 * time.Second, 25, 25}, {"step-2", 25 * time.Second, 50, 50}, {"step-3", 25 * time.Second, 75, 75}, {"step-4", 25 * time.Second, 100, 100},
		}, "sweep,peak=100/s,from=0.25,to=1,steps=4"},
		{"sweep down", "sweep", 3 * time.Second, map[string]string{"from": "1", "to": "0.5", "steps": "3"}, []Phase{
			{"step-1", time.Second, 100, 100}, {"step-2", time.Second, 75, 75}, {"step-3", time.Second, 50, 50},
		}, "sweep,peak=100/s,from=1,to=0.5,steps=3"},

		{"spike defaults", "spike-train", 21 * time.Minute, nil, []Phase{
			{"base", 10 * time.Minute, 40, 40}, {"spike", time.Minute, 100, 100}, {"base", 9 * time.Minute, 40, 40}, {"spike", time.Minute, 100, 100},
		}, "spike-train,peak=100/s,base=0.4,every=10m,for=1m"},
		{"spike params", "spike-train", 10 * time.Second, map[string]string{"base": "0", "every": "0.3", "for": "1s"}, []Phase{
			{"base", 3 * time.Second, 0, 0}, {"spike", time.Second, 100, 100}, {"base", 2 * time.Second, 0, 0}, {"spike", time.Second, 100, 100}, {"base", 2 * time.Second, 0, 0}, {"spike", time.Second, 100, 100},
		}, "spike-train,peak=100/s,base=0,every=0.3,for=1s"},
		{"truncated spike", "spike-train", 10500 * time.Millisecond, map[string]string{"every": "10s", "for": "1s"}, []Phase{
			{"base", 10 * time.Second, 40, 40}, {"spike", 500 * time.Millisecond, 100, 100},
		}, "spike-train,peak=100/s,base=0.4,every=10s,for=1s"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pattern, err := Compile(tc.pattern, 100, tc.duration, tc.params)
			require.NoError(t, err)
			require.Equal(t, &Pattern{tc.pattern, 100, tc.duration, tc.want, tc.canonical}, pattern)
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
		{"steady", map[string]string{"up": "1s"}, "valid parameters: peak"},
		{"sweep", map[string]string{"up": "1s"}, "valid parameters: peak, from, to, steps"},

		{"sweep", map[string]string{"steps": "1025"}, "steps"},
		{"sweep", map[string]string{"steps": "1"}, "steps"},
		{"sweep", map[string]string{"from": "NaN"}, "from"},
		{"sweep", map[string]string{"to": "1.01"}, "to"},

		{"ramp-hold-ramp", map[string]string{"up": "0.6", "down": "0.5"}, "exceed"},
		{"ramp-hold-ramp", map[string]string{"up": "bad"}, "up"},
		{"ramp-hold-ramp", map[string]string{"down": "-1s"}, "down"},

		{"spike-train", map[string]string{"every": "0s"}, "every"},
		{"spike-train", map[string]string{"for": "11m"}, "for"},
		{"spike-train", map[string]string{"every": "1ns", "for": "1ns"}, "1024 phases"},

		{"ramp-hold-ramp", map[string]string{"up": "10%"}, "use a fraction such as 0.1"},
		{"spike-train", map[string]string{"every": "10%"}, "use a fraction such as 0.1"},
		{"spike-train", map[string]string{"for": "10%"}, "use a fraction such as 0.1"},
		{"sweep", map[string]string{"from": "25%"}, "use a fraction such as 0.1"},
	} {
		t.Run(tc.pattern+tc.want, func(t *testing.T) {
			_, err := Compile(tc.pattern, 100, time.Hour, tc.params)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestParsePeak(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  float64
	}{
		{"120", 120},
		{"120/s", 120},
		{"0.5/s", 0.5},
		{"10000/s", 10000},
	} {
		peak, err := ParsePeak(tc.value)
		require.NoError(t, err)
		require.Equal(t, tc.want, peak)
	}

	for _, value := range []string{"", "0", "-1", "NaN", "Inf", "1/m", "1e9", "10000.1"} {
		_, err := ParsePeak(value)
		require.ErrorContains(t, err, "peak")
	}
}

func TestParseSpaceSeparated(t *testing.T) {
	comma, err := Parse("spike-train,peak=120/s,every=5m", 20*time.Minute)
	require.NoError(t, err)

	for _, value := range []string{
		"spike-train peak=120/s every=5m",
		"spike-train, peak = 120/s, every = 5m",
		"spike-train,peak=120/s every=5m",
	} {
		spaced, err := Parse(value, 20*time.Minute)
		require.NoError(t, err)
		require.Equal(t, comma, spaced)
	}

	_, err = Parse("spike-train peak=120/s evry=5m", 20*time.Minute)
	require.EqualError(t, err, `unknown parameter "evry" for spike-train (did you mean "every"?); valid parameters: peak, base, every, for; e.g. --load spike-train peak=120/s every=5m`)

	_, err = Parse("steady peak=2 peak=3", time.Minute)
	require.EqualError(t, err, `duplicate --load parameter "peak"; valid parameters: peak; e.g. --load steady peak=120/s`)
}

func TestCompilePeakCeiling(t *testing.T) {
	_, err := Compile("steady", 10000.1, time.Second, nil)
	require.EqualError(t, err, "--load requires a positive finite peak= <= 10000 iterations/s")

	_, err = NewPacer([]Phase{{Name: "hold", Duration: time.Second, From: 10001, To: 10001}})
	require.EqualError(t, err, `invalid load phase "hold": require positive duration and finite nonnegative rates <= 10000/s`)
}
