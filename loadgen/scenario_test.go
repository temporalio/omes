package loadgen

import (
	"context"
	"math"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/operatorservicemock/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	sdkmocks "go.temporal.io/sdk/mocks"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestRegisterDefaultSearchAttributes(t *testing.T) {
	unexpected := serviceerror.NewUnavailable("visibility storage unavailable")
	for _, tc := range []struct {
		name    string
		err     error
		wantErr bool
	}{
		{name: "success"},
		{name: "typed duplicate", err: serviceerror.NewAlreadyExists("duplicate")},
		{name: "wrapped typed duplicate", err: fmt.Errorf("operator: %w", serviceerror.NewAlreadyExists("duplicate"))},
		{name: "grpc duplicate", err: status.Error(codes.AlreadyExists, "duplicate")},
		{name: "legacy allocation race", err: serviceerror.NewUnavailable("Generated search attributes mapping unavailble."), wantErr: true},
		{name: "unrelated unavailable", err: unexpected, wantErr: true},
		{name: "permission denied", err: serviceerror.NewPermissionDenied("registration forbidden", ""), wantErr: true},
		{name: "untyped duplicate text", err: errors.New("already exists"), wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			operatorClient := operatorservicemock.NewMockOperatorServiceClient(gomock.NewController(t))
			operatorClient.EXPECT().AddSearchAttributes(gomock.Any(), gomock.Any()).Return(nil, tc.err)
			temporalClient := &sdkmocks.Client{}
			temporalClient.On("OperatorService").Return(operatorClient)
			info := ScenarioInfo{Client: temporalClient, Namespace: "test-namespace"}
			err := info.RegisterDefaultSearchAttributes(t.Context())
			if tc.wantErr {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestExecuteKitchenSinkWorkflowPreservesClientActionError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cause := errors.New("describe failed")
		cleanupErr := errors.New("terminate failed")
		core, logs := observer.New(zap.DebugLevel)
		workflowRun := &sdkmocks.WorkflowRun{}
		workflowRun.On("GetRunID").Return("test-run")
		workflowRun.On("Get", mock.Anything, mock.Anything).Return(func(ctx context.Context, _ any) error { <-ctx.Done(); return ctx.Err() })
		temporalClient := &sdkmocks.Client{}
		temporalClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, "kitchenSink", mock.Anything).Return(workflowRun, nil)
		temporalClient.On("DescribeWorkflowExecution", mock.Anything, "test-workflow", "").Return(nil, cause)
		temporalClient.On("TerminateWorkflow", mock.Anything, "test-workflow", "", "client actions failed", nil).Return(cleanupErr)
		info := ScenarioInfo{Client: temporalClient, Logger: zap.New(core).Sugar()}
		run := Run{ScenarioInfo: &info, Logger: info.Logger}
		action := &kitchensink.ClientAction{
			Variant: &kitchensink.ClientAction_DoDescribe{DoDescribe: &kitchensink.DoDescribe{}},
		}
		err := run.ExecuteKitchenSinkWorkflow(t.Context(), &KitchenSinkWorkflowOptions{
			StartOptions: client.StartWorkflowOptions{ID: "test-workflow"},
			Params: &kitchensink.TestInput{
				ClientSequence: &kitchensink.ClientSequence{
					ActionSets: []*kitchensink.ClientActionSet{{Actions: []*kitchensink.ClientAction{action}}},
				},
			},
		})
		require.ErrorIs(t, err, cause)
		require.NotErrorIs(t, err, context.Canceled)
		synctest.Wait()
		require.Equal(t, 1, logs.FilterMessageSnippet(cleanupErr.Error()).Len())
		require.Zero(t, logs.FilterMessageSnippet(cause.Error()).Len())
		temporalClient.AssertExpectations(t)
	})
}
