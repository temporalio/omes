package scenarios

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/internal/workertest"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	namespacev1 "go.temporal.io/api/namespace/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
)

func TestThroughputStress(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-%d", time.Now().Unix())

	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute),
		workertest.WithNexusEndpoint(runID))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: runID,
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag:                          "2",
			ContinueAsNewAfterIterFlag:        "1",
			NexusEnabledFlag:                  "true",
			NexusEndpointFlag:                 env.NexusEndpointName(),
			SleepTimeFlag:                     "1ms", // reduce to safe time
			VisibilityVerificationTimeoutFlag: "10s", // lower timeout to fail fast
		}),
	}

	t.Run("Run executor", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully")

		state := executor.Snapshot().(tpsState)
		require.Equal(t, state.CompletedIterations, 2)
	})

	t.Run("Run executor again, resuming from middle", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		err := executor.LoadState(func(v any) error {
			s := v.(*tpsState)
			s.CompletedIterations = 0 // execution will start from iteration 1
			return nil
		})
		require.NoError(t, err)

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully when resuming from middle")
	})

	t.Run("Run executor again, resuming from end", func(t *testing.T) {
		executor := newThroughputStressExecutor()

		err := executor.LoadState(func(v any) error {
			s := v.(*tpsState)
			s.CompletedIterations = s.CompletedIterations
			return nil
		})
		require.NoError(t, err)

		_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
		require.NoError(t, err, "Executor should complete successfully when resuming from end")
	})
}

// TestThroughputStressFeatureAutoEnable exercises capability-gated feature options
// end to end. The dev server runs with the standalone-activity, operator-command,
// batch-operation, and standalone-Nexus gates on, and the scenario leaves those feature options
// unset, so they have to resolve from what the namespace reports.
func TestThroughputStressFeatureAutoEnable(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-feature-%d", time.Now().Unix())

	// No WithNexusEndpoint here: the scenario creates its own endpoint for the run,
	// which is part of the path under test. Pre-creating one would collide on the
	// same NexusEndpointForRun name.
	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute),
		workertest.WithDynamicConfig(map[string]any{
			// Gates the capabilities the feature options read.
			"activity.enableStandalone":                             true,
			"history.enableStandaloneActivityOperatorCommands":      true,
			"frontend.enableBatchOperationsForStandaloneActivities": true,
			"nexusoperation.enableStandalone":                       true,
			// Standalone Nexus system callbacks require CHASM callbacks.
			"history.enableCHASMCallbacks": true,
		}))

	// Without the namespace advertising standalone activities there is nothing for
	// the feature option to resolve from, and the rest of this test proves nothing.
	desc, err := env.TemporalClient().WorkflowService().DescribeNamespace(t.Context(),
		&workflowservice.DescribeNamespaceRequest{Namespace: "default"})
	require.NoError(t, err)
	caps := desc.GetNamespaceInfo().GetCapabilities()
	require.True(t, caps.GetStandaloneActivities(),
		"namespace should report standalone activities with activity.enableStandalone on")
	require.True(t, caps.GetStandaloneActivityOperatorCommands(),
		"namespace should report standalone activity operator commands with their gate on")
	require.True(t, caps.GetStandaloneActivityBatchOperations(),
		"namespace should report standalone activity batch operations with their gate on")

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: runID,
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag:                          "1",
			ContinueAsNewAfterIterFlag:        "1",
			SleepTimeFlag:                     "1ms",
			VisibilityVerificationTimeoutFlag: "10s",
			// Requesting Nexus load is the user's call; the probe then decides
			// whether standalone operations are part of it.
			NexusEnabledFlag: "true",
		}),
	}

	executor := newThroughputStressExecutor()
	_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	require.True(t, executor.config.IncludeStandaloneActivity,
		"standalone activities should auto-enable from the namespace capability")
	require.True(t, executor.config.IncludeStandaloneActivityOperatorCommands,
		"standalone activity operator commands should auto-enable from the namespace capability")
	require.True(t, executor.config.IncludeStandaloneActivityBatchOperations,
		"standalone activity batch operations should auto-enable from the namespace capability")
	require.Equal(t, 5, executor.config.StandaloneActivityBatchSize)
	// Nexus load was requested, so standalone Nexus follows whatever this server
	// version reports.
	require.True(t, executor.config.NexusEnabled)
	require.Equal(t, caps.GetStandaloneNexusOperation(), executor.config.IncludeStandaloneNexus)
	require.Equal(t, 1, executor.Snapshot().(tpsState).CompletedIterations)
}

