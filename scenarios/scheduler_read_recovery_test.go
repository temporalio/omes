package scenarios

import (
	"context"
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type schedulerReadClient struct {
	client.Client
	api *schedulerReadAPI
}

func (c *schedulerReadClient) ScheduleClient() client.ScheduleClient { return c.api }

type schedulerReadAPI struct {
	client.ScheduleClient
	mu                                               sync.Mutex
	schedules                                        map[string]client.ScheduleOptions
	created                                          []client.ScheduleOptions
	reads, updates, deletes                          []string
	readFailures                                     int
	readError, createError, updateError, deleteError error
	readDelay                                        time.Duration
	cancelOnRead                                     context.CancelFunc
}

func (a *schedulerReadAPI) Create(ctx context.Context, o client.ScheduleOptions) (client.ScheduleHandle, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	a.created = append(a.created, o)
	a.schedules[o.ID] = o
	return &schedulerReadHandle{api: a, id: o.ID}, a.createError
}
func (a *schedulerReadAPI) GetHandle(_ context.Context, id string) client.ScheduleHandle {
	return &schedulerReadHandle{api: a, id: id}
}

type schedulerReadHandle struct {
	client.ScheduleHandle
	api *schedulerReadAPI
	id  string
}

func (h *schedulerReadHandle) Describe(ctx context.Context) (*client.ScheduleDescription, error) {
	a := h.api
	a.mu.Lock()
	defer a.mu.Unlock()
	a.reads = append(a.reads, h.id)
	if a.cancelOnRead != nil {
		a.cancelOnRead()
	}
	if a.readDelay > 0 {
		select {
		case <-time.After(a.readDelay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if a.readFailures > 0 {
		a.readFailures--
		return nil, a.readError
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	o, ok := a.schedules[h.id]
	if !ok {
		return nil, serviceerror.NewNotFound("missing schedule")
	}
	return &client.ScheduleDescription{Schedule: client.Schedule{Spec: &o.Spec, Action: o.Action}}, nil
}
func (h *schedulerReadHandle) Update(ctx context.Context, o client.ScheduleUpdateOptions) error {
	a := h.api
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	a.updates = append(a.updates, h.id)
	current := a.schedules[h.id]
	update, err := o.DoUpdate(client.ScheduleUpdateInput{Description: client.ScheduleDescription{Schedule: client.Schedule{Spec: &current.Spec, Action: current.Action}}})
	if err != nil {
		return err
	}
	current.Action = update.Schedule.Action
	a.schedules[h.id] = current
	return a.updateError
}
func (h *schedulerReadHandle) Delete(ctx context.Context) error {
	a := h.api
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	a.deletes = append(a.deletes, h.id)
	delete(a.schedules, h.id)
	return a.deleteError
}

func schedulerReadInfo(api *schedulerReadAPI, enabled bool) loadgen.ScenarioInfo {
	options := loadgen.MustResolveScenarioOptions("scheduler_stress", map[string]string{
		"schedule-creation-per-iteration": "1", "schedule-reads-per-creation": "1", "schedule-updates-per-creation": "1",
		"scheduler-duration-per-iteration": "1s", "wait-time-before-cleanup": "0s", "operation-interval": "1ms",
	})
	// Keep this fixture independent of scenario option registration.
	if options.Lookup("retry-schedule-reads") == nil {
		options.Bool("retry-schedule-reads", false, "")
	}
	if enabled {
		if err := options.Set("retry-schedule-reads", "true"); err != nil {
			panic(err)
		}
	}
	return loadgen.ScenarioInfo{RunID: "read-recovery", Client: &schedulerReadClient{api: api}, Logger: zap.NewNop().Sugar(), Options: options}
}

func TestSchedulerReadRecovery(t *testing.T) {
	cases := []struct {
		name      string
		enabled   bool
		failures  int
		err       error
		wantReads int
		wantErr   bool
	}{
		{"healthy", true, 0, nil, 1, false},
		{"timeout_then_success", true, 1, serviceerror.NewDeadlineExceeded("server persistence timeout"), 2, false},
		{"raw_context_deadline", true, 1, context.DeadlineExceeded, 1, true},
		{"raw_context_cancellation", true, 1, context.Canceled, 1, true},
		{"disabled", false, 1, serviceerror.NewDeadlineExceeded("server persistence timeout"), 1, true},
		{"exhausted", true, 2, serviceerror.NewDeadlineExceeded("server persistence timeout"), 2, true},
		{"permission", true, 1, serviceerror.NewPermissionDenied("denied", ""), 1, true},
		{"invalid", true, 1, serviceerror.NewInvalidArgument("invalid"), 1, true},
		{"returned_unavailable", true, 1, serviceerror.NewUnavailable("returned Unavailable"), 1, true},
		{"not_found_compatibility", true, 1, serviceerror.NewNotFound("missing"), 1, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				api := &schedulerReadAPI{schedules: map[string]client.ScheduleOptions{}, readFailures: tc.failures, readError: tc.err}
				info := schedulerReadInfo(api, tc.enabled)
				err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
				if tc.wantErr {
					require.Error(t, err)
					require.ErrorIs(t, err, tc.err)
				} else {
					require.NoError(t, err)
				}
				require.Len(t, api.created, 1)
				require.Len(t, api.reads, tc.wantReads)
				require.Equal(t, []string{api.created[0].ID}, api.updates)
				require.Equal(t, api.updates, api.deletes)
				for _, id := range api.reads {
					require.Equal(t, api.created[0].ID, id)
				}
				require.Empty(t, api.schedules)
			})
		})
	}
}

func TestSchedulerReadRecoveryStops(t *testing.T) {
	for _, tc := range []string{"parent_canceled", "during_backoff", "window_expired", "window_too_short", "retry_deadline"} {
		t.Run(tc, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				api := &schedulerReadAPI{schedules: map[string]client.ScheduleOptions{}, readFailures: 1, readError: serviceerror.NewDeadlineExceeded("timeout")}
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				switch tc {
				case "parent_canceled":
					api.cancelOnRead = cancel
				case "during_backoff":
					api.cancelOnRead = func() { go func() { time.Sleep(50 * time.Millisecond); cancel() }() }
				case "window_expired":
					api.readDelay = 4 * time.Second
				case "window_too_short":
					api.readDelay = 3 * time.Second
				case "retry_deadline":
					api.readDelay = 2 * time.Second
				}
				info := schedulerReadInfo(api, true)
				err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
				require.Error(t, err)
				wantReads := 1
				if tc == "retry_deadline" {
					wantReads = 2
					require.ErrorIs(t, err, context.DeadlineExceeded)
				}
				require.Len(t, api.reads, wantReads)
				require.Len(t, api.created, 1)
				if tc == "parent_canceled" || tc == "during_backoff" {
					require.ErrorIs(t, err, context.Canceled)
					require.Len(t, api.schedules, 1)
				} else {
					require.False(t, api.created[0].Spec.EndAt.After(time.Now()))
					require.Empty(t, api.schedules)
				}
			})
		})
	}
}

