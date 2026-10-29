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
	enumspb "go.temporal.io/api/enums/v1"
	"go.uber.org/zap"
)

func TestThroughputRetryScenarioTiming(t *testing.T) {
	for _, timeout := range []time.Duration{time.Second, 5 * time.Second, 10 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			options := loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
				NexusEnabledFlag:          "false",
				IterFlag:                  "1",
				IncludeRetryScenariosFlag: "true",
			})
			require.NoError(t, options.Set("retry-scenario-timeout", timeout.String()))
			info := loadgen.ScenarioInfo{RunID: "retry-timing", ExecutionID: "exec", Options: options, Logger: zap.NewNop().Sugar()}
			executor := newThroughputStressExecutor()
			require.NoError(t, executor.Configure(info))
			var probes []*ks.ExecuteActivityAction
			walkActions(executor.createActions(info.NewRun(1))[0].GetActions(), func(action *ks.Action) {
				activity := action.GetExecActivity()
				if activity.GetRetryableError() != nil || activity.GetTimeout() != nil || activity.GetHeartbeat() != nil {
					probes = append(probes, activity)
				}
			})
			require.Len(t, probes, 3)
			for _, probe := range probes {
				require.Equal(t, int32(2), probe.GetRetryPolicy().GetMaximumAttempts())
				require.Equal(t, 500*time.Millisecond, probe.GetRetryPolicy().GetInitialInterval().AsDuration())
				switch {
				case probe.GetRetryableError() != nil:
					require.Equal(t, int32(1), probe.GetRetryableError().GetFailAttempts())
					require.Equal(t, timeout, probe.GetStartToCloseTimeout().AsDuration())
				case probe.GetTimeout() != nil:
					require.Equal(t, int32(1), probe.GetTimeout().GetFailAttempts())
					require.Equal(t, timeout, probe.GetStartToCloseTimeout().AsDuration())
					require.Equal(t, 2*timeout, probe.GetTimeout().GetFailureDuration().AsDuration())
					require.Zero(t, probe.GetTimeout().GetSuccessDuration().AsDuration())
				case probe.GetHeartbeat() != nil:
					require.Equal(t, int32(1), probe.GetHeartbeat().GetFailAttempts())
					require.Equal(t, timeout, probe.GetHeartbeatTimeout().AsDuration())
					require.Equal(t, max(10*time.Second, 2*timeout), probe.GetStartToCloseTimeout().AsDuration())
					require.Zero(t, probe.GetHeartbeat().GetSuccessDuration().AsDuration())
				}
			}
		})
	}
}

func TestThroughputRetryScenarioTimingExecutesExpectedRetries(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(time.Minute))
	info := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("retry-timing-%d", time.Now().UnixNano()), ExecutionID: "timing-exec",
		Logger:        zap.NewNop().Sugar(),
		Configuration: loadgen.RunConfiguration{Iterations: 1},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			NexusEnabledFlag: "false", IterFlag: "1", SleepTimeFlag: "1ms",
			IncludeRetryScenariosFlag: "true", RetryScenarioTimeoutFlag: "5s",
			VisibilityVerificationTimeoutFlag: "10s",
		}),
	}
	_, err := env.RunExecutorTest(t, newThroughputStressExecutor(), info, clioptions.LangGo)
	require.NoError(t, err)

	types := map[int64]string{}
	completed := map[int64]bool{}
	attempts := map[int64]int32{}
	history := env.TemporalClient().GetWorkflowHistory(t.Context(), info.NewRun(1).DefaultStartWorkflowOptions().ID, "", false, enumspb.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT)
	for history.HasNext() {
		event, err := history.Next()
		require.NoError(t, err)
		if scheduled := event.GetActivityTaskScheduledEventAttributes(); scheduled != nil {
			types[event.GetEventId()] = scheduled.GetActivityType().GetName()
		}
		if started := event.GetActivityTaskStartedEventAttributes(); started != nil {
			attempts[started.GetScheduledEventId()] = started.GetAttempt()
		}
		if result := event.GetActivityTaskCompletedEventAttributes(); result != nil {
			completed[result.GetScheduledEventId()] = true
		}
	}
	for _, name := range []string{"retryable_error", "timeout", "heartbeat"} {
		found := false
		for id, activityType := range types {
			if activityType == name {
				require.True(t, completed[id], name)
				require.Equal(t, int32(2), attempts[id], name)
				found = true
			}
		}
		require.True(t, found, name)
	}
}

func TestThroughputRetryScenarioTimeoutValidation(t *testing.T) {
	for _, value := range []string{"0s", "-1s", time.Duration(1<<63 - 1).String()} {
		t.Run(value, func(t *testing.T) {
			info := loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
				"retry-scenario-timeout": value,
			})}
			require.ErrorContains(t, newThroughputStressExecutor().Configure(info), "retry-scenario-timeout")
		})
	}
}
