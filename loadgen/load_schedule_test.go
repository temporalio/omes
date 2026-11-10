package loadgen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/load"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoadStartupConcurrencyWarning(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cap     int
		pattern bool
		want    []map[string]interface{}
	}{
		{"too small", 2, true, []map[string]interface{}{{"maxConcurrent": int64(2), "peak": float64(120), "assumedIterationLatency": time.Second, "suggestedMaxConcurrent": float64(120)}}},
		{"sufficient", 120, true, nil},
		{"legacy", 2, false, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			core, logs := observer.New(zap.InfoLevel)
			config := RunConfiguration{Duration: time.Minute, MaxConcurrent: tc.cap}
			if tc.pattern {
				pattern, err := load.Compile("steady", 120, time.Minute, nil)
				require.NoError(t, err)
				config.LoadPattern = pattern
			}

			_, err := newLoadSchedule(config, zap.New(core).Sugar())
			require.NoError(t, err)
			var warnings []map[string]interface{}
			for _, entry := range logs.FilterLevelExact(zap.WarnLevel).All() {
				require.Equal(t, "--max-concurrent may limit peak load; size it for peak × iteration latency", entry.Message)
				warnings = append(warnings, entry.ContextMap())
			}
			require.Equal(t, tc.want, warnings)
		})
	}
}

func TestLoadStartupCanonicalDefaults(t *testing.T) {
	pattern, err := load.Parse("spike-train peak=120/s every=5m", 6*time.Minute)
	require.NoError(t, err)
	core, logs := observer.New(zap.InfoLevel)

	_, err = newLoadSchedule(RunConfiguration{Duration: pattern.Duration, LoadPattern: pattern, MaxConcurrent: 120}, zap.New(core).Sugar())
	require.NoError(t, err)
	entries := logs.All()
	require.Len(t, entries, 1)
	require.Equal(t, map[string]interface{}{
		"pattern": "spike-train,peak=120/s,base=0.4,every=5m,for=1m", "peak": float64(120), "duration": 6 * time.Minute,
		"phases": []interface{}{"base [0s,5m0s) 48→48 iterations/s", "spike [5m0s,6m0s) 120→120 iterations/s"},
	}, entries[0].ContextMap())
}