func TestSchedulerReadRecoveryDoesNotReplayWrites(t *testing.T) {
	for _, operation := range []string{"create", "update", "delete"} {
		t.Run(operation, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				lost := serviceerror.NewDeadlineExceeded("write committed, response lost")
				api := &schedulerReadAPI{schedules: map[string]client.ScheduleOptions{}}
				switch operation {
				case "create":
					api.createError = lost
				case "update":
					api.updateError = lost
				case "delete":
					api.deleteError = lost
				}
				info := schedulerReadInfo(api, true)
				err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
				require.Error(t, err)
				require.True(t, errors.Is(err, lost))
				require.Len(t, api.created, 1)
				if operation == "create" {
					require.Len(t, api.schedules, 1)
					require.Empty(t, api.deletes)
				} else {
					require.Len(t, api.updates, 1)
					require.Len(t, api.deletes, 1)
					require.Empty(t, api.schedules)
				}
			})
		})
	}
}

func TestSchedulerReadRecoveryOption(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		values := map[string]string{}
		if enabled {
			values["retry-schedule-reads"] = "true"
		}
		options := loadgen.MustResolveScenarioOptions("scheduler_stress", values)
		actual, err := options.GetBool("retry-schedule-reads")
		require.NoError(t, err)
		require.Equal(t, enabled, actual)
	}
}