func TestThroughputStressConfigurePayload(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID: "tps-payload",
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			PayloadDistributionJsonFlag: `{"size":{"type":"discrete","weights":{"1024":1}}}`,
		}),
	}
	require.NoError(t, executor.Configure(info))
	require.NotNil(t, executor.config.Payload)

	// Single-value distribution always samples the configured value.
	require.Equal(t, 1024, executor.samplePayloadSize(rand.New(rand.NewSource(1))))
}

func TestThroughputStressConfigureNoPayload(t *testing.T) {
	t.Parallel()

	// Without the option, payload sizing falls back to the previous hardcoded 256.
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(loadgen.ScenarioInfo{
		RunID:   "tps-no-payload",
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", nil),
	}))
	require.Nil(t, executor.config.Payload)
	require.Equal(t, 256, executor.samplePayloadSize(rand.New(rand.NewSource(1))))
}

// capableOfStandaloneNexus resolves a throughput_stress option set as if the
// namespace reported standalone Nexus support, so the auto-enabled path can be
// tested without a server. Setting the option directly would instead mark it
// user-specified, which is a different case.
func capableOfStandaloneNexus(t *testing.T, provided map[string]string) *loadgen.OptionSet {
	t.Helper()
	set := loadgen.MustResolveScenarioOptions("throughput_stress", provided)
	require.NoError(t, set.ResolveFeaturesFromCapabilities(loadgen.Capabilities{
		Namespace: &namespacev1.NamespaceInfo_Capabilities{StandaloneNexusOperation: true},
		System:    &workflowservice.GetSystemInfoResponse_Capabilities{Nexus: true},
	}))
	return set
}

// Nexus load is on by default, so the capability probe's answer is what decides
// whether standalone operations are part of it.
func TestThroughputStressConfigureStandaloneNexusAutoEnablesUnderNexus(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		nexus map[string]string
	}{
		{name: "nexus left at its default", nexus: map[string]string{}},
		{name: "nexus explicitly on", nexus: map[string]string{NexusEnabledFlag: "true"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			executor := newThroughputStressExecutor()
			info := loadgen.ScenarioInfo{
				RunID:   "tps-feature-auto",
				Logger:  zap.NewNop().Sugar(),
				Options: capableOfStandaloneNexus(t, tc.nexus),
			}

			require.NoError(t, executor.Configure(info))
			require.True(t, executor.config.NexusEnabled)
			require.True(t, executor.config.IncludeStandaloneNexus,
				"a supported standalone Nexus should be included once Nexus load is on")
		})
	}
}

// Turning Nexus off takes its sub-features with it, rather than failing on a
// sub-feature the user never asked for.
func TestThroughputStressConfigureStandaloneNexusDroppedWithoutNexus(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:   "tps-feature-suppressed",
		Logger:  zap.NewNop().Sugar(),
		Options: capableOfStandaloneNexus(t, map[string]string{NexusEnabledFlag: "false"}),
	}

	require.NoError(t, executor.Configure(info),
		"an auto-enabled sub-feature should not fail a run that asked for no Nexus")
	require.False(t, executor.config.NexusEnabled)
	require.False(t, executor.config.IncludeStandaloneNexus,
		"standalone Nexus should be dropped when there is no Nexus load to add it to")
}

func TestThroughputStressConfigureExplicitStandaloneNexusRequiresNexusEnabled(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:  "tps-explicit-both",
		Logger: zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IncludeStandaloneNexusFlag: "true",
			NexusEnabledFlag:           "false",
		}),
	}

	err := executor.Configure(info)
	require.Error(t, err)
	require.Contains(t, err.Error(), IncludeStandaloneNexusFlag)
	require.Contains(t, err.Error(), NexusEnabledFlag)
}

func TestThroughputStressConfigureInvalidPayload(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID: "tps-payload-invalid",
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			PayloadDistributionJsonFlag: `{"size":{"type":"bogus"}}`,
		}),
	}
	require.Error(t, executor.Configure(info))
}

