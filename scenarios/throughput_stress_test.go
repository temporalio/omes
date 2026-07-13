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
		ScenarioOptions: map[string]string{
			IterFlag:                          "2",
			ContinueAsNewAfterIterFlag:        "1",
			NexusEndpointFlag:                 env.NexusEndpointName(),
			SleepTimeFlag:                     "1ms", // reduce to safe time
			VisibilityVerificationTimeoutFlag: "10s", // lower timeout to fail fast
		},
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

func TestThroughputStressConfigurePayload(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID: "tps-payload",
		ScenarioOptions: map[string]string{
			PayloadDistributionJsonFlag: `{"size":{"type":"discrete","weights":{"1024":1}}}`,
		},
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
	require.NoError(t, executor.Configure(loadgen.ScenarioInfo{RunID: "tps-no-payload"}))
	require.Nil(t, executor.config.Payload)
	require.Equal(t, 256, executor.samplePayloadSize(rand.New(rand.NewSource(1))))
}

func TestThroughputStressConfigureInvalidPayload(t *testing.T) {
	t.Parallel()

	executor := newThroughputStressExecutor()
	info := loadgen.ScenarioInfo{
		RunID: "tps-payload-invalid",
		ScenarioOptions: map[string]string{
			PayloadDistributionJsonFlag: `{"size":{"type":"bogus"}}`,
		},
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
		ScenarioOptions: map[string]string{
			IterFlag:                    "4", // > ContinueAsNewAfterIter, so chunks are nested via CAN
			ContinueAsNewAfterIterFlag:  "1",
			PayloadDistributionJsonFlag: `{"size":{"type":"uniform","min":"1","max":"1000000"}}`,
		},
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
