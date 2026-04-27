package devserver

import (
	"bytes"
	"cmp"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Port slot indices into the slice returned by allocatePorts. Order matches
// the env vars consumed by the temporal server config template.
const (
	portFrontendGRPC = iota
	portFrontendMembership
	portFrontendHTTP
	portHistoryGRPC
	portHistoryMembership
	portMatchingGRPC
	portMatchingMembership
	portWorkerGRPC
	portWorkerMembership
	portCount
)

// buildServerEnv returns the env vars that drive temporal-server's embedded
// config template (common/config/config_template_embedded.yaml). Inherits the
// current process env so PATH/TMPDIR etc. work, then overrides with the
// specific variables that map our Options into the template's env keys.
func buildServerEnv(workDir string, p PersistenceOptions, cluster ClusterEndpoint, dynConfigPath, host string, ports [portCount]int) []string {
	env := append([]string(nil), os.Environ()...)
	add := func(k, v string) { env = append(env, k+"="+v) }

	add("BIND_ON_IP", host)
	add("NUM_HISTORY_SHARDS", "4")
	add("LOG_LEVEL", "info")
	add("DYNAMIC_CONFIG_FILE_PATH", dynConfigPath)

	// Service ports.
	add("FRONTEND_GRPC_PORT", strconv.Itoa(ports[portFrontendGRPC]))
	add("FRONTEND_MEMBERSHIP_PORT", strconv.Itoa(ports[portFrontendMembership]))
	add("FRONTEND_HTTP_PORT", strconv.Itoa(ports[portFrontendHTTP]))
	add("HISTORY_GRPC_PORT", strconv.Itoa(ports[portHistoryGRPC]))
	add("HISTORY_MEMBERSHIP_PORT", strconv.Itoa(ports[portHistoryMembership]))
	add("MATCHING_GRPC_PORT", strconv.Itoa(ports[portMatchingGRPC]))
	add("MATCHING_MEMBERSHIP_PORT", strconv.Itoa(ports[portMatchingMembership]))
	add("WORKER_GRPC_PORT", strconv.Itoa(ports[portWorkerGRPC]))
	add("WORKER_MEMBERSHIP_PORT", strconv.Itoa(ports[portWorkerMembership]))

	// Active cluster pointer. Empty values let the template's default kick in
	// (127.0.0.1:<grpc/http>).
	if cluster.RPCAddress != "" {
		add("CLUSTER_RPC_ADDRESS", cluster.RPCAddress)
	}
	if cluster.HTTPAddress != "" {
		add("CLUSTER_HTTP_ADDRESS", cluster.HTTPAddress)
	}

	// Persistence. Driver maps to the template's $db switch.
	driver := cmp.Or(p.Driver, "sqlite")
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

// dynamicConfigDefaults mirrors the dynamic-config the Temporal CLI dev-server
// applies (see github.com/temporalio/cli internal/temporalcli/commands.server.go)
// plus the SDK testsuite's server-version-check disable. Without these:
//   - freshly AddSearchAttributes'd attributes race the cluster-metadata cache
//     and produce "Namespace X has no mapping defined for search attribute Y"
//     on workflow start;
//   - the SDK rejects servers with minor version skew;
//   - older server versions (pre useSystemCallbackURL=true default) fail Nexus
//     tasks because the callback URL template is "unset".
func dynamicConfigDefaults(frontendHTTPAddr string) map[string]any {
	return map[string]any{
		"system.forceSearchAttributesCacheRefreshOnRead": true,
		// SA cache disabled above, so bump persistence QPS to absorb extra reads.
		"frontend.persistenceMaxQPS":                           10000,
		"history.persistenceMaxQPS":                            45000,
		"frontend.enableServerVersionCheck":                    false,
		"component.nexusoperations.callback.endpoint.template": fmt.Sprintf("http://%s/namespaces/{{.NamespaceName}}/nexus/callback", frontendHTTPAddr),
		// "temporal://system" callback URL for worker-target Nexus (default in
		// current main; false in pre-1.31 releases). Without this, older release
		// servers emit the template URL above which the worker SDK rejects
		// unless an allowed-addresses list is also configured.
		"component.nexusoperations.useSystemCallbackURL": true,
	}
}

func writeDynamicConfig(workDir, frontendHTTPAddr string, overrides map[string]any) (string, error) {
	cfg := dynamicConfigDefaults(frontendHTTPAddr)
	for k, v := range overrides {
		cfg[k] = v
	}

	keys := make([]string, 0, len(cfg))
	for k := range cfg {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		val, err := yaml.Marshal(cfg[k])
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
