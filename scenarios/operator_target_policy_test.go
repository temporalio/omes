package scenarios

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen"
)

func TestOperatorTargetPolicyPreservesDefaultsAndValidatesOverrides(t *testing.T) {
	for _, scenario := range []string{"throughput_stress", "standalone_activity_operator_commands"} {
		info := loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions(scenario, nil)}
		policy, err := operatorTargetOptions(info)
		require.NoError(t, err)
		require.Equal(t, 5*time.Second, policy.success)
		require.Equal(t, 3*time.Second, policy.heartbeat)
		require.Equal(t, 15*time.Second, policy.startToClose)
		require.Equal(t, int32(2), policy.attempts)
		info.Options = loadgen.MustResolveScenarioOptions(scenario, map[string]string{StandaloneActivityOperatorSuccessDurationFlag: "20s", StandaloneActivityOperatorHeartbeatTimeoutFlag: "10s", StandaloneActivityOperatorStartToCloseTimeoutFlag: "30s", StandaloneActivityOperatorMaxAttemptsFlag: "5"})
		policy, err = operatorTargetOptions(info)
		require.NoError(t, err)
		require.Equal(t, 20*time.Second, policy.success)
		require.Equal(t, 10*time.Second, policy.heartbeat)
		require.Equal(t, 30*time.Second, policy.startToClose)
		require.Equal(t, int32(5), policy.attempts)
	}
	for _, values := range []map[string]string{{StandaloneActivityOperatorHeartbeatTimeoutFlag: "1s"}, {StandaloneActivityOperatorStartToCloseTimeoutFlag: "2s"}, {StandaloneActivityOperatorMaxAttemptsFlag: "0"}, {StandaloneActivityOperatorMaxAttemptsFlag: "2147483648"}} {
		_, err := operatorTargetOptions(loadgen.ScenarioInfo{Options: loadgen.MustResolveScenarioOptions("throughput_stress", values)})
		require.Error(t, err)
	}
}
