package devserver

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// buildServerEnv returns the env vars that drive temporal-server's embedded
// config template (common/config/config_template_embedded.yaml). Inherits the
// current process env so PATH/TMPDIR etc. work, then overrides with the
// specific variables that map our Options into the template's env keys.
func buildServerEnv(workDir string, p PersistenceOptions, cluster ClusterEndpoint, dynConfigPath string, ports portSet) []string {
	env := append([]string(nil), os.Environ()...)
	add := func(k, v string) { env = append(env, k+"="+v) }

	add("BIND_ON_IP", ports.Host)
	add("NUM_HISTORY_SHARDS", "4")
	add("LOG_LEVEL", "info")
	add("DYNAMIC_CONFIG_FILE_PATH", dynConfigPath)

	// Service ports.
	add("FRONTEND_GRPC_PORT", strconv.Itoa(ports.FrontendGRPC))
	add("FRONTEND_MEMBERSHIP_PORT", strconv.Itoa(ports.FrontendMembership))
	add("FRONTEND_HTTP_PORT", strconv.Itoa(ports.FrontendHTTP))
	add("HISTORY_GRPC_PORT", strconv.Itoa(ports.HistoryGRPC))
	add("HISTORY_MEMBERSHIP_PORT", strconv.Itoa(ports.HistoryMembership))
	add("MATCHING_GRPC_PORT", strconv.Itoa(ports.MatchingGRPC))
	add("MATCHING_MEMBERSHIP_PORT", strconv.Itoa(ports.MatchingMembership))
	add("WORKER_GRPC_PORT", strconv.Itoa(ports.WorkerGRPC))
	add("WORKER_MEMBERSHIP_PORT", strconv.Itoa(ports.WorkerMembership))

	// Active cluster pointer. Empty values let the template's default kick in
	// (127.0.0.1:<grpc/http>).
	if cluster.RPCAddress != "" {
		add("CLUSTER_RPC_ADDRESS", cluster.RPCAddress)
	}
	if cluster.HTTPAddress != "" {
		add("CLUSTER_HTTP_ADDRESS", cluster.HTTPAddress)
	}

	// Persistence. Driver maps to the template's $db switch.
	driver := p.driverOrDefault()
	add("DB", driver)
	switch driver {
	case "sqlite":
		add("DBNAME", filepath.Join(workDir, "temporal_default.db"))
		add("VISIBILITY_DBNAME", filepath.Join(workDir, "temporal_visibility.db"))
	case "postgres12", "postgres12_pgx":
		host, port := splitHostPort(p.ConnectAddr, "5432")
		add("DBNAME", "temporal")
		add("VISIBILITY_DBNAME", "temporal_visibility")
		add("POSTGRES_SEEDS", host)
		add("DB_PORT", port)
		add("POSTGRES_USER", p.User)
		add("POSTGRES_PWD", p.Password)
	case "mysql8":
		host, port := splitHostPort(p.ConnectAddr, "3306")
		add("DBNAME", "temporal")
		add("VISIBILITY_DBNAME", "temporal_visibility")
		add("MYSQL_SEEDS", host)
		add("DB_PORT", port)
		add("MYSQL_USER", p.User)
		add("MYSQL_PWD", p.Password)
	}
	return env
}

// splitHostPort parses "host:port" / "host" / "" and falls back to defaultPort
// when no port is present.
func splitHostPort(addr, defaultPort string) (string, string) {
	if addr == "" {
		return "", defaultPort
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, defaultPort
	}
	if port == "" {
		port = defaultPort
	}
	return host, port
}

// dynamicConfigDefaultsTemplate mirrors the dynamic-config the Temporal CLI
// dev-server applies (see github.com/temporalio/cli
// internal/temporalcli/commands.server.go) plus the SDK testsuite's
// server-version-check disable. Without these:
//   - freshly AddSearchAttributes'd attributes race the cluster-metadata
//     cache and produce "Namespace X has no mapping defined for search
//     attribute Y" on workflow start;
//   - the SDK rejects servers with minor version skew;
//   - older server versions (pre useSystemCallbackURL=true default) fail
//     Nexus tasks because the callback URL template is "unset".
const dynamicConfigDefaultsTemplate = `
system.forceSearchAttributesCacheRefreshOnRead:
- value: true
  constraints: {}
# Since SA cache is disabled, bump persistence QPS to absorb the extra reads.
frontend.persistenceMaxQPS:
- value: 10000
  constraints: {}
history.persistenceMaxQPS:
- value: 45000
  constraints: {}
frontend.enableServerVersionCheck:
- value: false
  constraints: {}
component.nexusoperations.callback.endpoint.template:
- value: http://%s/namespaces/{{.NamespaceName}}/nexus/callback
  constraints: {}
# Use "temporal://system" callback URL for worker-target Nexus (default in
# current main; false in pre-1.31 releases). Without this, older release
# servers emit the template URL above which the worker SDK rejects unless
# an allowed-addresses list is also configured.
component.nexusoperations.useSystemCallbackURL:
- value: true
  constraints: {}
`

func writeDynamicConfig(workDir string, overrides map[string]any, ports portSet) (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, dynamicConfigDefaultsTemplate, net.JoinHostPort(ports.Host, strconv.Itoa(ports.FrontendHTTP)))
	keys := make([]string, 0, len(overrides))
	for k := range overrides {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val, err := yaml.Marshal(overrides[k])
		if err != nil {
			return "", fmt.Errorf("marshal dynamic-config %s: %w", k, err)
		}
		fmt.Fprintf(&buf, "%s:\n- value: %s  constraints: {}\n", k, strings.TrimRight(string(val), "\n")+"\n")
	}

	path := filepath.Join(workDir, "dynamicconfig.yaml")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return "", err
	}
	return path, nil
}
