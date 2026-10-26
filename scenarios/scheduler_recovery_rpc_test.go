package scenarios

import (
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	commonpb "go.temporal.io/api/common/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type scheduleFault struct {
	code          codes.Code
	apply         bool
	delayedEffect time.Duration
	foreign       bool
	cancel        context.CancelFunc
	block         bool
	vanish        bool
}

type scheduleRecord struct {
	schedule *schedulepb.Schedule
	memo     *commonpb.Memo
}

type scheduleRecoveryService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	mu             sync.Mutex
	effects        sync.WaitGroup
	records        map[string]scheduleRecord
	faults         map[string]map[int]scheduleFault
	calls          map[string]int
	creates        []*workflowservice.CreateScheduleRequest
	updates        []*workflowservice.UpdateScheduleRequest
	deletes        []*workflowservice.DeleteScheduleRequest
	mutateDescribe func(*schedulepb.Schedule)
	mutationAt     int
}

func (s *scheduleRecoveryService) fault(op string) scheduleFault {
	s.calls[op]++
	return s.faults[op][s.calls[op]]
}
func (s *scheduleRecoveryService) GetSystemInfo(context.Context, *workflowservice.GetSystemInfoRequest) (*workflowservice.GetSystemInfoResponse, error) {
	return &workflowservice.GetSystemInfoResponse{}, nil
}
func (s *scheduleRecoveryService) CreateSchedule(ctx context.Context, r *workflowservice.CreateScheduleRequest) (*workflowservice.CreateScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.creates = append(s.creates, proto.Clone(r).(*workflowservice.CreateScheduleRequest))
	f := s.fault("create")
	if f.block {
		<-ctx.Done()
		return nil, status.Error(codes.DeadlineExceeded, "slow create")
	}
	if f.cancel != nil {
		f.cancel()
		return nil, status.Error(codes.Canceled, "parent canceled")
	}
	if f.code == codes.OK || f.apply {
		if _, exists := s.records[r.ScheduleId]; exists {
			return nil, status.Error(codes.AlreadyExists, "exists")
		}
		var memo *commonpb.Memo
		if r.Memo != nil {
			memo = proto.Clone(r.Memo).(*commonpb.Memo)
		}
		if f.foreign {
			memo = &commonpb.Memo{}
		}
		s.records[r.ScheduleId] = scheduleRecord{proto.Clone(r.Schedule).(*schedulepb.Schedule), memo}
	}
	if f.code != codes.OK {
		return nil, status.Error(f.code, "scripted create response loss or rejection")
	}
	return &workflowservice.CreateScheduleResponse{}, nil
}
func (s *scheduleRecoveryService) DescribeSchedule(_ context.Context, r *workflowservice.DescribeScheduleRequest) (*workflowservice.DescribeScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.fault("describe")
	if s.mutateDescribe != nil && s.calls["describe"] == s.mutationAt {
		record := s.records[r.ScheduleId]
		s.mutateDescribe(record.schedule)
		s.records[r.ScheduleId] = record
	}
	if f.code == codes.NotFound && f.apply {
		delete(s.records, r.ScheduleId)
	}
	if f.code != codes.OK {
		return nil, status.Error(f.code, "scripted describe rejection")
	}
	record, ok := s.records[r.ScheduleId]
	if !ok {
		return nil, status.Error(codes.NotFound, "absent")
	}
	var memo *commonpb.Memo
	if record.memo != nil {
		memo = proto.Clone(record.memo).(*commonpb.Memo)
	}
	return &workflowservice.DescribeScheduleResponse{Schedule: proto.Clone(record.schedule).(*schedulepb.Schedule), Memo: memo, Info: &schedulepb.ScheduleInfo{}}, nil
}
func (s *scheduleRecoveryService) UpdateSchedule(_ context.Context, r *workflowservice.UpdateScheduleRequest) (*workflowservice.UpdateScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, proto.Clone(r).(*workflowservice.UpdateScheduleRequest))
	f := s.fault("update")
	record, ok := s.records[r.ScheduleId]
	if !ok {
		return nil, status.Error(codes.NotFound, "absent")
	}
	if f.code == codes.OK || f.apply {
		updated := proto.Clone(r.Schedule).(*schedulepb.Schedule)
		if f.delayedEffect == 0 {
			record.schedule = updated
			s.records[r.ScheduleId] = record
		} else {
			s.effects.Add(1)
			go func() {
				defer s.effects.Done()
				time.Sleep(f.delayedEffect)
				s.mu.Lock()
				defer s.mu.Unlock()
				if current, ok := s.records[r.ScheduleId]; ok {
					current.schedule = updated
					s.records[r.ScheduleId] = current
				}
			}()
		}
	}
	if f.vanish {
		delete(s.records, r.ScheduleId)
	}
	if f.cancel != nil {
		f.cancel()
		return nil, status.Error(codes.Canceled, "parent canceled after update")
	}
	if f.code != codes.OK {
		return nil, status.Error(f.code, "scripted update response loss or rejection")
	}
	return &workflowservice.UpdateScheduleResponse{}, nil
}
func (s *scheduleRecoveryService) DeleteSchedule(_ context.Context, r *workflowservice.DeleteScheduleRequest) (*workflowservice.DeleteScheduleResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deletes = append(s.deletes, proto.Clone(r).(*workflowservice.DeleteScheduleRequest))
	f := s.fault("delete")
	if _, ok := s.records[r.ScheduleId]; !ok {
		return nil, status.Error(codes.NotFound, "absent")
	}
	if f.code == codes.OK || f.apply {
		delete(s.records, r.ScheduleId)
	}
	if f.code != codes.OK {
		return nil, status.Error(f.code, "scripted delete response loss or rejection")
	}
	return &workflowservice.DeleteScheduleResponse{}, nil
}

