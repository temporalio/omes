package scenarios

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
)

func TestThroughputStressChildPayloadOverride(t *testing.T) {
	for _, tc := range []struct {
		name       string
		root       string
		child      string
		rootBytes  int32
		childBytes int32
	}{
		{name: "default", rootBytes: 256, childBytes: 256},
		{name: "inherit root", root: `{"size":{"type":"fixed","value":11}}`, rootBytes: 11, childBytes: 11},
		{name: "empty override inherits", root: `{"size":{"type":"fixed","value":11}}`, child: `{}`, rootBytes: 11, childBytes: 11},
		{name: "zero root positive child", root: `{"size":{"type":"fixed","value":0}}`, child: `{"size":{"type":"fixed","value":128}}`, childBytes: 128},
		{name: "explicit zero child", root: `{"size":{"type":"fixed","value":97}}`, child: `{"size":{"type":"fixed","value":0}}`, rootBytes: 97},
	} {
		t.Run(tc.name, func(t *testing.T) {
			info := loadgen.ScenarioInfo{
				RunID: "child-payload-contract", ExecutionID: "exec", Logger: zap.NewNop().Sugar(),
				Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
					IterFlag: "1", NexusEnabledFlag: "false",
					PayloadDistributionJsonFlag:       tc.root,
					"child-payload-distribution-json": tc.child,
				}),
			}
			executor := newThroughputStressExecutor()
			require.NoError(t, executor.Configure(info))
			sets := executor.createActions(info.NewRun(1))
			require.Len(t, sets, 1)
			conv := converter.NewProtoJSONPayloadConverter()
			encoded, err := conv.ToPayload(sets[0])
			require.NoError(t, err)
			var decoded ks.ActionSet
			require.NoError(t, conv.FromPayload(encoded, &decoded))
			rootSizes := directPayloadSizes(decoded.Actions)
			require.NotEmpty(t, rootSizes)
			require.Contains(t, rootSizes, tc.rootBytes)
			for _, size := range rootSizes {
				require.True(t, size == 0 || size == tc.rootBytes, "unexpected root payload size %d", size)
			}
			var children int
			var inspect func([]*ks.Action)
			inspect = func(actions []*ks.Action) {
				for _, action := range actions {
					if child := action.GetExecChildWorkflow(); child != nil {
						children++
						var input ks.WorkflowInput
						require.NoError(t, conv.FromPayload(child.Input[0], &input))
						sizes := directPayloadSizes(input.InitialActions[0].Actions)
						require.Len(t, sizes, 6, "three child activities, each with receive/return size")
						for _, size := range sizes {
							require.Equal(t, tc.childBytes, size)
						}
					}
					if nested := action.GetNestedActionSet(); nested != nil {
						inspect(nested.Actions)
					}
				}
			}
			inspect(decoded.Actions)
			require.Equal(t, 1, children)
		})
	}
}

func TestThroughputStressRejectsInvalidChildPayload(t *testing.T) {
	executor := newThroughputStressExecutor()
	err := executor.Configure(loadgen.ScenarioInfo{
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			"child-payload-distribution-json": `{"size":{"type":"bogus"}}`,
		}),
	})
	require.ErrorContains(t, err, "invalid child-payload-distribution-json")
}

func TestThroughputStressChildPayloadInheritanceSequence(t *testing.T) {
	for _, child := range []string{"", "{}"} {
		executor := newThroughputStressExecutor()
		require.NoError(t, executor.Configure(loadgen.ScenarioInfo{
			Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
				PayloadDistributionJsonFlag:      `{"size":{"type":"uniform","min":1,"max":1000}}`,
				ChildPayloadDistributionJsonFlag: child,
			}),
		}))
		rootRng := rand.New(rand.NewSource(347))
		childRng := rand.New(rand.NewSource(347))
		for range 100 {
			require.Equal(t, executor.samplePayloadSize(rootRng), executor.sampleChildPayloadSize(childRng))
		}
	}
}
