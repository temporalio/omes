package scenarios_test

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/facebookgo/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/scenarios"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflow/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/mocks"
	"go.uber.org/zap"
)

func TestExponentialSample(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		lambda         float64
		numSamples     int
		expectedCounts []int
	}{
		{"large lambda", 3.0, 500, []int{480, 19, 1, 0, 0}},
		{"lambda=1.0", 1.0, 500, []int{338, 111, 35, 9, 7}},
		{"small lambda", 1e-9, 500, []int{107, 122, 86, 102, 83}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := rand.New(rand.NewSource(6174))
			counts := make([]int, len(tc.expectedCounts))
			for i := 0; i < tc.numSamples; i++ {
				sample, err := scenarios.ExponentialSample(len(tc.expectedCounts), tc.lambda, g.Float64())
				require.NoError(t, err)
				counts[sample]++
			}
			assert.Equal(t, tc.expectedCounts, counts,
				"Counts should be roughly proportional to the exponential distribution")
		})
	}
}

func TestNewCompletionCallbackScenario(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                    string
		scenarioOptionsOverride func(opts *scenarios.CompletionCallbackScenarioOptions)
		expectedErrSubstring    string
	}{
		{
			name: "zero port",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.StartingPort = 0
			},
			expectedErrSubstring: fmt.Sprintf("%q is required", scenarios.OptionKeyStartingPort),
		},
		{
			name: "zero callback hosts",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.NumCallbackHosts = 0
			},
			expectedErrSubstring: fmt.Sprintf("%q is required", scenarios.OptionKeyNumCallbackHosts),
		},
		{
			name: "negative lambda",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.Lambda = -1.0
			},
			expectedErrSubstring: fmt.Sprintf("%q must be > 0", scenarios.OptionKeyLambda),
		},
		{
			name: "zero half-life",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.HalfLife = 0.0
			},
			expectedErrSubstring: fmt.Sprintf("%q must be > 0", scenarios.OptionKeyHalfLife),
		},
		{
			name: "negative delay distribution max",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.MaxDelay = -1.0
			},
			expectedErrSubstring: fmt.Sprintf("%q must be >= 0s", scenarios.OptionKeyMaxDelay),
		},
		{
			name: "negative error probability max",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.MaxErrorProbability = -1.0
			},
			expectedErrSubstring: fmt.Sprintf("%q must be in [0, 1]", scenarios.OptionKeyMaxErrorProbability),
		},
		{
			name: "error probability max > 1",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.MaxErrorProbability = 1.1
			},
			expectedErrSubstring: fmt.Sprintf("%q must be in [0, 1]", scenarios.OptionKeyMaxErrorProbability),
		},
		{
			name: "no host name",
			scenarioOptionsOverride: func(opts *scenarios.CompletionCallbackScenarioOptions) {
				opts.CallbackHostName = ""
			},
			expectedErrSubstring: fmt.Sprintf("%q is required", scenarios.OptionKeyCallbackHostName),
		},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			scenarioOptions := &scenarios.CompletionCallbackScenarioOptions{
				CallbackHostName:    "localhost",
				StartingPort:        1024,
				NumCallbackHosts:    10,
				Lambda:              1.0,
				HalfLife:            1.0,
				MaxDelay:            5 * time.Second,
				MaxErrorProbability: 0.0,
			}
			if tc.scenarioOptionsOverride != nil {
				tc.scenarioOptionsOverride(scenarioOptions)
			}
			_, err := scenarios.NewCompletionCallbackScenario(scenarioOptions)
			if tc.expectedErrSubstring != "" {
				assert.ErrorContains(t, err, tc.expectedErrSubstring)
			}
		})
	}
}

func TestCompletionCallbackScenario_RunIteration(t *testing.T) {
	t.Parallel()

	// Mock SDK client.
	sdkClient := &mocks.Client{}
	executeWorkflowRequests := make(chan client.StartWorkflowOptions, 1)
	workflowRun := &mocks.WorkflowRun{}
	workflowRun.On("Get", mock.Anything, mock.Anything).Return(nil)
	workflowRun.On("GetID").Return("test-workflow-id")
	workflowRun.On("GetRunID").Return("test-run-id")
	sdkClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		executeWorkflowRequests <- args.Get(1).(client.StartWorkflowOptions)
	}).Return(workflowRun, nil)

	// First call to DescribeWorkflowExecution returns a backing off callback.
	sdkClient.On("DescribeWorkflowExecution", mock.Anything, mock.Anything, mock.Anything).Return(&workflowservice.DescribeWorkflowExecutionResponse{
		Callbacks: []*workflow.CallbackInfo{
			{
				State: enums.CALLBACK_STATE_BACKING_OFF,
			},
		},
	}, nil).Times(1)

	// Advance the clock so that we retry after 1 timer.
	var wg sync.WaitGroup
	clk := clock.NewMock()
	wg.Add(1)
	defer wg.Wait()
	go func() {
		defer wg.Done()
		clk.Wait(clock.Calls{
			Timer: 1,
		})
		clk.Add(time.Second)
	}()

	// Second call to DescribeWorkflowExecution returns a succeeded callback.
	sdkClient.On("DescribeWorkflowExecution", mock.Anything, mock.Anything, mock.Anything).Return(&workflowservice.DescribeWorkflowExecutionResponse{
		Callbacks: []*workflow.CallbackInfo{
			{
				State: enums.CALLBACK_STATE_SUCCEEDED,
			},
		},
	}, nil).Times(1)

	// Create the scenario.
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	opts := &scenarios.CompletionCallbackScenarioOptions{
		Clock:               clk,
		Logger:              logger.Sugar(),
		CallbackHostName:    "localhost",
		SdkClient:           sdkClient,
		StartingPort:        1024,
		NumCallbackHosts:    1,
		Lambda:              1.0,
		HalfLife:            1.0,
		MaxDelay:            0,
		MaxErrorProbability: 0.0,
	}
	scenario, err := scenarios.NewCompletionCallbackScenario(opts)
	require.NoError(t, err)

	// Run an iteration.
	ctx := context.Background()
	err = scenario.RunIteration(ctx, client.StartWorkflowOptions{})
	require.NoError(t, err)

	// Get the request sent to the SDK client and verify it.
	startWorkflowOptions := <-executeWorkflowRequests
	if assert.Len(t, startWorkflowOptions.CompletionCallbacks, 1) {
		cb := startWorkflowOptions.CompletionCallbacks[0]
		nexusCb := cb.GetNexus()
		require.NotNilf(t, nexusCb, "Completion callback should be a Nexus callback")
		u, err := url.Parse(nexusCb.Url)
		require.NoError(t, err)
		assert.Equal(t, "http", u.Scheme)
		assert.Equal(t, "localhost", u.Hostname())
		assert.Equal(t, "1024", u.Port())
		assert.Empty(t, u.Path)
		q := u.Query()
		duration, err := time.ParseDuration(q.Get("delay"))
		require.NoError(t, err)
		assert.Zero(t, duration)
		errorProbability, err := strconv.ParseFloat(q.Get("failure-probability"), 32)
		require.NoError(t, err)
		assert.Zero(t, errorProbability)
	}
}
