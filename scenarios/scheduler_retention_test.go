package scenarios

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/serviceerror"
	"google.golang.org/grpc/codes"
)

func TestSchedulerRetainSchedules(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		retain, recover, rejectRead bool
	}{
		{name: "default deletes"},
		{name: "retain finite schedules", retain: true},
		{name: "retain with recovery", retain: true, recover: true},
		{name: "retention preserves read failure", retain: true, rejectRead: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, api, _, logger := scheduleRecoveryRPCFixture(t)
			values := map[string]string{
				ScheduleCreationPerIterationFlag: "1", ScheduleReadsPerCreationFlag: "1", ScheduleUpdatesPerCreationFlag: "1",
				SchedulerDurationPerIterationFlag: "1s", WaitTimeBeforeCleanupFlag: "0s", OperationIntervalFlag: "1ms",
			}
			if tc.retain {
				values[RetainSchedulesFlag] = "true"
			}
			if tc.recover {
				values[RecoverScheduleOperationsFlag] = "true"
			}
			if tc.rejectRead {
				api.faults["describe"] = map[int]scheduleFault{1: {code: codes.PermissionDenied}}
			}
			info := loadgen.ScenarioInfo{RunID: "migration-retention", Client: c, Logger: logger, Options: loadgen.MustResolveScenarioOptions("scheduler_stress", values)}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			start := time.Now()
			err := (&SchedulerExecutor{}).Execute(ctx, info.NewRun(0))
			if tc.rejectRead {
				var denied *serviceerror.PermissionDenied
				require.ErrorAs(t, err, &denied)
			} else {
				require.NoError(t, err)
			}
			api.mu.Lock()
			defer api.mu.Unlock()
			require.Len(t, api.creates, 1)
			require.Len(t, api.updates, 1)
			if tc.retain {
				require.Empty(t, api.deletes)
				require.Len(t, api.records, 1)
				for _, record := range api.records {
					end := record.schedule.Spec.EndTime.AsTime()
					require.True(t, end.After(start))
					require.True(t, end.Before(start.Add(4*time.Second)))
				}
			} else {
				require.Len(t, api.deletes, 1)
				require.Empty(t, api.records)
			}
		})
	}
}