func scheduleRecoveryRPCFixture(t *testing.T) (client.Client, *scheduleRecoveryService, *observer.ObservedLogs, *zap.SugaredLogger) {
	t.Helper()
	s := &scheduleRecoveryService{records: map[string]scheduleRecord{}, faults: map[string]map[int]scheduleFault{}, calls: map[string]int{}}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	workflowservice.RegisterWorkflowServiceServer(server, s)
	done := make(chan error, 1)
	go func() { done <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done); s.effects.Wait() })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	c, err := client.DialContext(ctx, client.Options{HostPort: listener.Addr().String(), Namespace: "scheduler-recovery"})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	core, logs := observer.New(zap.InfoLevel)
	return c, s, logs, zap.New(core).Sugar()
}

func scheduleRecoveryInfo(c client.Client, logger *zap.SugaredLogger, enabled bool) loadgen.ScenarioInfo {
	options := loadgen.MustResolveScenarioOptions("scheduler_stress", map[string]string{
		"schedule-creation-per-iteration": "1", "schedule-reads-per-creation": "1", "schedule-updates-per-creation": "1",
		"scheduler-duration-per-iteration": "1s", "wait-time-before-cleanup": "0s", "operation-interval": "1ms",
	})
	// Exercise faults independently of option registration.
	if options.Lookup("recover-schedule-operations") == nil {
		options.Bool("recover-schedule-operations", false, "")
	}
	if enabled {
		if err := options.Set("recover-schedule-operations", "true"); err != nil {
			panic(err)
		}
	}
	return loadgen.ScenarioInfo{RunID: "crud-recovery", Client: c, Logger: logger, Options: options}
}

