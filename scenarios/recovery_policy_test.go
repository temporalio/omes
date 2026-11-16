package scenarios

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/journal"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestRetryProbeAttemptPolicy(t *testing.T) {
	info := loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{RetryScenarioMaxAttemptsFlag: "4", RetryScenarioFailAttemptsFlag: "2", NexusEnabledFlag: "false"})}
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	probes := retryScenarioActionsWithPolicy(time.Second, executor.config.RetryScenarioFailAttempts, executor.config.RetryScenarioMaxAttempts)
	for _, action := range probes[1:] {
		activity := action.GetExecActivity()
		require.Equal(t, int32(4), activity.GetRetryPolicy().GetMaximumAttempts())
		switch {
		case activity.GetRetryableError() != nil:
			require.Equal(t, int32(2), activity.GetRetryableError().GetFailAttempts())
		case activity.GetTimeout() != nil:
			require.Equal(t, int32(2), activity.GetTimeout().GetFailAttempts())
		case activity.GetHeartbeat() != nil:
			require.Equal(t, int32(2), activity.GetHeartbeat().GetFailAttempts())
		}
	}
	for _, values := range []map[string]string{{RetryScenarioMaxAttemptsFlag: "2", RetryScenarioFailAttemptsFlag: "2"}, {RetryScenarioFailAttemptsFlag: "0"}, {RetryScenarioMaxAttemptsFlag: "1001"}} {
		info.Options = loadgen.MustResolveScenarioOptions("throughput_stress", values)
		require.Error(t, newThroughputStressExecutor().Configure(info))
	}
}
func TestScheduleRecoveryRestoresBudgetAndWitness(t *testing.T) {
	backend := &scenarioJournal{}
	session, err := journal.Claim(t.Context(), backend, "lane", 1)
	require.NoError(t, err)
	require.NoError(t, session.Configure(t.Context(), []byte("contract")))
	require.NoError(t, session.Begin(t.Context(), 1))
	policy := scheduleRecoveryPolicy{Calls: 7, Timeout: 20 * time.Second, Backoff: time.Millisecond}
	recovery := &scheduleRecovery{journal: session, journalPrefix: "resource", logger: zap.NewNop().Sugar(), policy: &policy, options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(time.Minute)}}}
	first, cancel := recovery.operation(t.Context(), "update")
	require.NoError(t, first.beginErr)
	require.NoError(t, first.call(func(_ context.Context) error { return nil }))
	originalDeadline, _ := first.ctx.Deadline()
	cancel()
	replacement, err := journal.Claim(t.Context(), backend, "lane", 2)
	require.NoError(t, err)
	changed := scheduleRecoveryPolicy{Calls: 100, Timeout: time.Minute, Backoff: time.Second}
	recovery.journal = replacement
	recovery.policy = &changed
	recovery.ordinals = nil
	restored, stop := recovery.operation(t.Context(), "update")
	defer stop()
	require.NoError(t, restored.beginErr)
	require.Equal(t, policy, restored.recoveryPolicy())
	require.Equal(t, first.witness, restored.witness)
	require.Equal(t, 1, restored.calls)
	deadline, _ := restored.ctx.Deadline()
	require.True(t, originalDeadline.Equal(deadline))
}

func TestInterruptedScheduleOperationRemainsPending(t *testing.T) {
	backend := &scenarioJournal{}
	session, err := journal.Claim(t.Context(), backend, "lane", 1)
	require.NoError(t, err)
	require.NoError(t, session.Begin(t.Context(), 1))
	parent, cancelParent := context.WithCancel(t.Context())
	defer cancelParent()
	recovery := &scheduleRecovery{journal: session, journalPrefix: "resource", logger: zap.NewNop().Sugar(), options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(time.Minute)}}}
	operation, cancel := recovery.operation(parent, "update")
	defer cancel()
	require.NoError(t, operation.beginErr)
	err = operation.call(func(context.Context) error { cancelParent(); return context.Canceled })
	require.ErrorIs(t, err, context.Canceled)
	require.ErrorIs(t, operation.persistResult(err), context.Canceled)
	item := backend.states["lane"].Pending.Items["resource/update/0"]
	require.Equal(t, uint32(1), item.Calls)
	require.False(t, item.Done)
	require.False(t, item.Unresolved)
}
