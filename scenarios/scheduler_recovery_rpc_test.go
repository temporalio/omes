package scenarios

import (
	"context"
	"maps"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	commonpb "go.temporal.io/api/common/v1"
	schedulepb "go.temporal.io/api/schedule/v1"
	"go.temporal.io/api/serviceerror"
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
	delayedEffect bool
	foreign       bool
	cancel        context.CancelFunc
	block         bool
	vanish        bool
}

type scheduleRecord struct {
	schedule           *schedulepb.Schedule
	memo               *commonpb.Memo
	pending            *schedulepb.Schedule
	probesUntilApplied int
}

type scheduleRecoveryService struct {
	workflowservice.UnimplementedWorkflowServiceServer
	mu             sync.Mutex
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
		s.records[r.ScheduleId] = scheduleRecord{schedule: proto.Clone(r.Schedule).(*schedulepb.Schedule), memo: memo}
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
	if record.pending != nil {
		if record.probesUntilApplied == 0 {
			record.schedule = record.pending
			record.pending = nil
		} else {
			record.probesUntilApplied--
		}
		s.records[r.ScheduleId] = record
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
		if f.delayedEffect {
			record.pending = updated
			record.probesUntilApplied = 1
		} else {
			record.schedule = updated
		}
		s.records[r.ScheduleId] = record
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
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-done) })
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	c, err := client.DialContext(ctx, client.Options{HostPort: listener.Addr().String(), Namespace: "scheduler-recovery"})
	require.NoError(t, err)
	t.Cleanup(c.Close)
	core, logs := observer.New(zap.InfoLevel)
	return c, s, logs, zap.New(core).Sugar()
}

func scheduleRecoveryInfo(c client.Client, logger *zap.SugaredLogger, enabled bool, overrides map[string]string) loadgen.ScenarioInfo {
	values := map[string]string{
		ScheduleCreationPerIterationFlag: "1", ScheduleReadsPerCreationFlag: "1", ScheduleUpdatesPerCreationFlag: "1",
		SchedulerDurationPerIterationFlag: "1s", WaitTimeBeforeCleanupFlag: "0s", OperationIntervalFlag: "1ms",
		RecoverScheduleOperationsFlag: strconv.FormatBool(enabled),
	}
	maps.Copy(values, overrides)
	options := loadgen.MustResolveScenarioOptions("scheduler_stress", values)
	return loadgen.ScenarioInfo{RunID: "crud-recovery", Client: c, Logger: logger, Options: options}
}

func TestSchedulerOperationRecoverySDK(t *testing.T) {
	cases := []struct {
		name                                 string
		faults                               map[string]map[int]scheduleFault
		wantErr                              error
		wantCode                             codes.Code
		options                              map[string]string
		creates, updates, deletes, remaining int
		outcome                              scheduleOutcome
		sdkRetry                             bool
	}{
		{name: "healthy", creates: 1, updates: 1, deletes: 1},
		{name: "read_permanent_rejection", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.PermissionDenied}}}, wantCode: codes.PermissionDenied, creates: 1, updates: 1, deletes: 1},
		{name: "create_last_probe_not_found_no_fifth_call", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded}}}, wantErr: errScheduleRecoveryBudget, creates: 2},
		{name: "sdk_unavailable_retries_same_rpc_request", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.Unavailable}}}, creates: 2, updates: 1, deletes: 1, sdkRetry: true},
		{name: "sdk_accepted_create_then_exists", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.Unavailable, apply: true}}}, creates: 2, updates: 1, deletes: 1, sdkRetry: true, outcome: outcomeObservedCreate},
		{name: "describe_not_found_compatible", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.NotFound, apply: true}}}, creates: 1, outcome: outcomeNotFoundCompatible},
		{name: "update_describe_not_found_compatible", faults: map[string]map[int]scheduleFault{"describe": {2: {code: codes.NotFound, apply: true}}}, creates: 1, outcome: outcomeNotFoundCompatible},
		{name: "delete_not_found_compatible", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.NotFound, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeNotFoundCompatible},
		{name: "create_accepted_response_lost", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeObservedCreate},
		{name: "create_not_accepted_same_frozen_intent", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded}}}, creates: 2, updates: 1, deletes: 1, outcome: outcomeAcknowledged},
		{name: "read_deadline", faults: map[string]map[int]scheduleFault{"describe": {1: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeAcknowledged},
		{name: "update_internal_describe_deadline", faults: map[string]map[int]scheduleFault{"describe": {2: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeAcknowledged},
		{name: "update_accepted_response_lost", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeObservedWitness},
		{name: "update_late_effect_no_replay", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true, delayedEffect: true}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeObservedWitness},
		{name: "update_identical_old_payload_not_success", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded}}}, wantErr: errScheduleUpdateUnknown, creates: 1, updates: 1, deletes: 1},
		{name: "update_response_lost_then_absent_is_unknown", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.DeadlineExceeded, apply: true, vanish: true}}}, wantErr: errScheduleUpdateUnknown, creates: 1, updates: 1},
		{name: "update_permanent_rejection", faults: map[string]map[int]scheduleFault{"update": {1: {code: codes.PermissionDenied}}}, wantCode: codes.PermissionDenied, creates: 1, updates: 1, deletes: 1},
		{name: "delete_accepted_response_lost", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded, apply: true}}}, creates: 1, updates: 1, deletes: 1, outcome: outcomeObservedAbsence},
		{name: "delete_not_accepted_retry_after_owned_probe", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded}}}, creates: 1, updates: 1, deletes: 2, outcome: outcomeAcknowledged},
		{name: "delete_last_response_lost_no_fifth_probe", faults: map[string]map[int]scheduleFault{"delete": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded, apply: true}}}, wantErr: errScheduleRecoveryBudget, creates: 1, updates: 1, deletes: 2},
		{name: "foreign_owner_never_adopted_or_deleted", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.AlreadyExists, apply: true, foreign: true}}}, wantErr: errScheduleIntentMismatch, creates: 1, remaining: 1},
		{name: "create_probe_exhausted_namespace_cleanup_owned", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.DeadlineExceeded, apply: true}}, "describe": {1: {code: codes.DeadlineExceeded}, 2: {code: codes.DeadlineExceeded}, 3: {code: codes.DeadlineExceeded}}}, wantErr: errScheduleRecoveryBudget, creates: 1, remaining: 1},
		{name: "fresh_witness_per_update", options: map[string]string{ScheduleUpdatesPerCreationFlag: "2"}, faults: map[string]map[int]scheduleFault{"update": {2: {code: codes.DeadlineExceeded}}}, wantErr: errScheduleUpdateUnknown, creates: 1, updates: 2, deletes: 1},
		{name: "retained_without_delete", options: map[string]string{RetainSchedulesFlag: "true"}, creates: 1, updates: 1, remaining: 1},
		{name: "permanent_create_rejection", faults: map[string]map[int]scheduleFault{"create": {1: {code: codes.InvalidArgument}}}, wantCode: codes.InvalidArgument, creates: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, s, logs, logger := scheduleRecoveryRPCFixture(t)
			if tc.faults != nil {
				s.faults = tc.faults
			}
			info := scheduleRecoveryInfo(c, logger, true, tc.options)
			err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else if tc.wantCode != codes.OK {
				require.Equal(t, tc.wantCode, serviceerror.ToStatus(err).Code())
			} else {
				require.NoError(t, err)
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
			witnesses := map[string]bool{}
			for _, u := range s.updates {
				require.False(t, witnesses[u.Schedule.State.Notes], "each update must carry a fresh witness")
				witnesses[u.Schedule.State.Notes] = true
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
					if e.ContextMap()["outcome"] == string(tc.outcome) {
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
	info := scheduleRecoveryInfo(c, logger, false, nil)
	err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
	require.Equal(t, codes.DeadlineExceeded, serviceerror.ToStatus(err).Code())
	require.Len(t, s.creates, 1)
	require.Nil(t, s.creates[0].Memo)
	require.Zero(t, s.calls["describe"])
	require.Len(t, s.records, 1)
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
			info := scheduleRecoveryInfo(c, logger, true, nil)
			err := new(SchedulerExecutor).Execute(t.Context(), info.NewRun(1))
			require.ErrorIs(t, err, errScheduleIntentMismatch)
			require.Empty(t, s.updates)
			require.Empty(t, s.deletes)
			require.Len(t, s.records, 1)
		})
	}
}

