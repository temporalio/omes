package scenarios

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestThroughputStressPayloadHeartbeats(t *testing.T) {
	for _, tc := range []struct {
		name    string
		option  string
		timeout time.Duration
	}{
		{name: "default unchanged"},
		{name: "explicit zero unchanged", option: "0"},
		{name: "initial heartbeat enabled", option: "10s", timeout: 10 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := map[string]string{
				IterFlag: "1", NexusEnabledFlag: "false", IncludeStandaloneActivityFlag: "true",
			}
			if tc.option != "" {
				options[RemotePayloadHeartbeatTimeoutFlag] = tc.option
			}
			info := loadgen.ScenarioInfo{
				RunID: "heartbeat-contract", ExecutionID: "exec", Logger: zap.NewNop().Sugar(),
				Options: loadgen.MustResolveScenarioOptions("throughput_stress", options),
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
			var remote, local int
			var inspect func(protoreflect.Message)
			inspect = func(message protoreflect.Message) {
				if child, ok := message.Interface().(*ks.ExecuteChildWorkflowAction); ok {
					var input ks.WorkflowInput
					require.NoError(t, conv.FromPayload(child.GetInput()[0], &input))
					inspect(input.ProtoReflect())
				}
				if activity, ok := message.Interface().(*ks.ExecuteActivityAction); ok && activity.GetPayload() != nil {
					if activity.GetIsLocal() != nil {
						local++
						require.Nil(t, activity.HeartbeatTimeout, "local payload must not heartbeat")
					} else {
						remote++
						require.Equal(t, tc.timeout, activity.GetHeartbeatTimeout().AsDuration())
						if tc.timeout == 0 {
							require.Nil(t, activity.HeartbeatTimeout, "disabled metadata stays absent")
						}
					}
				}
				message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
					if field.Kind() == protoreflect.MessageKind && !field.IsMap() {
						if field.IsList() {
							for i := 0; i < value.List().Len(); i++ {
								inspect(value.List().Get(i).Message())
							}
						} else {
							inspect(value.Message())
						}
					}
					return true
				})
			}
			inspect(decoded.ProtoReflect())
			require.Equal(t, 7, remote, "two root, three child, one update and one standalone payload")
			require.Equal(t, 6, local, "local payload shape remains unchanged")
			nexusActivity := executor.createNexusStandaloneActivityAction().GetNexusOperation().GetInput().GetStartActivity()
			require.IsType(t, &ks.ExecuteActivityAction_Noop{}, nexusActivity.ActivityType, "Nexus standalone is a noop, not a payload")
			require.Nil(t, nexusActivity.HeartbeatTimeout)
		})
	}
}

func TestThroughputStressRejectsNegativePayloadHeartbeatTimeout(t *testing.T) {
	executor := newThroughputStressExecutor()
	err := executor.Configure(loadgen.ScenarioInfo{
		Options: loadgen.MustResolveScenarioOptions("throughput_stress", map[string]string{
			RemotePayloadHeartbeatTimeoutFlag: "-1ns",
		}),
	})
	require.ErrorContains(t, err, RemotePayloadHeartbeatTimeoutFlag+" must be non-negative")
}
