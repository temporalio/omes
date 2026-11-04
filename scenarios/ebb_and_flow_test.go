package scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/operatorservicemock/v1"
	"go.temporal.io/api/serviceerror"
	sdkmocks "go.temporal.io/sdk/mocks"
	"go.uber.org/zap"
)

func TestEbbAndFlowRateMode(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(2*time.Minute))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("eaf-rate-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Duration: 10 * time.Second,
		},
		Options: loadgen.MustResolveScenarioOptions("ebb_and_flow", map[string]string{
			MinAddRateFlag:                    "1",
			MaxAddRateFlag:                    "3",
			RatePeriodFlag:                    "5s",
			MinBacklogFlag:                    "0",
			MaxBacklogFlag:                    "3",
			BacklogPeriodFlag:                 "5s",
			PeriodFlag:                        "5s",
			BacklogLogIntervalFlag:            "5s",
			VisibilityVerificationTimeoutFlag: "5s",
		}),
	}

	executor := newEbbAndFlowExecutor()
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Rate mode executor should complete successfully")

	state := executor.Snapshot().(ebbAndFlowState)
	require.GreaterOrEqual(t, state.TotalCompletedWorkflows, int64(1))
}

func TestEbbAndFlow(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(2*time.Minute))

	sleepActivityJson := `{
		"count": {
			"type": "fixed",
			"value": "5"
		},
		"groups": {
			"group_a": {
				"weight": 1,
				"sleepDuration": {
					"type": "fixed",
					"value": "1ms"
				},
				"fairnessKeys": {
					"type": "fixed",
					"value": "1"
				}
			},
			"group_b": {
				"weight": 1,
				"sleepDuration": {
					"type": "fixed",
					"value": "1ms"
				},
				"fairnessKeys": {
					"type": "fixed",
					"value": "2"
				}
			}
		}
	}`

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("eaf-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Duration: 10 * time.Second,
		},
		Options: loadgen.MustResolveScenarioOptions("ebb_and_flow", map[string]string{
			MinBacklogFlag:                    "0",
			MaxBacklogFlag:                    "1",
			PeriodFlag:                        "5s",
			BacklogLogIntervalFlag:            "5s",
			VisibilityVerificationTimeoutFlag: "5s",
			SleepActivityJsonFlag:             sleepActivityJson,
		}),
	}

	var state ebbAndFlowState

	t.Run("Run executor", func(t *testing.T) {
		executor := newEbbAndFlowExecutor()

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state = executor.Snapshot().(ebbAndFlowState)
		require.GreaterOrEqual(t, state.TotalCompletedWorkflows, int64(1))
	})

	t.Run("Run executor again, resuming from previous state", func(t *testing.T) {
		executor := newEbbAndFlowExecutor()

		// Load previous state
		previouState := state
		err := executor.LoadState(func(v any) error {
			s := v.(*ebbAndFlowState)
			*s = state
			return nil
		})
		require.NoError(t, err, "Should be able to load previous state")

		resumeScenarioInfo := scenarioInfo

		_, err = env.RunExecutorTest(t, executor, resumeScenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state = executor.Snapshot().(ebbAndFlowState)
		require.Greater(t, state.TotalCompletedWorkflows, previouState.TotalCompletedWorkflows)
	})

	t.Run("Run executor again, resuming from previous state but without any time left", func(t *testing.T) {
		executor := newEbbAndFlowExecutor()

		// Load previous state
		err := executor.LoadState(func(v any) error {
			s := v.(*ebbAndFlowState)
			*s = state
			return nil
		})
		require.NoError(t, err, "Should be able to load previous state")

		resumeScenarioInfo := scenarioInfo
		resumeScenarioInfo.Configuration.Duration = 0 // ie no more time left

		_, err = env.RunExecutorTest(t, executor, resumeScenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")
	})
}

func TestEbbAndFlowRegistration(t *testing.T) {
	for _, skip := range []bool{false, true} {
		t.Run(fmt.Sprintf("skip=%v", skip), func(t *testing.T) {
			cause := serviceerror.NewPermissionDenied("registration forbidden", "")
			temporalClient := &sdkmocks.Client{}
			if !skip {
				operatorClient := operatorservicemock.NewMockOperatorServiceClient(gomock.NewController(t))
				operatorClient.EXPECT().AddSearchAttributes(gomock.Any(), gomock.Any()).Return(nil, cause)
				temporalClient.On("OperatorService").Return(operatorClient)
			}
			info := loadgen.ScenarioInfo{
				Client:        temporalClient,
				Logger:        zap.NewNop().Sugar(),
				Configuration: loadgen.RunConfiguration{DoNotRegisterSearchAttributes: skip},
				Options:       loadgen.MustResolveScenarioOptions("ebb_and_flow", nil),
			}
			err := newEbbAndFlowExecutor().Run(t.Context(), info)
			if skip {
				require.ErrorContains(t, err, "No iterations completed")
			} else {
				require.ErrorIs(t, err, cause)
			}
		})
	}
}

func TestEbbAndFlowWorkflowAccounting(t *testing.T) {
	cause := errors.New("workflow failed")
	for _, tc := range []struct {
		name      string
		err       error
		completed int64
	}{{name: "success", completed: 1}, {name: "failure", err: cause}} {
		t.Run(tc.name, func(t *testing.T) {
			workflowRun := &sdkmocks.WorkflowRun{}
			workflowRun.On("Get", mock.Anything, mock.Anything).Return(tc.err)
			temporalClient := &sdkmocks.Client{}
			temporalClient.On("ExecuteWorkflow", mock.Anything, mock.Anything, "ebbAndFlowTrack", mock.Anything).Return(workflowRun, nil)
			executor := newEbbAndFlowExecutor()
			executor.ScenarioInfo = loadgen.ScenarioInfo{Client: temporalClient, Logger: zap.NewNop().Sugar()}
			err := executor.spawnWorkflowWithActivities(context.Background(), 1, 3, &loadgen.SleepActivityConfig{})
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.completed, executor.Snapshot().(ebbAndFlowState).TotalCompletedWorkflows)
			require.Equal(t, int64(3), executor.scheduledActivities.Load())
			require.Equal(t, int64(3), executor.completedActivities.Load())
		})
	}
}
