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
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/scenarios"
	"github.com/temporalio/omes/workers/go/harness"
	enumspb "go.temporal.io/api/enums/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/interceptor"
	"go.uber.org/zap"
)

type generatedNexusResponseLoss struct {
	interceptor.ClientInterceptorBase
	client     client.Client
	signalLost atomic.Bool
	startLost  atomic.Bool
	signals    atomic.Int32
	mu         sync.Mutex
	starts     []string
}

func (loss *generatedNexusResponseLoss) InterceptClient(next interceptor.ClientOutboundInterceptor) interceptor.ClientOutboundInterceptor {
	return &generatedNexusResponseLossOutbound{ClientOutboundInterceptorBase: interceptor.ClientOutboundInterceptorBase{Next: next}, loss: loss}
}

type generatedNexusResponseLossOutbound struct {
	interceptor.ClientOutboundInterceptorBase
	loss *generatedNexusResponseLoss
}

func (out *generatedNexusResponseLossOutbound) SignalWithStartWorkflow(ctx context.Context, input *interceptor.ClientSignalWithStartWorkflowInput) (client.WorkflowRun, error) {
	run, err := out.Next.SignalWithStartWorkflow(ctx, input)
	if err != nil || !strings.Contains(run.GetID(), "/nexus-workflow-") {
		return run, err
	}
	out.loss.mu.Lock()
	out.loss.starts = append(out.loss.starts, run.GetRunID())
	out.loss.mu.Unlock()
	if out.loss.startLost.CompareAndSwap(false, true) {
		return nil, serviceerror.NewUnavailable("injected loss after generated nonterminal Signal-With-Start")
	}
	return run, nil
}

func (out *generatedNexusResponseLossOutbound) SignalWorkflow(ctx context.Context, input *interceptor.ClientSignalWorkflowInput) error {
	err := out.Next.SignalWorkflow(ctx, input)
	if !strings.Contains(input.WorkflowID, "/nexus-workflow-") {
		return err
	}
	out.loss.signals.Add(1)
	if err == nil && out.loss.signalLost.CompareAndSwap(false, true) {
		if err := out.loss.client.GetWorkflow(ctx, input.WorkflowID, input.RunID).Get(ctx, nil); err != nil {
			return err
		}
		return serviceerror.NewUnavailable("injected loss after generated terminal signal completes target")
	}
	return err
}

func TestGeneratedNexusSequenceSurvivesLostAcknowledgements(t *testing.T) {
	if testing.Short() {
		t.Skip("local Temporal server required")
	}
	address := startNexusSignalTestServer(t)
	c, err := client.Dial(client.Options{HostPort: address, DataConverter: clioptions.OmesDataConverter()})
	require.NoError(t, err)
	defer c.Close()
	base, err := buildClient(harness.ClientConfig{TargetHost: address, Namespace: "default"})
	require.NoError(t, err)
	defer base.Close()
	loss := &generatedNexusResponseLoss{client: c}
	workerClient, err := client.NewClientFromExisting(base, client.Options{
		Namespace: "default", DataConverter: clioptions.OmesDataConverter(),
		Interceptors: []interceptor.ClientInterceptor{loss},
	})
	require.NoError(t, err)
	defer workerClient.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 35*time.Second)
	defer cancel()
	runID := "generated-nexus-" + uuid.NewString()
	queue := loadgen.TaskQueueForRun(runID)
	w := buildWorker(workerClient, harness.WorkerContext{TaskQueue: queue})
	require.NoError(t, w.Start())
	defer w.Stop()
	_, err = c.OperatorService().CreateNexusEndpoint(ctx, &operatorservice.CreateNexusEndpointRequest{
		Spec: &nexuspb.EndpointSpec{
			Name: runID,
			Target: &nexuspb.EndpointTarget{Variant: &nexuspb.EndpointTarget_Worker_{
				Worker: &nexuspb.EndpointTarget_Worker{Namespace: "default", TaskQueue: queue},
			}},
		},
	})
	require.NoError(t, err)
	info := loadgen.ScenarioInfo{
		RunID: runID, ExecutionID: uuid.NewString(), Namespace: "default", Client: c,
		Logger: zap.NewNop().Sugar(), MetricsHandler: client.MetricsNopHandler,
		Configuration: loadgen.RunConfiguration{Iterations: 1},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			scenarios.IterFlag: "1", scenarios.SleepTimeFlag: "1ms",
			scenarios.VisibilityVerificationTimeoutFlag: "10s",
			scenarios.NexusEnabledFlag:                  "true", scenarios.NexusEndpointFlag: runID,
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
	require.GreaterOrEqual(t, loss.signals.Load(), int32(2))
	loss.mu.Lock()
	starts := append([]string(nil), loss.starts...)
	loss.mu.Unlock()
	require.GreaterOrEqual(t, len(starts), 2)
	for _, id := range starts[1:] {
		require.Equal(t, starts[0], id, "nonterminal Signal-With-Start retry must reuse the actual run")
	}
	parentID := info.NewRun(1).DefaultStartWorkflowOptions().ID
	parent, err := c.DescribeWorkflowExecution(ctx, parentID, "")
	require.NoError(t, err)
	require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, parent.WorkflowExecutionInfo.Status)
	targetID := parentID + "/nexus-workflow-0"
	target, err := c.DescribeWorkflowExecution(ctx, targetID, "")
	require.NoError(t, err)
	require.Equal(t, enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED, target.WorkflowExecutionInfo.Status)
	require.Equal(t, starts[0], target.WorkflowExecutionInfo.Execution.RunId)
}
