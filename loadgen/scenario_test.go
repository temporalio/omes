package loadgen

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/operatorservicemock/v1"
	"go.temporal.io/api/serviceerror"
	sdkmocks "go.temporal.io/sdk/mocks"
)

func TestEnsureNexusEndpointReusesExistingEndpoint(t *testing.T) {
	operatorClient := operatorservicemock.NewMockOperatorServiceClient(gomock.NewController(t))
	operatorClient.EXPECT().
		CreateNexusEndpoint(gomock.Any(), gomock.Any()).
		Return(nil, serviceerror.NewAlreadyExists("endpoint already registered"))

	temporalClient := &sdkmocks.Client{}
	temporalClient.On("OperatorService").Return(operatorClient)

	endpoint, created, err := ensureNexusEndpoint(
		context.Background(), temporalClient, "test-namespace", "test-run")

	require.NoError(t, err)
	require.Equal(t, "test-nexus-endpoint-test-run", endpoint)
	require.False(t, created)
}

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
			name:          "positive rate",
			configuration: RunConfiguration{MaxIterationsPerSecond: 1},
		},
		{
			name:          "one nanosecond interval",
			configuration: RunConfiguration{MaxIterationsPerSecond: 1e9},
		},
		{
			name:          "negative rate",
			configuration: RunConfiguration{MaxIterationsPerSecond: -1},
			expectedErr:   "MaxIterationsPerSecond",
		},
		{
			name:          "NaN rate",
			configuration: RunConfiguration{MaxIterationsPerSecond: math.NaN()},
			expectedErr:   "MaxIterationsPerSecond",
		},
		{
			name:          "infinite rate",
			configuration: RunConfiguration{MaxIterationsPerSecond: math.Inf(1)},
			expectedErr:   "MaxIterationsPerSecond",
		},
		{
			name:          "interval below one nanosecond",
			configuration: RunConfiguration{MaxIterationsPerSecond: 1e9 + 1},
			expectedErr:   "MaxIterationsPerSecond",
		},
		{
			name:          "interval overflows duration",
			configuration: RunConfiguration{MaxIterationsPerSecond: 1e-12},
			expectedErr:   "MaxIterationsPerSecond",
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
