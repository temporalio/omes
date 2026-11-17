package kitchensink

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/journal"
	activitypb "go.temporal.io/api/activity/v1"
	enumspb "go.temporal.io/api/enums/v1"
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
	starts, pauses, unpauses, terminations, resets int
	resetIDs                                       []string
}

func (s *interruptedOperatorService) StartActivityExecution(context.Context, *workflowservicepb.StartActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.StartActivityExecutionResponse, error) {
	s.starts++
	return &workflowservicepb.StartActivityExecutionResponse{RunId: "original-run"}, nil
}
func (s *interruptedOperatorService) DescribeActivityExecution(context.Context, *workflowservicepb.DescribeActivityExecutionRequest, ...grpc.CallOption) (*workflowservicepb.DescribeActivityExecutionResponse, error) {
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
