package scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestScheduleRecoveryTransientCategory(t *testing.T) {
	for _, tc := range []struct {
		name  string
		err   error
		retry bool
	}{
		{"typed_deadline", serviceerror.NewDeadlineExceeded("persistence timeout"), true},
		{"wrapped_typed_deadline", fmt.Errorf("read: %w", serviceerror.NewDeadlineExceeded("timeout")), true},
		{"unavailable", serviceerror.NewUnavailable("unavailable"), true},
		{"resource_exhausted", serviceerror.NewResourceExhausted(0, "overload"), true},
		{"raw_grpc_aborted", status.Error(codes.Aborted, "conflict"), true},
		{"raw_context_deadline", context.DeadlineExceeded, false},
		{"canceled", context.Canceled, false},
		{"permission", serviceerror.NewPermissionDenied("denied", ""), false},
		{"invalid", serviceerror.NewInvalidArgument("invalid"), false},
		{"not_found", serviceerror.NewNotFound("absent"), false},
		{"internal", serviceerror.NewInternal("invariant"), false},
		{"budget", errors.Join(serviceerror.NewDeadlineExceeded("timeout"), errScheduleRecoveryBudget), false},
	} {
		t.Run(tc.name, func(t *testing.T) { require.Equal(t, tc.retry, transientScheduleError(tc.err)) })
	}
}

func TestScheduleRecoveryBudget(t *testing.T) {
	t.Run("logical_calls_and_backoff", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := &scheduleRecovery{options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(time.Minute)}}, logger: zap.NewNop().Sugar()}
			b, cancel := r.operation(t.Context(), "describe")
			defer cancel()
			start := time.Now()
			calls := 0
			for range 5 {
				err := b.call(func(context.Context) error { calls++; return serviceerror.NewDeadlineExceeded("timeout") })
				require.Error(t, err)
			}
			require.Equal(t, 4, calls)
			require.Equal(t, 4, b.calls)
			require.Equal(t, 300*time.Millisecond, time.Since(start))
			require.ErrorIs(t, b.call(func(context.Context) error { t.Fatal("extra call"); return nil }), errScheduleRecoveryBudget)
		})
	})
	t.Run("one_wallclock_budget_including_calls", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := &scheduleRecovery{options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(time.Minute)}}, logger: zap.NewNop().Sugar()}
			b, cancel := r.operation(t.Context(), "describe")
			defer cancel()
			start := time.Now()
			require.Error(t, b.call(func(context.Context) error {
				time.Sleep(4 * time.Second)
				return serviceerror.NewDeadlineExceeded("timeout")
			}))
			require.ErrorIs(t, b.call(func(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }), context.DeadlineExceeded)
			require.Equal(t, 5*time.Second, time.Since(start))
			require.Equal(t, 2, b.calls)
		})
	})
	t.Run("cancellation_during_backoff", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			r := &scheduleRecovery{options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(time.Minute)}}, logger: zap.NewNop().Sugar()}
			b, stop := r.operation(ctx, "describe")
			defer stop()
			require.Error(t, b.call(func(context.Context) error { return serviceerror.NewDeadlineExceeded("timeout") }))
			go func() { time.Sleep(50 * time.Millisecond); cancel() }()
			require.ErrorIs(t, b.call(func(context.Context) error { t.Fatal("call after cancellation"); return nil }), context.Canceled)
			require.Equal(t, 1, b.calls)
		})
	})
	t.Run("expired_endat_not_success_or_renewal", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			r := &scheduleRecovery{options: client.ScheduleOptions{Spec: client.ScheduleSpec{EndAt: time.Now().Add(-time.Second)}}, logger: zap.NewNop().Sugar()}
			for _, op := range []string{"create", "describe", "update"} {
				b, cancel := r.operation(t.Context(), op)
				require.ErrorIs(t, b.call(func(context.Context) error { t.Fatal("expired action window"); return nil }), context.DeadlineExceeded)
				cancel()
			}
			b, cancel := r.operation(t.Context(), "delete")
			defer cancel()
			require.NoError(t, b.call(func(context.Context) error { return nil }))
		})
	})
}
