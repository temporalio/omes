package worker

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/scenarios"
	"github.com/temporalio/omes/workers/go/harness"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
)

// lostNexusAcknowledgements drops the first acknowledgement of the generated
// Nexus Signal-With-Start and, after the target completes, of its terminal
// signal, so the Nexus handler must retry both.
type lostNexusAcknowledgements struct {
	interceptor.ClientInterceptorBase
	interceptor.ClientOutboundInterceptorBase
	client     client.Client
	startLost  atomic.Bool
	signalLost atomic.Bool
	mu         sync.Mutex
	startRuns  []string
}

func (l *lostNexusAcknowledgements) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	l.Next = next
	return l
}

func (l *lostNexusAcknowledgements) SignalWithStartWorkflow(ctx context.Context, input *interceptor.ClientSignalWithStartWorkflowInput) (client.WorkflowRun, error) {
	run, err := l.Next.SignalWithStartWorkflow(ctx, input)
	if err != nil || !strings.Contains(run.GetID(), "/nexus-workflow-") {
		return run, err
	}
	l.mu.Lock()
	l.startRuns = append(l.startRuns, run.GetRunID())
	l.mu.Unlock()
	if l.startLost.CompareAndSwap(false, true) {
		return nil, serviceerror.NewUnavailable("injected loss after Signal-With-Start")
	}
	return run, nil
}

func (l *lostNexusAcknowledgements) SignalWorkflow(ctx context.Context, input *interceptor.ClientSignalWorkflowInput) error {
	err := l.Next.SignalWorkflow(ctx, input)
	if err != nil || !strings.Contains(input.WorkflowID, "/nexus-workflow-") || !l.signalLost.CompareAndSwap(false, true) {
		return err
	}
	if err := l.client.GetWorkflow(ctx, input.WorkflowID, input.RunID).Get(ctx, nil); err != nil {
		return err
	}
	return serviceerror.NewUnavailable("injected loss after the terminal signal completed its target")
}

// A lost acknowledgement that is not retried, or a retry the server does not
// deduplicate, fails the run.
func TestGeneratedNexusSequenceSurvivesLostAcknowledgements(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	runID := "generated-nexus-" + uuid.NewString()
	server := workertest.StartDevServer(t, workertest.WithDynamicConfig(map[string]any{
		"history.enableChasm":                true,
		"history.enableCHASMCallbacks":       true,
		"history.enableCHASMSignalBacklinks": true,
	}))
	env := workertest.SetupTestEnvironment(t, workertest.WithDevServer(server), workertest.WithNexusEndpoint(runID))

	base, err := buildClient(harness.ClientConfig{TargetHost: env.DevServerAddress(), Namespace: env.Namespace()})
	require.NoError(t, err)
	t.Cleanup(base.Close)
	loss := &lostNexusAcknowledgements{client: env.TemporalClient()}
	workerClient, err := client.NewClientFromExisting(base, client.Options{
		Namespace:     env.Namespace(),
		DataConverter: clioptions.OmesDataConverter(),
		Interceptors:  []interceptor.ClientInterceptor{loss},
	})
	require.NoError(t, err)
	t.Cleanup(workerClient.Close)
	w := buildWorker(workerClient, harness.WorkerContext{TaskQueue: loadgen.TaskQueueForRun(runID)})
	require.NoError(t, w.Start())
	t.Cleanup(w.Stop)

	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
	defer cancel()
	info := loadgen.ScenarioInfo{
		RunID: runID, ExecutionID: uuid.NewString(), Namespace: env.Namespace(), Client: env.TemporalClient(),
		Logger: zap.NewNop().Sugar(), MetricsHandler: client.MetricsNopHandler,
		Configuration: loadgen.RunConfiguration{Iterations: 1},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			scenarios.IterFlag:                                      "1",
			scenarios.SleepTimeFlag:                                 "1ms",
			scenarios.VisibilityVerificationTimeoutFlag:             "10s",
			scenarios.NexusEnabledFlag:                              "true",
			scenarios.NexusEndpointFlag:                             env.NexusEndpointName(),
			scenarios.IncludeNexusWorkflowActionsFlag:               "true",
			scenarios.IncludeStandaloneNexusFlag:                    "false",
			scenarios.IncludeStandaloneActivityFlag:                 "false",
			scenarios.IncludeStandaloneActivityOperatorCommandsFlag: "false",
		}),
	}
	require.NoError(t, info.RegisterDefaultSearchAttributes(ctx))
	require.NoError(t, loadgen.GetScenario("throughput_stress").ExecutorFn().Run(ctx, info))
	require.True(t, loss.startLost.Load())
	require.True(t, loss.signalLost.Load())

	targetID := info.NewRun(1).DefaultStartWorkflowOptions().ID + "/nexus-workflow-0"
	target, err := env.TemporalClient().DescribeWorkflowExecution(ctx, targetID, "")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(loss.startRuns), 2)
	for _, runID := range loss.startRuns {
		require.Equal(t, target.WorkflowExecutionInfo.Execution.RunId, runID,
			"a retried Signal-With-Start must reuse the target's run")
	}
}