// Cancellation stops the iteration promptly at every stage, without cleanup or further probes.
func TestSchedulerOperationRecoveryCancellation(t *testing.T) {
	for _, tc := range []struct {
		name    string
		faults  func(cancel context.CancelFunc) map[string]map[int]scheduleFault
		options map[string]string
		// Zero cancels only through faults; otherwise the parent deadline.
		timeout                       time.Duration
		wantErr                       error
		updates, describes, remaining int
	}{
		{
			name: "cancel_during_create",
			faults: func(cancel context.CancelFunc) map[string]map[int]scheduleFault {
				return map[string]map[int]scheduleFault{"create": {1: {cancel: cancel}}}
			},
			wantErr: context.Canceled,
		},
		{
			name: "deadline_during_create_rpc",
			faults: func(context.CancelFunc) map[string]map[int]scheduleFault {
				return map[string]map[int]scheduleFault{"create": {1: {block: true}}}
			},
			options: map[string]string{SchedulerDurationPerIterationFlag: "30s"},
			timeout: 100 * time.Millisecond, wantErr: context.DeadlineExceeded,
		},
		{
			name:    "deadline_at_operation_tick",
			options: map[string]string{OperationIntervalFlag: "1h"},
			timeout: 100 * time.Millisecond, wantErr: context.DeadlineExceeded, remaining: 1,
		},
		{
			name: "cancel_after_prepared_update",
			faults: func(cancel context.CancelFunc) map[string]map[int]scheduleFault {
				return map[string]map[int]scheduleFault{"update": {1: {apply: true, cancel: cancel}}}
			},
			wantErr: context.Canceled, updates: 1, describes: 2, remaining: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, s, _, logger := scheduleRecoveryRPCFixture(t)
			ctx, cancel := context.WithCancel(t.Context())
			if tc.timeout > 0 {
				ctx, cancel = context.WithTimeout(t.Context(), tc.timeout)
			}
			defer cancel()
			if tc.faults != nil {
				s.faults = tc.faults(cancel)
			}
			info := scheduleRecoveryInfo(c, logger, true, tc.options)
			start := time.Now()
			err := new(SchedulerExecutor).Execute(ctx, info.NewRun(1))
			require.ErrorIs(t, err, tc.wantErr)
			require.Less(t, time.Since(start), time.Second)
			s.mu.Lock()
			defer s.mu.Unlock()
			require.Len(t, s.creates, 1)
			require.Len(t, s.updates, tc.updates)
			require.Equal(t, tc.describes, s.calls["describe"])
			require.Empty(t, s.deletes)
			require.Len(t, s.records, tc.remaining)
		})
	}
}
