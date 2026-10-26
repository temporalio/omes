package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.uber.org/zap"
)

func healthyHeartbeatActions(t *testing.T, options map[string]string) []*ks.ExecuteActivityAction {
	t.Helper()
	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:       "tps-healthy-heartbeat",
		ExecutionID: "exec",
		Logger:      zap.NewNop().Sugar(),
		Options:     loadgen.MustResolveScenarioOptions("throughput_stress", options),
	}
	require.NoError(t, executor.Configure(info))

	var activities []*ks.ExecuteActivityAction
	walkActions(executor.createActions(info.NewRun(1))[0].GetActions(), func(action *ks.Action) {
		activity := action.GetExecActivity()
		if heartbeat := activity.GetHeartbeat(); heartbeat != nil && heartbeat.GetFailAttempts() == 0 {
			activities = append(activities, activity)
		}
	})
	return activities
}

func TestThroughputStressHealthyHeartbeatActivities(t *testing.T) {
	t.Parallel()

	t.Run("off by default", func(t *testing.T) {
		require.Empty(t, healthyHeartbeatActions(t, map[string]string{NexusEnabledFlag: "false"}))
	})

	t.Run("configured per internal iteration", func(t *testing.T) {
		activities := healthyHeartbeatActions(t, map[string]string{
			NexusEnabledFlag:               "false",
			IterFlag:                       "3",
			HealthyHeartbeatActivitiesFlag: "2",
			HealthyHeartbeatDurationFlag:   "20s",
			HealthyHeartbeatTimeoutFlag:    "8s",
		})
		require.Len(t, activities, 6)
		for _, activity := range activities {
			require.Equal(t, 20*time.Second, activity.GetHeartbeat().GetSuccessDuration().AsDuration())
			require.Equal(t, 2*time.Second, activity.GetHeartbeat().GetHeartbeatInterval().AsDuration())
			require.Equal(t, 8*time.Second, activity.GetHeartbeatTimeout().AsDuration())
			require.Equal(t, 28*time.Second, activity.GetStartToCloseTimeout().AsDuration())
			require.Equal(t, int32(1), activity.GetRetryPolicy().GetMaximumAttempts())
			require.NotNil(t, activity.GetRemote())
		}
	})
}

func TestThroughputStressConfigureInvalidHealthyHeartbeat(t *testing.T) {
	t.Parallel()

	for flag, value := range map[string]string{
		HealthyHeartbeatActivitiesFlag: "-1",
		HealthyHeartbeatDurationFlag:   "0s",
		HealthyHeartbeatTimeoutFlag:    "0s",
	} {
		t.Run(flag, func(t *testing.T) {
			err := newThroughputStressExecutor().Configure(loadgen.ScenarioInfo{
				RunID:   "tps-invalid-healthy-heartbeat",
				Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{flag: value}),
			})
			require.ErrorContains(t, err, flag)
		})
	}
}

// The activity outlives its heartbeat timeout and is not retried, so the run
// only succeeds if the Go worker keeps heartbeating it.
func TestThroughputStressHealthyHeartbeatKeepsActivityAlive(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-healthy-heartbeat-%d", time.Now().UnixNano())
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	_, err := env.RunExecutorTest(t, newThroughputStressExecutor(), loadgen.ScenarioInfo{
		RunID:         runID,
		Configuration: loadgen.RunConfiguration{Iterations: 1},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag:                          "1",
			NexusEnabledFlag:                  "false",
			SleepTimeFlag:                     "1ms",
			VisibilityVerificationTimeoutFlag: "10s",
			HealthyHeartbeatActivitiesFlag:    "1",
			HealthyHeartbeatDurationFlag:      "6s",
			HealthyHeartbeatTimeoutFlag:       "3s",
		}),
	}, clioptions.LangGo)
	require.NoError(t, err)
}

func TestThroughputStressHealthyHeartbeatDurationOverflow(t *testing.T) {
	for _, tc := range []struct {
		name     string
		duration time.Duration
		invalid  bool
	}{
		{name: "largest representable sum", duration: time.Duration(1<<63-1) - time.Second},
		{name: "sum overflows", duration: time.Duration(1<<63 - 1), invalid: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
				NexusEnabledFlag:             "false",
				HealthyHeartbeatDurationFlag: tc.duration.String(),
				HealthyHeartbeatTimeoutFlag:  time.Second.String(),
			})}
			err := newThroughputStressExecutor().Configure(info)
			if tc.invalid {
				require.ErrorContains(t, err, HealthyHeartbeatDurationFlag)
			} else {
				require.NoError(t, err)
				require.Equal(t, time.Duration(1<<63-1), healthyHeartbeatActivity(tc.duration, time.Second).GetExecActivity().GetStartToCloseTimeout().AsDuration())
			}
		})
	}
}
