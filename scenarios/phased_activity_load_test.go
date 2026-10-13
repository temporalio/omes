package scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/phaseplan"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type standaloneTestClient struct {
	client.Client
	startErr    error
	countResult *client.CountActivitiesResult
}

func (c standaloneTestClient) ExecuteActivity(
	context.Context,
	client.StartActivityOptions,
	any,
	...any,
) (client.ActivityHandle, error) {
	return nil, c.startErr
}

func (c standaloneTestClient) CountActivities(
	context.Context,
	client.CountActivitiesOptions,
) (*client.CountActivitiesResult, error) {
	return c.countResult, nil
}

func TestPhasedActivityLoad(t *testing.T) {
	t.Parallel()

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(2*time.Minute),
		workertest.WithDynamicConfig(map[string]any{"activity.enableStandalone": true}))
	planJSON := `{"phases":[` +
		`{"name":"load","duration":"2s","addRate":2,"workerProfile":"test"},` +
		`{"name":"low-add-drain","duration":"2s","addRate":1,"workerProfile":"test"},` +
		`{"name":"zero-add-drain","duration":"1s","addRate":0,"workerProfile":"test"}]}`

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("phased-activity-load-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			StartAt: time.Now().Add(10 * time.Second),
			Timeout: 30 * time.Second,
		},
		Options: loadgen.MustResolveScenarioOptions("phased_activity_load", map[string]string{
			PhasePlanJSONFlag:                               planJSON,
			MinAchievedRateRatioFlag:                        "0",
			PhasedActivityVisibilityVerificationTimeoutFlag: "5s",
		}),
	}

	executor := newPhasedActivityLoadExecutor()
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Phased activity load executor should complete successfully")
}

func TestValidateAchievedRate(t *testing.T) {
	t.Parallel()

	phase := phaseplan.Phase{Name: "load", Duration: 10 * time.Second, AddRate: 1000}
	require.NoError(t, validateAchievedRate(phase, phaseRunStats{
		SuccessfulStarts: 9500,
		Elapsed:          10 * time.Second,
	}, 0.90, 10*time.Second))

	err := validateAchievedRate(phase, phaseRunStats{
		SuccessfulStarts: 8000,
		Elapsed:          10 * time.Second,
	}, 0.90, 10*time.Second)
	require.ErrorContains(t, err, "under-driven")

	phase.Duration = 9 * time.Second
	require.NoError(t, validateAchievedRate(phase, phaseRunStats{
		SuccessfulStarts: 1,
		Elapsed:          9 * time.Second,
	}, 0.90, 10*time.Second))
}

func TestPhasedActivityLoadRequiresStartAt(t *testing.T) {
	t.Parallel()

	scenarioInfo := loadgen.ScenarioInfo{
		Options: loadgen.MustResolveScenarioOptions("phased_activity_load", map[string]string{
			PhasePlanJSONFlag: `{"phases":[{"name":"load","duration":"1s","addRate":1,"workerProfile":null}]}`,
		}),
	}
	require.ErrorContains(t, newPhasedActivityLoadExecutor().Configure(scenarioInfo), "--start-at")
}

func TestStandaloneActivityHelpers(t *testing.T) {
	t.Parallel()

	require.Equal(t, 100*time.Microsecond, intervalForRate(10000))
	require.Equal(t, "phased-activity-load-exec-activity-000000000042", standaloneActivityID("exec", 42))
	require.Equal(t, int64(3), terminalActivityFailures(map[string]int64{
		"Completed":  10,
		"Failed":     1,
		"TimedOut":   1,
		"Canceled":   1,
		"Terminated": 0,
	}))
}

func TestPhasedActivityLoadReportsUnsupportedStandaloneActivities(t *testing.T) {
	t.Parallel()
	planJSON := `{"phases":[{"name":"load","duration":"1s","addRate":100,"workerProfile":null}]}`

	scenarioInfo := loadgen.ScenarioInfo{
		RunID:       "unsupported-standalone",
		ExecutionID: "test-execution",
		Logger:      zap.NewNop().Sugar(),
		Client: standaloneTestClient{
			startErr: errors.New("standalone activities disabled"),
		},
		Configuration: loadgen.RunConfiguration{
			StartAt:                       time.Now().Add(time.Second),
			DoNotRegisterSearchAttributes: true,
		},
		Options: loadgen.MustResolveScenarioOptions("phased_activity_load", map[string]string{
			PhasePlanJSONFlag:                      planJSON,
			PhasedActivityMaxConsecutiveErrorsFlag: "1",
		}),
	}

	err := newPhasedActivityLoadExecutor().Run(t.Context(), scenarioInfo)
	require.ErrorContains(t, err, "activity.enableStandalone")
}

func TestVerifyStandaloneActivitiesReportsTerminalFailure(t *testing.T) {
	t.Parallel()

	executor := newPhasedActivityLoadExecutor()
	executor.ScenarioInfo = loadgen.ScenarioInfo{
		ExecutionID: "failed-execution",
		Client: standaloneTestClient{countResult: &client.CountActivitiesResult{
			Count: 1,
			Groups: []client.CountActivitiesAggregationGroup{{
				GroupValues: []any{"Failed"},
				Count:       1,
			}},
		}},
	}
	executor.config = &phasedActivityLoadConfig{VisibilityVerificationTimeout: time.Second}

	err := executor.verifyStandaloneActivities(t.Context(), 1)
	require.ErrorContains(t, err, "terminal failures")
}