func TestSchedulerOperationRecoverySDK(t *testing.T) {
	cases := []struct {
		name                                 string
		faults                               map[string]map[int]scheduleFault
		wantErr                              string
		creates, updates, deletes, remaining int
		outcome                              string
		sdkRetry                             bool
	}{
		{name: "healthy", creates: 1, updates: 1, deletes: 1},
		{name: "read_permanent_rejection", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.PermissionDenied}}}, wantErr: "describe rejection", creates: 1, updates: 1, deletes: 1},
		{name: "create_last_probe_not_found_no_fifth_call", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded}}}, wantErr: "budget exhausted", creates: 2},
		{name: "sdk_unavailable_retries_same_rpc_request", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.Unavailable}}}, creates: 2, updates: 1, deletes: 1, sdkRetry: true},
		{name: "sdk_accepted_create_then_exists", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.Unavailable, apply: true}}}, creates: 2, updates: 1, deletes: 1, sdkRetry: true, outcome: "observed_owned_create"},
		{name: "describe_not_found_compatible", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.NotFound, apply: true}}}, creates: 1, outcome: "not_found_compatible"},
		{name: "update_describe_not_found_compatible", faults: map[string]map[int]scheduleFault{"describe": {2: {code: codes.NotFound, apply: true}}}, creates: 1, outcome: "not_found_compatible"},
		{name: "delete_not_found_compatible", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.NotFound, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: "not_found_compatible"},
		{name: "create_accepted_response_lost", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: "observed_owned_create"},
		{name: "create_not_accepted_same_frozen_intent", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded}}}, creates: 2, updates: 1, deletes: 1, outcome: "acknowledged"},
		{name: "read_deadline", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 1, outcome: "acknowledged"},
		{name: "update_internal_describe_deadline", faults: map[string]map[int]scheduleFault{"describe": {2: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 1, outcome: "acknowledged"},
		{name: "update_accepted_response_lost", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: "observed_update_witness"},
		{name: "update_late_effect_no_replay", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true, delayedEffect: 150 * time.Millisecond}}}, creates: 1, updates: 1, deletes: 1, outcome: "observed_update_witness"},
		{name: "update_identical_old_payload_not_success", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded}}}, wantErr: "outcome unknown", creates: 1, updates: 1, deletes: 1},
		{name: "update_response_lost_then_absent_is_unknown", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true, vanish: true}}}, wantErr: "outcome unknown", creates: 1, updates: 1},
		{name: "update_permanent_rejection", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.PermissionDenied}}}, wantErr: "update response loss or rejection", creates: 1, updates: 1, deletes: 1},
		{name: "delete_accepted_response_lost", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: "observed_absence"},
		{name: "delete_not_accepted_retry_after_owned_probe", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 2, outcome: "acknowledged"},
		{name: "delete_last_response_lost_no_fifth_probe", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded, apply: true}}}, wantErr: "budget exhausted", creates: 1, updates: 1, deletes: 2},
		{name: "foreign_owner_never_adopted_or_deleted", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.AlreadyExists, apply: true, foreign: true}}}, wantErr: "ownership or intent mismatch", creates: 1, remaining: 1},
		{name: "create_probe_exhausted_namespace_cleanup_owned", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded, apply: true}}, "describe": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded}, 3: {code: codes.DeadlineExceeded}}}, wantErr: "budget exhausted", creates: 1, remaining: 1},
		{name: "permanent_create_rejection", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.InvalidArgument}}}, wantErr: "create response loss or rejection", creates: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s, logs, logger := scheduleRecoveryRPCFixture(t)
			if tc.faults != nil {
				s.faults = tc.faults
			}
			info := scheduleRecoveryInfo(c, logger, true)
			err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tc.wantErr)
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			require.Len(t, s.creates, tc.creates)
			require.Len(t, s.updates, tc.updates)
			require.Len(t, s.deletes, tc.deletes)
			require.Len(t, s.records, tc.remaining)
			if tc.creates == 2 {
				require.Equal(t, s.creates[0].ScheduleId, s.creates[1].ScheduleId)
				require.True(t, proto.Equal(s.creates[0].Schedule, s.creates[1].Schedule))
				require.True(t, proto.Equal(s.creates[0].Memo, s.creates[1].Memo))
				if tc.sdkRetry {
					require.Equal(t, s.creates[0].RequestId, s.creates[1].RequestId)
				} else {
					require.NotEqual(t, s.creates[0].RequestId, s.creates[1].RequestId)
				}
			}
			for _, u := range s.updates {
				require.Equal(t, s.creates[0].ScheduleId, u.ScheduleId)
				require.True(t, proto.Equal(s.creates[0].Schedule.Spec.EndTime, u.Schedule.Spec.EndTime))
				before, after := s.creates[0].Schedule.Action.GetStartWorkflow(), u.Schedule.Action.GetStartWorkflow()
				require.Equal(t, before.WorkflowId, after.WorkflowId)
				require.True(t, proto.Equal(before.WorkflowType, after.WorkflowType))
				require.Equal(t, before.TaskQueue.Name, after.TaskQueue.Name)
				require.True(t, proto.Equal(before.Input, after.Input))
				require.NotEmpty(t, u.Schedule.State.Notes)
			}
			if tc.outcome != "" {
				found := false
				for _, e := range logs.All() {
					require.NotEmpty(t, e.ContextMap()["schedule_id"])
					if e.ContextMap()["operation"] == "update" && len(s.updates) > 0 {
						require.Equal(t, s.updates[0].Schedule.State.Notes, e.ContextMap()["mutation_id"])
					}
					if e.ContextMap()["outcome"] == tc.outcome {
						found = true
					}
				}
				require.True(t, found, "recovery outcome must be visible at Info")
			}
		})
	}
}

func TestSchedulerOperationRecoveryDefaultUnchanged(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	s.faults = map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded, apply: true}}}
	info := scheduleRecoveryInfo(c, logger, false)
	err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
	require.ErrorContains(t, err, "scripted create")
	require.Len(t, s.creates, 1)
	require.Nil(t, s.creates[0].Memo)
	require.Zero(t, s.calls["describe"])
	require.Len(t, s.records, 1)
}

