package devserver

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func envMap(env []string) map[string]string {
	out := map[string]string{}
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok {
			out[k] = v
		}
	}
	return out
}

func TestBuildServerEnv_SQLite(t *testing.T) {
	ports := [portCount]int{7233, 6933, 7243, 7234, 6934, 7235, 6935, 7239, 6939}
	env := buildServerEnv("/tmp/work", PersistenceOptions{}, ClusterEndpoint{}, "/tmp/dyn.yaml", "127.0.0.1", ports)
	m := envMap(env)

	require.Equal(t, "sqlite", m["DB"])
	require.Equal(t, "/tmp/work/temporal_default.db", m["DBNAME"])
	require.Equal(t, "/tmp/work/temporal_visibility.db", m["VISIBILITY_DBNAME"])
	require.Equal(t, "127.0.0.1", m["BIND_ON_IP"])
	require.Equal(t, "7233", m["FRONTEND_GRPC_PORT"])
	require.Equal(t, "7243", m["FRONTEND_HTTP_PORT"])
	require.Equal(t, "/tmp/dyn.yaml", m["DYNAMIC_CONFIG_FILE_PATH"])
	require.NotContains(t, m, "CLUSTER_RPC_ADDRESS") // default kicks in
}

func TestBuildServerEnv_Postgres(t *testing.T) {
	var ports [portCount]int
	env := buildServerEnv("/tmp/work", PersistenceOptions{
		Driver:      "postgres12",
		ConnectAddr: "10.0.0.5:5432",
		User:        "temporal",
		Password:    "secret",
	}, ClusterEndpoint{}, "/tmp/dyn.yaml", "127.0.0.1", ports)
	m := envMap(env)

	require.Equal(t, "postgres12", m["DB"])
	require.Equal(t, "10.0.0.5", m["POSTGRES_SEEDS"])
	require.Equal(t, "5432", m["DB_PORT"])
	require.Equal(t, "temporal", m["POSTGRES_USER"])
	require.Equal(t, "secret", m["POSTGRES_PWD"])
}

func TestBuildServerEnv_ClusterOverride(t *testing.T) {
	var ports [portCount]int
	ports[portFrontendGRPC] = 1234
	ports[portFrontendHTTP] = 1235
	env := buildServerEnv("/tmp/work", PersistenceOptions{}, ClusterEndpoint{
		RPCAddress:  "10.0.0.9:9999",
		HTTPAddress: "10.0.0.9:9998",
	}, "/tmp/dyn.yaml", "127.0.0.1", ports)
	m := envMap(env)
	require.Equal(t, "10.0.0.9:9999", m["CLUSTER_RPC_ADDRESS"])
	require.Equal(t, "10.0.0.9:9998", m["CLUSTER_HTTP_ADDRESS"])
}

func TestAllocatePorts(t *testing.T) {
	ports, err := allocatePorts("127.0.0.1")
	require.NoError(t, err)
	require.GreaterOrEqual(t, ports[0], safePortMin)
	require.Less(t, ports[portCount-1], 32768)
	for i := 1; i < portCount; i++ {
		require.Equal(t, ports[0]+i, ports[i], "ports must be consecutive")
	}
}

func TestWriteDynamicConfig_Overrides(t *testing.T) {
	tmp := t.TempDir()
	path, err := writeDynamicConfig(tmp, "127.0.0.1:7243", map[string]any{
		"nexusoperation.enableStandalone": true,
		"my.numeric.setting":              42,
		"my.string.setting":               "hello",
	})
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	body := string(data)
	require.Contains(t, body, "system.forceSearchAttributesCacheRefreshOnRead")
	require.Contains(t, body, "component.nexusoperations.callback.endpoint.template")
	require.Contains(t, body, "127.0.0.1:7243/namespaces/{{.NamespaceName}}/nexus/callback")
	require.Contains(t, body, "nexusoperation.enableStandalone:\n- value: true")
	require.Contains(t, body, "my.numeric.setting:\n- value: 42")
	require.Contains(t, body, "my.string.setting:\n- value: hello")
}
