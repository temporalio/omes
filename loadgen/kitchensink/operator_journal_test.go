package kitchensink

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/journal"
	activitypb "go.temporal.io/api/activity/v1"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	sdkpb "go.temporal.io/api/sdk/v1"
	"go.temporal.io/api/serviceerror"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/grpc"
)

type operatorJournal struct {
	mu    sync.Mutex
	state journal.State
}

func (m *operatorJournal) Apply(ctx context.Context, _ string, c journal.Command) (journal.State, error) {
	if ctx.Err() != nil {
		return journal.State{}, ctx.Err()
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	n, err := m.state.Apply(c)
	if err == nil {
		m.state = n
	}
	return n.Clone(), err
}

type interruptedOperatorService struct {
	workflowservicepb.WorkflowServiceClient
	cancel                                         context.CancelFunc
	paused                                         bool
	pausedReadFailures                             int
	starts, pauses, unpauses, terminations, resets int
	resetIDs                                       []string
}

func (s *interruptedOperatorService) StartActivityExecution(context.Context, *workflowservicepb.StartActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.StartActivityExecutionResponse, error) {
	s.starts++
	return &workflowservicepb.StartActivityExecutionResponse{RunId: "original-run"}, nil
}
func (s *interruptedOperatorService) DescribeActivityExecution(ctx context.Context, _ *workflowservicepb.DescribeActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.paused && s.pausedReadFailures > 0 {
		s.pausedReadFailures--
		return nil, context.DeadlineExceeded
	}
	info := &activitypb.ActivityExecutionInfo{RunId: "original-run", Status: enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, RunState: enumspb.PENDING_ACTIVITY_STATE_STARTED}
	if s.paused {
		info.Status = enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED
		info.RunState = enumspb.PENDING_ACTIVITY_STATE_PAUSED
	}
	return &workflowservicepb.DescribeActivityExecutionResponse{Info: info}, nil
}
func (s *interruptedOperatorService) PauseActivityExecution(context.Context, *workflowservicepb.PauseActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.PauseActivityExecutionResponse, error) {
	s.pauses++
	s.paused = true
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
		return nil, context.Canceled
	}
	return &workflowservicepb.PauseActivityExecutionResponse{}, nil
}
func (s *interruptedOperatorService) UnpauseActivityExecution(context.Context, *workflowservicepb.UnpauseActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.UnpauseActivityExecutionResponse, error) {
	s.unpauses++
	s.paused = false
	return &workflowservicepb.UnpauseActivityExecutionResponse{}, nil
}
func (s *interruptedOperatorService) ResetActivityExecution(_ context.Context, request *workflowservicepb.ResetActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.ResetActivityExecutionResponse, error) {
	s.resets++
	s.resetIDs = append(s.resetIDs, request.RequestId)
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
		return nil, context.Canceled
	}
	return &workflowservicepb.ResetActivityExecutionResponse{}, nil
}

func TestJournaledResetRestoresTheFrozenDeduplicationID(t *testing.T) {
	backend := &operatorJournal{state: journal.New("lane")}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	service := &interruptedOperatorService{cancel: cancel}
	wrapper := &journalOperatorService{WorkflowServiceClient: service, session: operatorSession(t, backend, 1)}
	request := &workflowservicepb.ResetActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", RunId: "original-run", RequestId: "frozen-reset"}
	_, err := wrapper.ResetActivityExecution(ctx, request)
	require.ErrorIs(t, err, context.Canceled)
	wrapper.session = operatorSession(t, backend, 2)
	request.RequestId = "regenerated-id"
	_, err = wrapper.ResetActivityExecution(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, []string{"frozen-reset", "frozen-reset"}, service.resetIDs)
	require.True(t, backend.state.Pending.Items["operator/reset"].Done)
}

func TestJournaledOperatorPersistsKnownTerminalFailure(t *testing.T) {
	backend := &operatorJournal{state: journal.New("lane")}
	service := &interruptedOperatorService{}
	transport := journaledOperatorClient{service: service, resultErr: temporal.NewTimeoutError(enumspb.TIMEOUT_TYPE_HEARTBEAT, nil)}
	action := &DoStandaloneActivityOperatorCommands{CommandType: DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE, Activity: &ExecuteActivityAction{TaskQueue: "queue", ActivityType: &ExecuteActivityAction_Noop{}}}
	e := &ClientActionsExecutor{Client: transport, Namespace: "namespace", Journal: operatorSession(t, backend, 1)}
	err := e.executeStandaloneActivityOperatorCommands(t.Context(), action)
	require.ErrorIs(t, err, journal.ErrTerminal)
	require.True(t, backend.state.Pending.Items["operator/result"].Done)
	e.Journal = operatorSession(t, backend, 2)
	require.ErrorIs(t, e.executeStandaloneActivityOperatorCommands(t.Context(), action), journal.ErrTerminal)
	require.Equal(t, 1, service.starts)
	require.Equal(t, 1, service.pauses)
	require.NoError(t, e.Journal.Commit(t.Context(), journal.Failed, "terminal heartbeat"))
	require.Equal(t, journal.Counts{Allocated: 1, Failed: 1}, backend.state.Counts)
}
func (s *interruptedOperatorService) TerminateActivityExecution(context.Context, *workflowservicepb.TerminateActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.TerminateActivityExecutionResponse, error) {
	s.terminations++
	return &workflowservicepb.TerminateActivityExecutionResponse{}, nil
}

