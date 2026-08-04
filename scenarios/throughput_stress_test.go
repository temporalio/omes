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
// end to end. The dev server runs with the standalone-activity and standalone-Nexus
// gates on, and the scenario passes neither feature option, so both have to resolve
// from what the namespace reports.
func TestThroughputStressFeatureAutoEnable(t *testing.T) {
	t.Parallel()

	runID := fmt.Sprintf("tps-feature-%d", time.Now().Unix())

	// No WithNexusEndpoint here: if standalone Nexus auto-enables, the scenario
	// enables Nexus itself and creates its own endpoint, which is the path under
	// test. Pre-creating one would collide on the same NexusEndpointForRun name.
	env := workertest.SetupTestEnvironment(t,
		workertest.WithExecutorTimeout(1*time.Minute),
		workertest.WithDynamicConfig(map[string]any{
			// Gates the capabilities the two feature options read.
			"activity.enableStandalone":       true,
			"nexusoperation.enableStandalone": true,
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
		}),
	}

	executor := newThroughputStressExecutor()
	_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	require.True(t, executor.config.IncludeStandaloneActivity,
		"standalone activities should auto-enable from the namespace capability")
	// Standalone Nexus follows whatever this server version reports. When it is on,
	// the scenario turns nexus-enabled on by itself.
	require.Equal(t, caps.GetStandaloneNexusOperation(), executor.config.IncludeStandaloneNexus)
	if caps.GetStandaloneNexusOperation() {
		require.True(t, executor.config.NexusEnabled,
			"auto-enabled standalone Nexus should also enable Nexus")
	}
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

// TestThroughputStressConfigureFeatureAutoEnablesNexus covers the interaction
// added for capability-gated feature options: when include-standalone-nexus
// resolves to true (as it would after ResolveFeatureOptions probes a capable
// namespace) and nexus-enabled was never touched by the user, Configure turns
// Nexus on rather than failing on an option the user never set.
//
// ResolveFeatureOptions itself needs a dialed client, so this simulates its
// outcome directly on the resolved option set with Set, exactly as
// resolveFeatures would for an unset feature option.
func TestThroughputStressConfigureFeatureAutoEnablesNexus(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:  "tps-feature-auto",
		Logger: zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			SleepActivityJsonFlag: "", // keep defaults minimal
		}),
	}
	require.NoError(t, info.Options.Set(IncludeStandaloneNexusFlag, "true"))

	require.NoError(t, executor.Configure(info))
	require.True(t, executor.config.NexusEnabled)
	require.True(t, executor.config.IncludeStandaloneNexus)
}

// TestThroughputStressConfigureFeatureRespectsExplicitNexusDisabled covers the
// other half: an explicit nexus-enabled=false must not be silently overridden
// even when the feature auto-enables standalone Nexus.
func TestThroughputStressConfigureFeatureRespectsExplicitNexusDisabled(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID:  "tps-feature-explicit-off",
		Logger: zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			NexusEnabledFlag: "false",
		}),
	}
	require.NoError(t, info.Options.Set(IncludeStandaloneNexusFlag, "true"))

	err := executor.Configure(info)
	require.Error(t, err)
	require.Contains(t, err.Error(), IncludeStandaloneNexusFlag)
	require.Contains(t, err.Error(), NexusEnabledFlag)
}

// TestThroughputStressConfigureExplicitStandaloneNexusRequiresNexusEnabled
// preserves the pre-existing behavior: a user who explicitly asks for standalone
// Nexus while explicitly disabling Nexus gets an error either way.
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