func TestSchedulerOperationRecoveryParentCancellation(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.faults = map[string]map[int]scheduleFault{"create": {1: {cancel: cancel}}}
	start := time.Now()
	info := scheduleRecoveryInfo(c, logger, true)
	err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, time.Since(start), time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.creates, 1)
	require.Empty(t, s.updates)
	require.Empty(t, s.deletes)
}

func TestSchedulerOperationRecoveryIntentMismatch(t *testing.T) {
	for _, field := range []string{"action_id", "end_at", "payload", "overlap"} {
		t.Run(field, func(t *testing.T) {
			c, s, _, logger := scheduleRecoveryRPCFixture(t)
			s.mutationAt = 1
			s.mutateDescribe = func(schedule *schedulepb.Schedule) {
				switch field {
				case "action_id":
					schedule.Action.GetStartWorkflow().WorkflowId = "another-owner"
				case "end_at":
					schedule.Spec.EndTime.Seconds++
				case "payload":
					schedule.Action.GetStartWorkflow().Input.Payloads[0].Data = []byte("changed")
				case "overlap":
					schedule.Policies.OverlapPolicy = 0
				}
			}
			info := scheduleRecoveryInfo(c, logger, true)
			err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
			require.ErrorContains(t, err, "ownership or intent mismatch")
			require.Empty(t, s.updates)
			require.Empty(t, s.deletes)
			require.Len(t, s.records, 1)
		})
	}
}

func TestSchedulerOperationRecoveryCancellationAtTick(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	info := scheduleRecoveryInfo(c, logger, true)
	require.NoError(t, info.Options.Set("operation-interval", "1h"))
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.creates, 1)
	require.Empty(t, s.deletes)
	require.Len(t, s.records, 1)
}

func TestSchedulerOperationRecoveryPreparedUpdateCancellation(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s.faults = map[string]map[int]scheduleFault{"update": {1: {apply: true, cancel: cancel}}}
	info := scheduleRecoveryInfo(c, logger, true)
	err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
	require.ErrorIs(t, err, context.Canceled)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.updates, 1)
	require.Empty(t, s.deletes)
	require.Len(t, s.records, 1)
}

func TestSchedulerOperationRecoverySlowRPCBudget(t *testing.T) {
	c, s, logs, logger := scheduleRecoveryRPCFixture(t)
	s.faults = map[string]map[int]scheduleFault{"create": {1: {block: true}, 2: {block: true}}}
	info := scheduleRecoveryInfo(c, logger, true)
	require.NoError(t, info.Options.Set("scheduler-duration-per-iteration", "30s"))
	start := time.Now()
	err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
	require.True(t, errors.Is(err, errScheduleRecoveryBudget) || errors.Is(err, context.DeadlineExceeded), "unexpected recovery result: %v", err)
	require.Less(t, time.Since(start), 2*scheduleOperationTimeout)
	entries := logs.FilterMessage("Schedule operation recovery").All()
	require.Len(t, entries, 1)
	require.LessOrEqual(t, entries[0].ContextMap()["logical_calls"].(int64), int64(scheduleOperationCalls))
	s.mu.Lock()
	defer s.mu.Unlock()
	require.NotEmpty(t, s.creates)
	require.Empty(t, s.updates)
	require.Empty(t, s.deletes)
}

func TestSchedulerOperationRecoveryFreshUpdateWitness(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	s.faults = map[string]map[int]scheduleFault{"update": {2: {code: codes.DeadlineExceeded}}}
	info := scheduleRecoveryInfo(c, logger, true)
	require.NoError(t, info.Options.Set("schedule-updates-per-creation", "2"))
	err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
	require.ErrorContains(t, err, "outcome unknown")
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.updates, 2)
	require.NotEqual(t, s.updates[0].Schedule.State.Notes, s.updates[1].Schedule.State.Notes)
	require.Len(t, s.deletes, 1)
	require.Empty(t, s.records)
}

func TestSchedulerOperationRecoveryParentDeadlineDuringRPC(t *testing.T) {
	c, s, _, logger := scheduleRecoveryRPCFixture(t)
	s.faults = map[string]map[int]scheduleFault{"create": {1: {block: true}}}
	info := scheduleRecoveryInfo(c, logger, true)
	require.NoError(t, info.Options.Set("scheduler-duration-per-iteration", "30s"))
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	require.Len(t, s.creates, 1)
	require.Empty(t, s.updates)
	require.Empty(t, s.deletes)
	require.Zero(t, s.calls["describe"])
}
