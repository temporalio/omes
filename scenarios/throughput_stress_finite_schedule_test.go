package scenarios

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
)

func finiteScheduleInputs(t *testing.T, actions []*ks.Action) []ks.FiniteScheduleInput {
	t.Helper()
	var inputs []ks.FiniteScheduleInput
	for _, action := range actions {
		if nested := action.GetNestedActionSet(); nested != nil {
			inputs = append(inputs, finiteScheduleInputs(t, nested.Actions)...)
		}
		if generic := action.GetExecActivity().GetGeneric(); generic != nil && generic.Type == ks.CreateFiniteScheduleActivityName {
			activity := action.GetExecActivity()
			require.Equal(t, 10*time.Second, activity.StartToCloseTimeout.AsDuration())
			require.Equal(t, 30*time.Second, activity.ScheduleToCloseTimeout.AsDuration())
			require.Equal(t, int32(3), activity.RetryPolicy.MaximumAttempts)
			require.Equal(t, ks.ActivityCancellationType_WAIT_CANCELLATION_COMPLETED, activity.GetRemote().CancellationType)
			var input ks.FiniteScheduleInput
			require.NoError(t, converter.GetDefaultDataConverter().FromPayloads(&commonpb.Payloads{Payloads: generic.Arguments}, &input))
			inputs = append(inputs, input)
		}
		if action.GetContinueAsNew() != nil {
			inputs = append(inputs, finiteScheduleInputs(t, decodeContinueAsNewChunk(t, actions))...)
		}
	}
	return inputs
}

func finiteScheduleTestInfo(can string) loadgen.ScenarioInfo {
	return loadgen.ScenarioInfo{
		RunID: "finite-profile", ExecutionID: "execution-one", Logger: zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag: "4", ContinueAsNewAfterIterFlag: can, NexusEnabledFlag: "false",
			PayloadDistributionJsonFlag: `{"size":{"type":"uniform","min":1,"max":1000000}}`,
		}),
	}
}

func TestThroughputFiniteScheduleGeneration(t *testing.T) {
	var canonical []ks.FiniteScheduleInput
	for _, can := range []string{"0", "1", "3"} {
		t.Run("CAN-"+can, func(t *testing.T) {
			info := finiteScheduleTestInfo(can)
			executor := newThroughputStressExecutor()
			require.NoError(t, executor.Configure(info))
			executor.config.FiniteScheduleRootPercentage = 100
			sets := executor.createActions(info.NewRun(1))
			encoded, err := converter.NewProtoJSONPayloadConverter().ToPayload(sets[0])
			require.NoError(t, err)
			var decoded ks.ActionSet
			require.NoError(t, converter.NewProtoJSONPayloadConverter().FromPayload(encoded, &decoded))
			inputs := finiteScheduleInputs(t, decoded.Actions)
			require.Len(t, inputs, 4, "one finite schedule per internal iteration")
			ids := make(map[string]bool)
			for _, input := range inputs {
				require.Contains(t, input.ScheduleID, "omes-finite-")
				require.False(t, ids[input.ScheduleID])
				ids[input.ScheduleID] = true
			}
			if canonical == nil {
				canonical = inputs
			} else {
				require.Equal(t, canonical, inputs, "CAN chunking must not change schedule identity")
			}
			otherRoot := finiteScheduleInputs(t, executor.createActions(info.NewRun(2))[0].Actions)
			for _, input := range otherRoot {
				require.False(t, ids[input.ScheduleID], "outer iterations must not share schedules")
			}
			restarted := newThroughputStressExecutor()
			require.NoError(t, restarted.Configure(info))
			restarted.config.FiniteScheduleRootPercentage = 100
			require.Equal(t, inputs, finiteScheduleInputs(t, restarted.createActions(info.NewRun(1))[0].Actions))
		})
	}
}

func TestThroughputFiniteScheduleDisabled(t *testing.T) {
	info := finiteScheduleTestInfo("1")
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	require.Empty(t, finiteScheduleInputs(t, executor.createActions(info.NewRun(1))[0].Actions))
}