// TestThroughputStressPayloadSequenceAcrossContinueAsNew guards against the payload-size
// rng restarting at each continue-as-new boundary: the whole nested action tree is built
// client-side from a single per-iteration rng, so a later chunk must continue the sequence
// rather than repeat the first chunk verbatim.
func TestThroughputStressPayloadSequenceAcrossContinueAsNew(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:       "tps-can-seq",
		ExecutionID: "exec",
		Logger:      zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IterFlag:                    "4", // > ContinueAsNewAfterIter, so chunks are nested via CAN
			ContinueAsNewAfterIterFlag:  "1",
			PayloadDistributionJsonFlag: `{"size":{"type":"uniform","min":"1","max":"1000000"}}`,
		}),
	}
	require.NoError(t, executor.Configure(info))

	sets := executor.createActions(info.NewRun(1))
	require.Len(t, sets, 1)

	chunk1 := sets[0].GetActions()
	chunk1Sizes := directPayloadSizes(chunk1)
	require.NotEmpty(t, chunk1Sizes)

	chunk2Sizes := directPayloadSizes(decodeContinueAsNewChunk(t, chunk1))
	require.NotEmpty(t, chunk2Sizes)

	require.NotEqual(t, chunk1Sizes, chunk2Sizes,
		"payload-size sequence must advance across continue-as-new, not restart identically")
}

// TestThroughputStressOperatorCommandsRunEachInternalIterationAcrossContinueAsNew
// guards against losing operator load at a Continue-As-New boundary or restarting
// its command rotation in each new workflow run.
func TestThroughputStressOperatorCommandsRunEachInternalIterationAcrossContinueAsNew(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:       "tps-operator-can",
		ExecutionID: "exec",
		Logger:      zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IncludeStandaloneActivityOperatorCommandsFlag: "true",
			NexusEnabledFlag: "false",
		}),
	}
	require.NoError(t, executor.Configure(info))

	sets := executor.createActions(info.NewRun(1))
	require.Len(t, sets, 1)

	var commandTypes []ks.DoStandaloneActivityOperatorCommands_CommandType
	var commandsPerChunk []int
	chunk := sets[0].GetActions()
	for {
		terminalIndex := -1
		for actionIndex, action := range chunk {
			if action.GetContinueAsNew() != nil || action.GetReturnResult() != nil {
				terminalIndex = actionIndex
				break
			}
		}
		require.NotEqual(t, -1, terminalIndex)

		commands := standaloneActivityOperatorCommandsInConcurrentGroups(chunk[:terminalIndex])
		commandsPerChunk = append(commandsPerChunk, len(commands))
		for _, command := range commands {
			commandTypes = append(commandTypes, command.GetCommandType())
		}

		if chunk[terminalIndex].GetContinueAsNew() == nil {
			break
		}
		chunk = decodeContinueAsNewChunk(t, chunk)
	}

	require.Equal(t, []int{3, 3, 3, 1}, commandsPerChunk)
	require.Equal(t, []ks.DoStandaloneActivityOperatorCommands_CommandType{
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_RESET,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_UPDATE,
		ks.DoStandaloneActivityOperatorCommands_COMMAND_TYPE_PAUSE,
	}, commandTypes)
}

// TestThroughputStressActivityBatchOperationsAcrossContinueAsNew
// guards against losing batch load at a Continue-As-New boundary, restarting its
// operation rotation, or ignoring the configured batch size.
func TestThroughputStressActivityBatchOperationsAcrossContinueAsNew(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:       "tps-activity-batch-can",
		ExecutionID: "exec",
		Logger:      zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			IncludeStandaloneActivityBatchOperationsFlag: "true",
			StandaloneActivityBatchSizeFlag:              "5",
			NexusEnabledFlag:                             "false",
		}),
	}
	require.NoError(t, executor.Configure(info))

	sets := executor.createActions(info.NewRun(1))
	require.Len(t, sets, 1)

	var operationTypes []ks.DoStandaloneActivityBatchOperations_OperationType
	var operationsPerChunk []int
	chunk := sets[0].GetActions()
	for {
		terminalIndex := -1
		for actionIndex, action := range chunk {
			if action.GetContinueAsNew() != nil || action.GetReturnResult() != nil {
				terminalIndex = actionIndex
				break
			}
		}
		require.NotEqual(t, -1, terminalIndex)

		operations := standaloneActivityBatchOperationsInConcurrentGroups(chunk[:terminalIndex])
		operationsPerChunk = append(operationsPerChunk, len(operations))
		for _, operation := range operations {
			operationTypes = append(operationTypes, operation.GetOperationType())
			require.Equal(t, int32(5), operation.GetBatchSize())
		}

		if chunk[terminalIndex].GetContinueAsNew() == nil {
			break
		}
		chunk = decodeContinueAsNewChunk(t, chunk)
	}

	require.Equal(t, []int{3, 3, 3, 1}, operationsPerChunk)
	require.Equal(t, []ks.DoStandaloneActivityBatchOperations_OperationType{
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_TERMINATE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_DELETE,
		ks.DoStandaloneActivityBatchOperations_OPERATION_TYPE_CANCEL,
	}, operationTypes)
}

