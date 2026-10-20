package scenarios

import (
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
)

func childCountExecutor(t *testing.T, options map[string]string) (*tpsExecutor, *loadgen.Run) {
	t.Helper()
	values := map[string]string{
		NexusEnabledFlag:                              "false",
		IncludeStandaloneNexusFlag:                    "false",
		IncludeStandaloneActivityFlag:                 "false",
		IncludeStandaloneActivityOperatorCommandsFlag: "false",
		PayloadDistributionJsonFlag:                   `{"size":{"type":"fixed","value":0}}`,
		ChildPayloadDistributionJsonFlag:              `{"size":{"type":"fixed","value":17}}`,
		RemotePayloadHeartbeatTimeoutFlag:             "10s",
	}
	for key, value := range options {
		values[key] = value
	}
	info := loadgen.ScenarioInfo{
		RunID: "child-count", ExecutionID: "child-count-exec", Logger: zap.NewNop().Sugar(),
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", values),
	}
	executor := newThroughputStressExecutor()
	require.NoError(t, executor.Configure(info))
	return executor, info.NewRun(1)
}

func serializedChildActivityCount(t *testing.T, action *ks.Action) int {
	t.Helper()
	child := action.GetExecChildWorkflow()
	require.NotNil(t, child)
	var input ks.WorkflowInput
	require.NoError(t, converter.NewProtoJSONPayloadConverter().FromPayload(child.Input[0], &input))
	require.Len(t, input.InitialActions, 1)
	set := input.InitialActions[0]
	require.False(t, set.Concurrent)
	require.NotEmpty(t, set.Actions)
	require.NotNil(t, set.Actions[len(set.Actions)-1].GetReturnResult())
	for _, action := range set.Actions[:len(set.Actions)-1] {
		activity := action.GetExecActivity()
		require.NotNil(t, activity)
		require.NotNil(t, activity.GetRemote())
		require.NotNil(t, activity.GetPayload())
		require.EqualValues(t, 17, activity.GetPayload().BytesToReceive)
		require.EqualValues(t, 17, activity.GetPayload().BytesToReturn)
		require.EqualValues(t, 10, activity.HeartbeatTimeout.Seconds)
	}
	return len(set.Actions) - 1
}

func TestThroughputStressChildActivityFixedCount(t *testing.T) {
	for _, count := range []int{0, 1, 1000} {
		name := strconv.Itoa(count)
		if count == 0 {
			name = "default"
		}
		t.Run(name, func(t *testing.T) {
			options := map[string]string{}
			want := 3
			if count != 0 {
				want = count
				options[ChildActivityCountMinFlag] = strconv.Itoa(count)
				options[ChildActivityCountMaxFlag] = strconv.Itoa(count)
			}
			executor, run := childCountExecutor(t, options)
			rng := rand.New(rand.NewSource(37))
			wantRng := rand.New(rand.NewSource(37))
			require.Equal(t, want, serializedChildActivityCount(t, executor.createChildWorkflowAction(run, 1, rng)))
			require.Equal(t, wantRng.Int63(), rng.Int63(), "fixed counts and fixed payloads must not consume random draws")
		})
	}
}

func TestThroughputStressChildActivityUniformCount(t *testing.T) {
	executor, run := childCountExecutor(t, map[string]string{ChildActivityCountMinFlag: "1", ChildActivityCountMaxFlag: "4"})
	rng := rand.New(rand.NewSource(73))
	wantRng := rand.New(rand.NewSource(73))
	seen := map[int]bool{}
	for i := range 64 {
		count := serializedChildActivityCount(t, executor.createChildWorkflowAction(run, i, rng))
		require.Equal(t, 1+wantRng.Intn(4), count)
		seen[count] = true
	}
	require.Len(t, seen, 4)
}

func TestThroughputStressRejectsInvalidChildActivityRange(t *testing.T) {
	for _, bounds := range [][2]int{{0, 3}, {-1, 3}, {3, 0}, {5, 4}, {1, 1001}} {
		t.Run(strconv.Itoa(bounds[0])+"-"+strconv.Itoa(bounds[1]), func(t *testing.T) {
			executor := newThroughputStressExecutor()
			err := executor.Configure(loadgen.ScenarioInfo{
				Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
					NexusEnabledFlag:          "false",
					ChildActivityCountMinFlag: strconv.Itoa(bounds[0]),
					ChildActivityCountMaxFlag: strconv.Itoa(bounds[1]),
				}),
			})
			require.Error(t, err)
			require.ErrorContains(t, err, ChildActivityCountMinFlag+"/"+ChildActivityCountMaxFlag)
		})
	}
}