func TestThroughputFiniteScheduleSelectionAndIsolation(t *testing.T) {
	info := finiteScheduleTestInfo("3")
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	executor.config.FiniteScheduleRootPercentage = 50
	selected := 0
	for iteration := 1; iteration <= 200; iteration++ {
		run := info.NewRun(iteration)
		inputs := finiteScheduleInputs(t, executor.createActions(run)[0].Actions)
		if finiteScheduleRootSelected(run.DefaultStartWorkflowOptions().ID, 50) {
			selected++
			require.Len(t, inputs, 4, "selection applies to the whole root, not individual internal steps")
		} else {
			require.Empty(t, inputs)
		}
	}
	require.InDelta(t, 100, selected, 40, "fixed root identities must exercise selected and unselected classes")
	baseID := info.NewRun(1).DefaultStartWorkflowOptions().ID
	first := finiteScheduleID(baseID, 0)
	info.ExecutionID = "execution-two"
	require.NotEqual(t, first, finiteScheduleID(info.NewRun(1).DefaultStartWorkflowOptions().ID, 0))
	info.RunID = "another-scenario"
	require.NotEqual(t, first, finiteScheduleID(info.NewRun(1).DefaultStartWorkflowOptions().ID, 0))
}

func TestThroughputFiniteSchedulePreservesPayloadSequence(t *testing.T) {
	info := finiteScheduleTestInfo("1")
	generate := func(percentage int) []*ks.Action {
		executor := newThroughputStressExecutor()
		require.NoError(t, executor.Configure(info))
		executor.config.FiniteScheduleRootPercentage = percentage
		return executor.createActions(info.NewRun(1))[0].Actions
	}
	without, with := generate(0), generate(100)
	for chunk := 0; chunk < 4; chunk++ {
		require.Equal(t, directPayloadSizes(without), directPayloadSizes(with), "schedule generation must not consume workload RNG")
		if chunk < 3 {
			without = decodeContinueAsNewChunk(t, without)
			with = decodeContinueAsNewChunk(t, with)
		}
	}
}

func TestThroughputFiniteScheduleNativeGeneration(t *testing.T) {
	env := workertest.SetupTestEnvironment(t, workertest.WithExecutorTimeout(2*time.Minute))
	info := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("finite-native-%d", time.Now().UnixNano()), ExecutionID: "finite-execution",
		Configuration: loadgen.RunConfiguration{Iterations: 1},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag: "2", ContinueAsNewAfterIterFlag: "1", SleepTimeFlag: "1ms",
			NexusEnabledFlag: "false", IncludeStandaloneActivityFlag: "false",
			IncludeStandaloneActivityOperatorCommandsFlag: "false",
		}),
	}
	var checked int
	execute := &loadgen.GenericExecutor{Execute: func(ctx context.Context, run *loadgen.Run) (retErr error) {
		native := newThroughputStressExecutor()
		if err := native.Configure(*run.ScenarioInfo); err != nil {
			return err
		}
		native.config.FiniteScheduleRootPercentage = 100
		rootID := run.DefaultStartWorkflowOptions().ID
		ids := []string{finiteScheduleID(rootID, 0), finiteScheduleID(rootID, 1)}
		defer func() {
			for _, id := range ids {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err := ks.DeleteFiniteSchedule(cleanupCtx, run.Client.ScheduleClient(), ks.FiniteScheduleInput{ScheduleID: id}, run.TaskQueue())
				cancel()
				if err != nil {
					retErr = errors.Join(retErr, fmt.Errorf("cleaning generated schedule %s: %w", id, err))
				}
			}
		}()
		if err := run.ExecuteKitchenSinkWorkflow(ctx, &loadgen.KitchenSinkWorkflowOptions{
			StartOptions: run.DefaultStartWorkflowOptions(),
			Params: &ks.TestInput{WorkflowInput: &ks.WorkflowInput{
				InitialActions: native.createActions(run),
			}},
		}); err != nil {
			return err
		}
		verifyCtx, cancel := context.WithTimeout(ctx, 65*time.Second)
		defer cancel()
		for _, id := range ids {
			if err := ks.VerifyFiniteSchedule(verifyCtx, run.Client, ks.FiniteScheduleInput{ScheduleID: id}, run.TaskQueue()); err != nil {
				return err
			}
			checked += 4
		}
		count, err := run.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
			Namespace: run.Namespace,
			Query:     fmt.Sprintf("%s='%s'", loadgen.OmesExecutionIDSearchAttribute, run.ExecutionID),
		})
		if err != nil {
			return err
		}
		if count.Count != 4 {
			return fmt.Errorf("parent/child/CAN count changed: expected4, got%d", count.Count)
		}
		return nil
	}}
	_, err := env.RunExecutorTest(t, execute, info, clioptions.LangGo)
	require.NoError(t, err)
	require.Equal(t, 8, checked)
}
