package loadgen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestScenarioConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		configuration RunConfiguration
		expectedErr   string
	}{
		{
			name:          "default",
			configuration: RunConfiguration{},
			expectedErr:   "",
		},
		{
			name:          "only iterations",
			configuration: RunConfiguration{Iterations: 3},
			expectedErr:   "",
		},
		{
			name:          "start iterations smaller than iterations",
			configuration: RunConfiguration{Iterations: 10, StartFromIteration: 3},
			expectedErr:   "",
		},
		{
			name:          "start iterations larger than iterations",
			configuration: RunConfiguration{Iterations: 3, StartFromIteration: 10},
			expectedErr:   "StartFromIteration 10 is greater than Iterations 3",
		},
		{
			name:          "only duration",
			configuration: RunConfiguration{Duration: time.Hour},
			expectedErr:   "",
		},
		{
			name:          "negative duration",
			configuration: RunConfiguration{Duration: -time.Second},
			expectedErr:   "Duration cannot be negative",
		},
		{
			name:          "both duration and iterations",
			configuration: RunConfiguration{Duration: 3 * time.Second, Iterations: 3},
			expectedErr:   "iterations and duration are mutually exclusive",
		},
		{
			name:          "both duration and start iteration (allowed)",
			configuration: RunConfiguration{Duration: 3 * time.Second, StartFromIteration: 3},
			expectedErr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.configuration.Validate()
			if tt.expectedErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