func standaloneActivityBatchOperationsInConcurrentGroups(actions []*ks.Action) (
	operations []*ks.DoStandaloneActivityBatchOperations,
) {
	for _, action := range actions {
		if set := action.GetNestedActionSet(); set != nil && set.GetConcurrent() {
			for _, nestedAction := range set.GetActions() {
				activity := nestedAction.GetExecActivity()
				if activity == nil || activity.GetClient() == nil {
					continue
				}
				for _, clientSet := range activity.GetClient().GetClientSequence().GetActionSets() {
					for _, clientAction := range clientSet.GetActions() {
						if operation := clientAction.GetDoStandaloneActivityBatchOperations(); operation != nil {
							operations = append(operations, operation)
						}
					}
				}
			}
		}
	}
	return operations
}

func standaloneActivityOperatorCommandsInConcurrentGroups(actions []*ks.Action) (
	commands []*ks.DoStandaloneActivityOperatorCommands,
) {
	for _, action := range actions {
		if set := action.GetNestedActionSet(); set != nil && set.GetConcurrent() {
			commands = append(commands, standaloneActivityOperatorCommands(set.GetActions())...)
		}
	}
	return commands
}

func standaloneActivityOperatorCommands(actions []*ks.Action) (
	commands []*ks.DoStandaloneActivityOperatorCommands,
) {
	for _, action := range actions {
		if command := standaloneActivityOperatorCommand(action); command != nil {
			commands = append(commands, command)
		}
		if set := action.GetNestedActionSet(); set != nil {
			commands = append(commands, standaloneActivityOperatorCommands(set.GetActions())...)
		}
	}
	return commands
}

func standaloneActivityOperatorCommand(action *ks.Action) *ks.DoStandaloneActivityOperatorCommands {
	activity := action.GetExecActivity()
	if activity == nil || activity.GetClient() == nil {
		return nil
	}
	for _, set := range activity.GetClient().GetClientSequence().GetActionSets() {
		for _, clientAction := range set.GetActions() {
			if command := clientAction.GetDoStandaloneActivityOperatorCommands(); command != nil {
				return command
			}
		}
	}
	return nil
}

// directPayloadSizes collects the receive/return byte sizes of payload activities directly
// in the given actions (descending into nested action sets but not into child workflows or
// continue-as-new arguments).
func directPayloadSizes(actions []*ks.Action) []int32 {
	var out []int32
	for _, a := range actions {
		switch v := a.GetVariant().(type) {
		case *ks.Action_ExecActivity:
			if p := v.ExecActivity.GetPayload(); p != nil {
				out = append(out, p.GetBytesToReceive(), p.GetBytesToReturn())
			}
		case *ks.Action_NestedActionSet:
			out = append(out, directPayloadSizes(v.NestedActionSet.GetActions())...)
		}
	}
	return out
}

// decodeContinueAsNewChunk finds the ContinueAsNew action in a chunk and decodes its
// argument back into the next chunk's actions.
func decodeContinueAsNewChunk(t *testing.T, actions []*ks.Action) []*ks.Action {
	t.Helper()
	for _, a := range actions {
		can, ok := a.GetVariant().(*ks.Action_ContinueAsNew)
		if !ok {
			continue
		}
		require.NotEmpty(t, can.ContinueAsNew.GetArguments())
		var input ks.WorkflowInput
		conv := converter.NewProtoJSONPayloadConverter()
		require.NoError(t, conv.FromPayload(can.ContinueAsNew.GetArguments()[0], &input))
		require.NotEmpty(t, input.GetInitialActions())
		return input.GetInitialActions()[0].GetActions()
	}
	t.Fatal("no ContinueAsNew action found in chunk")
	return nil
}
