package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/converter"
)

func TestFiniteLifecycleActualRunAndStaleHeartbeat(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(2*time.Minute))
	info := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("finite-lifecycle-%d", time.Now().UnixNano()), ExecutionID: "retained-execution",
		Configuration: loadgen.RunConfiguration{Iterations: 2},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag: "2", ContinueAsNewAfterIterFlag: "1", SleepTimeFlag: "1ms",
			NexusEnabledFlag: "false", IncludeStandaloneActivityFlag: "false",
			IncludeStandaloneActivityOperatorCommandsFlag: "false",
			FiniteScheduleRootPercentageFlag:              "100",
		}),
	}
	first := newThroughputStressExecutor()
	_, err := env.RunExecutorTest(t, first, info, clioptions.LangGo)
	require.NoError(t, err)
	complete := first.Snapshot().(tpsState)
	require.Equal(t, 2, complete.CompletedIterations)
	require.Equal(t, 2, complete.FiniteSchedules.LastPlannedIteration)
	require.Equal(t, 2, complete.FiniteSchedules.VerifiedThrough)
	defer func() {
		for iteration := 1; iteration <= 2; iteration++ {
			for _, input := range complete.FiniteSchedules.inputsForRoot(iteration) {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := ks.DeleteFiniteSchedule(cleanupCtx, env.TemporalClient().ScheduleClient(), input, loadgen.TaskQueueForRun(info.RunID))
				cancel()
				require.NoError(t, err)
			}
		}
	}()
	for _, input := range complete.FiniteSchedules.inputsForRoot(1) {
		_, err := env.TemporalClient().ScheduleClient().GetHandle(t.Context(), input.ScheduleID).Describe(t.Context())
		require.NoError(t, err, "successful activity must retain evidence until parent cleanup")
	}
	stale := complete
	stale.FiniteSchedules = complete.FiniteSchedules.clone()
	stale.CompletedIterations = 1
	stale.FiniteSchedules.LastPlannedIteration = 1
	stale.FiniteSchedules.VerifiedThrough = 0
	dc := converter.GetDefaultDataConverter()
	payload, err := dc.ToPayload(stale)
	require.NoError(t, err)
	resumed := newThroughputStressExecutor()
	require.NoError(t, resumed.LoadState(func(target any) error { return dc.FromPayload(payload, target) }))
	created := 0
	resumed.onActionsCreated = func([]*ks.ActionSet) { created++ }
	info.Configuration.Iterations = 0
	info.Configuration.Duration = 0
	_, err = env.RunExecutorTest(t, resumed, info, clioptions.LangGo)
	require.NoError(t, err)
	require.Zero(t, created, "expired generation must not silently start another run")
	recovered := resumed.Snapshot().(tpsState)
	require.Equal(t, 2, recovered.FiniteSchedules.LastPlannedIteration, "server inventory must recover root omitted from heartbeat")
	require.Equal(t, 2, recovered.FiniteSchedules.VerifiedThrough)
	require.Equal(t, 2, recovered.CompletedIterations)
	missingInput := complete.FiniteSchedules.inputsForRoot(2)[0]
	cleanupCtx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	require.NoError(t, ks.DeleteFiniteSchedule(cleanupCtx, env.TemporalClient().ScheduleClient(), missingInput, loadgen.TaskQueueForRun(info.RunID)))
	cancel()
	missing := newThroughputStressExecutor()
	require.NoError(t, missing.LoadState(func(target any) error { return dc.FromPayload(payload, target) }))
	_, err = env.RunExecutorTest(t, missing, info, clioptions.LangGo)
	var notFound *serviceerror.NotFound
	require.ErrorAs(t, err, &notFound, "missing retained evidence must fail rather than restart or pass")
	require.Zero(t, missing.Snapshot().(tpsState).FiniteSchedules.VerifiedThrough)
}

func TestFiniteLifecyclePercentageBounds(t *testing.T) {
	for _, value := range []string{"-1", "101"} {
		executor := newThroughputStressExecutor()
		err := executor.Configure(loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			FiniteScheduleRootPercentageFlag: value,
		})})
		require.ErrorContains(t, err, FiniteScheduleRootPercentageFlag)
	}
}

func TestFiniteLifecycleReconcilesActualRoots(t *testing.T) {
	env := workertest.SetupTestEnvironment(t)
	info := finiteScheduleTestInfo("1")
	info.RunID = fmt.Sprintf("inventory-%d", time.Now().UnixNano())
	info.Namespace = env.Namespace()
	info.Client = env.TemporalClient()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	require.NoError(t, info.RegisterDefaultSearchAttributes(ctx))
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	executor.config.FiniteScheduleRootPercentage = 100
	require.NoError(t, executor.prepareFiniteScheduleState(info))
	require.NoError(t, executor.recordFiniteScheduleRoot(1))
	var started []string
	defer func() {
		for _, id := range started {
			cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := info.Client.TerminateWorkflow(cleanupCtx, id, "", "inventory probe cleanup")
			cleanupCancel()
			require.NoError(t, err)
		}
	}()
	for _, id := range []string{
		info.NewRun(1).DefaultStartWorkflowOptions().ID,
		info.NewRun(2).DefaultStartWorkflowOptions().ID,
		info.NewRun(2).DefaultStartWorkflowOptions().ID + "/child-999",
		"foreign-root-999",
	} {
		options := info.NewRun(1).DefaultStartWorkflowOptions()
		options.ID = id
		_, err := info.Client.ExecuteWorkflow(ctx, options, "inventory-probe")
		require.NoError(t, err)
		started = append(started, id)
	}
	require.NoError(t, executor.reconcileFiniteScheduleRoots(ctx, info))
	require.Equal(t, 2, executor.Snapshot().(tpsState).FiniteSchedules.LastPlannedIteration)
}
