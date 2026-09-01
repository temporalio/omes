package encryption

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

// fullConfig sets every knob to a distinct non-zero value, so a field read from
// the wrong source is as visible as one left out.
func fullConfig() projectConfig {
	return projectConfig{
		NexusEndpoint:     "endpoint",
		ConcurrentUpdates: 19,
		MemoEntries:       2,
		ActivityCount:     3,
		MarkerCount:       5,
		ChildCount:        7,
		SignalCount:       11,
		NexusCount:        13,
		FailureDepth:      17,
		PayloadBytes:      1024,
	}
}

func TestWithDefaults(t *testing.T) {
	require.Equal(t, projectConfig{
		ConcurrentUpdates: 1, MemoEntries: 1, ActivityCount: 1, MarkerCount: 1,
		ChildCount: 1, SignalCount: 1, NexusCount: 1, FailureDepth: 1,
	}, projectConfig{}.withDefaults())

	clamped := projectConfig{MemoEntries: 25, MarkerCount: -3, PayloadBytes: -1}.withDefaults()
	require.Equal(t, 25, clamped.MemoEntries, "a configured count survives")
	require.Equal(t, 1, clamped.MarkerCount, "a negative count clamps to one")
	require.Zero(t, clamped.PayloadBytes, "a negative size clamps to zero")
}

func TestProjectConfigParsesFlatJSON(t *testing.T) {
	const raw = `{
		"nexusEndpoint": "endpoint",
		"concurrentUpdates": 19,
		"memoEntries": 2,
		"activityCount": 3,
		"markerCount": 5,
		"childCount": 7,
		"signalCount": 11,
		"nexusCount": 13,
		"failureDepth": 17,
		"payloadBytes": 1024
	}`

	var config projectConfig
	require.NoError(t, json.Unmarshal([]byte(raw), &config))
	require.Equal(t, fullConfig(), config)
}

func TestMakeFiller(t *testing.T) {
	require.Empty(t, makeFiller(0))
	require.Empty(t, makeFiller(-5))
	require.Len(t, makeFiller(2048), 2048)
}

// TestNewFailureChain is the point of FailureDepth: each level has to be its own
// Failure proto, so depth adds nodes along Failure.cause rather than making one
// details payload larger. Depth 0 still produces one level, so a failure always
// has a cause.
func TestNewFailureChain(t *testing.T) {
	filler := makeFiller(512)

	for _, tc := range []struct{ depth, wantLevels int }{
		{depth: 0, wantLevels: 1},
		{depth: 1, wantLevels: 1},
		{depth: 4, wantLevels: 4},
	} {
		err := newFailureChain("step", testMessage, filler, tc.depth)

		var levels []FailureDetails
		for err != nil {
			var appErr *temporal.ApplicationError
			require.ErrorAs(t, err, &appErr)
			require.Equal(t, nestedFailureErrorType, appErr.Type())

			var details FailureDetails
			require.NoError(t, appErr.Details(&details))
			levels = append(levels, details)
			err = appErr.Unwrap()
		}

		require.Len(t, levels, tc.wantLevels, "depth %d", tc.depth)
		for i, details := range levels {
			// Level 1 is the outermost cause and level depth the innermost, so
			// the numbering reads outward-in as a reader unwraps.
			require.Equal(t, i+1, details.Level, "depth %d", tc.depth)
			require.Len(t, details.Filler, 512)
			require.Equal(t, testMessage, details.Message)
		}
	}
}
