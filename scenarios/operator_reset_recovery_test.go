package scenarios

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/journal"
	"go.temporal.io/api/serviceerror"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"google.golang.org/grpc"
)

type resetReplyLoss struct {
	workflowservice.WorkflowServiceClient
	mu              sync.Mutex
	ids             []string
	resetStarted    time.Time
	originalStarted time.Time
	target          *workflowservice.DescribeActivityExecutionRequest
}

func (s *resetReplyLoss) ResetActivityExecution(ctx context.Context, request *workflowservice.ResetActivityExecutionRequest, options ...grpc.CallOption) (*workflowservice.ResetActivityExecutionResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	before, err := s.WorkflowServiceClient.DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{Namespace: request.Namespace, ActivityId: request.ActivityId, RunId: request.RunId})
	if err != nil {
		return nil, err
	}
	response, err := s.WorkflowServiceClient.ResetActivityExecution(ctx, request, options...)
	if err != nil {
		return nil, err
	}
	s.ids = append(s.ids, request.RequestId)
	if len(s.ids) == 1 {
		s.originalStarted = before.GetInfo().GetLastStartedTime().AsTime()
		s.target = &workflowservice.DescribeActivityExecutionRequest{Namespace: request.Namespace, ActivityId: request.ActivityId, RunId: request.RunId}
		return nil, serviceerror.NewDeadlineExceeded("reset applied; reply lost")
	}
	until := time.NewTimer(1500 * time.Millisecond)
	defer until.Stop()
	for {
		description, err := s.WorkflowServiceClient.DescribeActivityExecution(ctx, &workflowservice.DescribeActivityExecutionRequest{Namespace: request.Namespace, ActivityId: request.ActivityId, RunId: request.RunId})
		if err != nil {
			return nil, err
		}
		if !description.GetInfo().GetLastStartedTime().AsTime().Equal(s.resetStarted) {
			return nil, serviceerror.NewFailedPrecondition("same reset request changed the activity attempt twice")
		}
		select {
		case <-until.C:
			return response, nil
		case <-time.After(20 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

type resetRecoveryJournal struct {
	journal.Client
	service *resetReplyLoss
}

func (b *resetRecoveryJournal) Apply(ctx context.Context, lane string, command journal.Command) (journal.State, error) {
	if command.Kind == "issue" && command.Key == "operator/reset" && command.Call == 2 {
		if err := b.service.awaitResetStart(ctx); err != nil {
			return journal.State{}, err
		}
	}
	return b.Client.Apply(ctx, lane, command)
}
func (s *resetReplyLoss) awaitResetStart(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for {
		after, err := s.WorkflowServiceClient.DescribeActivityExecution(ctx, s.target)
		if err != nil {
			return err
		}
		if after.GetInfo().GetLastStartedTime().AsTime().After(s.originalStarted) {
			s.resetStarted = after.GetInfo().GetLastStartedTime().AsTime()
			return nil
		}
		select {
		case <-time.After(10 * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

type resetLossClient struct {
	client.Client
	service *resetReplyLoss
}

func (c resetLossClient) WorkflowService() workflowservice.WorkflowServiceClient { return c.service }

func TestJournaledOperatorsRecoverRealResetReplyLoss(t *testing.T) {
	server := workertest.StartDevServer(t, workertest.WithDynamicConfig(map[string]any{"activity.enableStandalone": true, "history.enableStandaloneActivityOperatorCommands": true}))
	env := workertest.SetupTestEnvironment(t, workertest.WithDevServer(server), workertest.WithExecutorTimeout(time.Minute))
	backend := &scenarioJournal{}
	recovery := &resetRecoveryJournal{Client: backend}
	name := "standalone_activity_operator_commands"
	info := loadgen.ScenarioInfo{ScenarioName: name, RunID: "reset-reply-loss", ExecutionID: "reset-reply-loss", Configuration: loadgen.RunConfiguration{Iterations: 3, MaxConcurrent: 3, MaxIterationAttempts: 1, DoNotRegisterSearchAttributes: true}, Options: loadgen.MustResolveScenarioOptions(name, map[string]string{StandaloneActivityOperatorSuccessDurationFlag: "20s", StandaloneActivityOperatorHeartbeatTimeoutFlag: "10s", StandaloneActivityOperatorStartToCloseTimeoutFlag: "30s", StandaloneActivityOperatorMaxAttemptsFlag: "5"}), Recovery: &loadgen.RecoveryOptions{Client: recovery, Epoch: 1, Contract: []byte("operator reset contract"), Streams: map[string][]string{name: {"operator/0", "operator/1", "operator/2"}}, GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(2 * time.Minute)}}
	var service *resetReplyLoss
	executor := loadgen.ExecutorFunc(func(ctx context.Context, info loadgen.ScenarioInfo) error {
		service = &resetReplyLoss{WorkflowServiceClient: info.Client.WorkflowService()}
		recovery.service = service
		info.Client = resetLossClient{Client: info.Client, service: service}
		return runStandaloneActivityOperatorCommands(ctx, info)
	})
	_, err := env.RunExecutorTest(t, executor, info, clioptions.LangGo)
	require.NoError(t, err)
	require.Len(t, service.ids, 2)
	require.NotEmpty(t, service.ids[0])
	require.Equal(t, service.ids[0], service.ids[1])
	total, err := info.Recovery.Totals(t.Context(), name)
	require.NoError(t, err)
	require.Equal(t, journal.Counts{Allocated: 3, Succeeded: 3}, total)
}