type journaledOperatorClient struct {
	client.Client
	service   *interruptedOperatorService
	resultErr error
}

func (c journaledOperatorClient) WorkflowService() workflowservicepb.WorkflowServiceClient {
	return c.service
}
func (c journaledOperatorClient) GetActivityHandle(options client.GetActivityHandleOptions) client.ActivityHandle {
	return operatorHandle{runID: options.RunID, err: c.resultErr}
}

type operatorHandle struct {
	client.ActivityHandle
	runID string
	err   error
}

func (h operatorHandle) GetRunID() string               { return h.runID }
func (h operatorHandle) Get(context.Context, any) error { return h.err }
func operatorSession(t *testing.T, backend *operatorJournal, epoch uint64) *journal.Session {
	t.Helper()
	s, err := journal.Claim(t.Context(), backend, "lane", epoch)
	require.NoError(t, err)
	if s.State().Pending == nil {
		require.NoError(t, s.Configure(t.Context(), []byte("contract")))
		require.NoError(t, s.Begin(t.Context(), 1))
	}
	return s
}
func TestJournaledOperatorShutdownDoesNotCleanUpAndResumesPausedRun(t *testing.T) {
	backend := &operatorJournal{state: journal.New("lane")}
	first := operatorSession(t, backend, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	service := &interruptedOperatorService{cancel: cancel}
	c := journaledOperatorClient{service: service}
	action := &DoStandaloneActivityOperatorCommands{CommandType: DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE, Activity: &ExecuteActivityAction{TaskQueue: "queue", ActivityType: &ExecuteActivityAction_Noop{}}}
	e := &ClientActionsExecutor{Client: c, Namespace: "namespace", Journal: first}
	require.ErrorIs(t, e.executeStandaloneActivityOperatorCommands(ctx, action), context.Canceled)
	require.Equal(t, 1, service.starts)
	require.Equal(t, 1, service.pauses)
	require.Zero(t, service.terminations)
	pending := backend.state.Pending.Items["operator/pause"]
	require.False(t, pending.Done)
	require.Equal(t, uint32(1), pending.Calls)
	e = &ClientActionsExecutor{Client: c, Namespace: "namespace", Journal: operatorSession(t, backend, 2)}
	require.NoError(t, e.executeStandaloneActivityOperatorCommands(t.Context(), action))
	require.Equal(t, 1, service.starts)
	require.Equal(t, 1, service.pauses)
	require.Equal(t, 1, service.unpauses)
	require.Zero(t, service.terminations)
	require.True(t, backend.state.Pending.Items["operator/result"].Done)
}
func TestJournaledOperatorNeverReplaysUnprovenReset(t *testing.T) {
	backend := &operatorJournal{state: journal.New("lane")}
	first := operatorSession(t, backend, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	service := &interruptedOperatorService{cancel: cancel}
	wrapper := &journalOperatorService{WorkflowServiceClient: service, session: first}
	request := &workflowservicepb.ResetActivityExecutionRequest{Namespace: "namespace", ActivityId: "activity", RunId: "original-run"}
	_, err := wrapper.ResetActivityExecution(ctx, request)
	require.ErrorIs(t, err, context.Canceled)
	wrapper.session = operatorSession(t, backend, 2)
	_, err = wrapper.ResetActivityExecution(t.Context(), request)
	require.ErrorIs(t, err, journal.ErrUnresolved)
	require.Equal(t, 1, service.resets)
	require.True(t, backend.state.Pending.Items["operator/reset"].Unresolved)
}

func TestJournaledOperatorRecoversPauseVerification(t *testing.T) {
	for _, expired := range []bool{false, true} {
		name := "transient read"
		if expired {
			name = "expired phase"
		}
		t.Run(name, func(t *testing.T) {
			backend := &operatorJournal{state: journal.New("lane")}
			session := operatorSession(t, backend, 1)
			service := &interruptedOperatorService{}
			if expired {
				_, err := session.Plan(t.Context(), "operator/state/paused", time.Now().Add(-time.Second), func() ([]byte, error) { return []byte("paused"), nil })
				require.NoError(t, err)
			} else {
				service.pausedReadFailures = 1
			}
			action := &DoStandaloneActivityOperatorCommands{CommandType: DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE, Activity: &ExecuteActivityAction{TaskQueue: "queue", ActivityType: &ExecuteActivityAction_Noop{}}}
			executor := &ClientActionsExecutor{Client: journaledOperatorClient{service: service}, Namespace: "namespace", Journal: session}

			err := executor.executeStandaloneActivityOperatorCommands(t.Context(), action)
			require.Equal(t, 1, service.starts)
			require.Equal(t, 1, service.pauses)
			if expired {
				require.ErrorIs(t, err, journal.ErrUnresolved)
				require.Equal(t, 1, service.terminations)
				require.Zero(t, service.unpauses)
				require.False(t, backend.state.Pending.Items["operator/result"].Done)
			} else {
				require.NoError(t, err)
				require.Zero(t, service.terminations)
				require.Equal(t, 1, service.unpauses)
				require.True(t, backend.state.Pending.Items["operator/result"].Done)
			}
		})
	}
}

type throttledOperatorStart struct {
	workflowservicepb.WorkflowServiceClient
	requests  []*workflowservicepb.StartActivityExecutionRequest
	describes int
	mismatch  bool
	missing   bool
	lostReply bool
}

func (s *throttledOperatorStart) StartActivityExecution(_ context.Context, req *workflowservicepb.StartActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.StartActivityExecutionResponse, error) {
	s.requests = append(s.requests, req)
	if s.lostReply && len(s.requests) == 1 {
		return nil, context.DeadlineExceeded
	}
	if len(s.requests) == 1 || s.missing || s.lostReply {
		return nil, serviceerror.NewResourceExhausted(enumspb.RESOURCE_EXHAUSTED_CAUSE_RPS_LIMIT, "start throttled")
	}
	return &workflowservicepb.StartActivityExecutionResponse{RunId: "unexpected-start"}, nil
}

func (s *throttledOperatorStart) DescribeActivityExecution(_ context.Context, _ *workflowservicepb.DescribeActivityExecutionRequest, _ ...grpc.CallOption) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
	s.describes++
	if s.missing {
		return nil, serviceerror.NewNotFound("missing activity")
	}
	req := s.requests[0]
	metadata := req.UserMetadata
	if s.mismatch {
		metadata = &sdkpb.UserMetadata{Details: &commonpb.Payload{}}
	}
	return &workflowservicepb.DescribeActivityExecutionResponse{Info: &activitypb.ActivityExecutionInfo{
		RunId: "observed-run", ActivityType: req.ActivityType, TaskQueue: req.TaskQueue.GetName(), UserMetadata: metadata,
	}}, nil
}

func TestJournaledOperatorReconcilesThrottledStart(t *testing.T) {
	for _, mode := range []string{"lost reply", "foreign execution", "no execution"} {
		t.Run(mode, func(t *testing.T) {
			service := &throttledOperatorStart{mismatch: mode == "foreign execution", missing: mode == "no execution", lostReply: mode == "lost reply"}
			session := operatorSession(t, &operatorJournal{state: journal.New("lane")}, 1)
			e := &ClientActionsExecutor{Client: journaledOperatorClient{}, Namespace: "namespace", Journal: session}
			wrapper := &journalOperatorService{WorkflowServiceClient: service, session: session}
			act := &ExecuteActivityAction{TaskQueue: "queue"}
			handle, err := e.startJournaledOperator(t.Context(), wrapper, act, "frozen-activity", "activity-type", nil)
			item := session.State().Pending.Items["operator/start"]
			switch mode {
			case "lost reply":
				require.NoError(t, err)
				require.Equal(t, "observed-run", handle.GetRunID())
				require.True(t, item.Done)
				require.Len(t, service.requests, 2)
			case "foreign execution":
				require.ErrorIs(t, err, journal.ErrConflict)
				require.Len(t, service.requests, 1)
				require.False(t, item.Done)
			case "no execution":
				require.ErrorIs(t, err, journal.ErrUnresolved)
				require.True(t, item.Unresolved)
				require.Len(t, service.requests, 4)
				for _, req := range service.requests {
					require.Equal(t, "frozen-activity", req.GetActivityId())
					require.Equal(t, "frozen-activity", req.GetRequestId())
				}
			}
			expectedDescribes := len(service.requests)
			if mode == "lost reply" {
				expectedDescribes--
			}
			require.Equal(t, expectedDescribes, service.describes)
		})
	}
}
