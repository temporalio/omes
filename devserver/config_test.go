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
	ports := portSet{
		Host:               "127.0.0.1",
		FrontendGRPC:       7233,
		FrontendMembership: 6933,
		FrontendHTTP:       7243,
		HistoryGRPC:        7234,
		HistoryMembership:  6934,
		MatchingGRPC:       7235,
		MatchingMembership: 6935,
		WorkerGRPC:         7239,
		WorkerMembership:   6939,
	}
	env := buildServerEnv("/tmp/work", PersistenceOptions{}, ClusterEndpoint{}, "/tmp/dyn.yaml", ports)
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
	env := buildServerEnv("/tmp/work", PersistenceOptions{
		Driver:      "postgres12",
		ConnectAddr: "10.0.0.5:5432",
		User:        "temporal",
		Password:    "secret",
	}, ClusterEndpoint{}, "/tmp/dyn.yaml", portSet{Host: "127.0.0.1"})
	m := envMap(env)

	require.Equal(t, "postgres12", m["DB"])
	require.Equal(t, "10.0.0.5", m["POSTGRES_SEEDS"])
	require.Equal(t, "5432", m["DB_PORT"])
	require.Equal(t, "temporal", m["POSTGRES_USER"])
	require.Equal(t, "secret", m["POSTGRES_PWD"])
}

func TestBuildServerEnv_ClusterOverride(t *testing.T) {
	env := buildServerEnv("/tmp/work", PersistenceOptions{}, ClusterEndpoint{
		RPCAddress:  "10.0.0.9:9999",
		HTTPAddress: "10.0.0.9:9998",
	}, "/tmp/dyn.yaml", portSet{Host: "127.0.0.1", FrontendGRPC: 1234, FrontendHTTP: 1235})
	m := envMap(env)
	require.Equal(t, "10.0.0.9:9999", m["CLUSTER_RPC_ADDRESS"])
	require.Equal(t, "10.0.0.9:9998", m["CLUSTER_HTTP_ADDRESS"])
}

func TestAllocatePortSet_PortBase(t *testing.T) {
	p, err := allocatePortSet("", 7230)
	require.NoError(t, err)
	require.Equal(t, 7230, p.FrontendGRPC)
	require.Equal(t, 7231, p.FrontendMembership)
	require.Equal(t, 7232, p.FrontendHTTP)
	require.Equal(t, 7238, p.WorkerMembership)
}

func TestAllocatePortSet_PortBaseConflictsWithBindPort(t *testing.T) {
	_, err := allocatePortSet("127.0.0.1:9999", 7230)
	require.Error(t, err)
}

func TestWriteDynamicConfig_Overrides(t *testing.T) {
	tmp := t.TempDir()
	ports := portSet{Host: "127.0.0.1", FrontendHTTP: 7243}
	path, err := writeDynamicConfig(tmp, map[string]any{
		"nexusoperation.enableStandalone": true,
		"my.numeric.setting":              42,
		"my.string.setting":               "hello",
	}, ports)
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

func TestSanitizeRef(t *testing.T) {
	require.Equal(t, "abc123", sanitizeRef("abc123"))
	require.Equal(t, "release_v1.30.0", sanitizeRef("release/v1.30.0"))
}
