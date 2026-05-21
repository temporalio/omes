package devserver

import (
	"bytes"
	"cmp"
	"fmt"
	"maps"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"

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

// Postgres stores cluster_membership.rpc_port as SMALLINT (max 32767), so
// every port must stay below that. Linux ephemeral ports start at 32768
// which would overflow — pick from a fixed safe window instead.
const (
	safePortMin = 10000
	safePortMax = 30000
)

const maxPortAllocationAttempts = 50

// allocatePorts returns portCount free ports in the postgres-safe
// range (<32768). It binds each port to prove it's free, then releases the
// listeners before returning so the caller can rebind.
func allocatePorts(host string) ([portCount]int, error) {
	var ports [portCount]int
	used := make(map[int]struct{}, portCount)
	listeners := make([]net.Listener, 0, portCount)
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()

	for i := range portCount {
		port, l, err := tryAllocatePort(host, used)
		if err != nil {
			return [portCount]int{},
				fmt.Errorf("allocatePorts: no free port in [%d, %d) after %d attempts: %w", safePortMin, safePortMax, maxPortAllocationAttempts, err)
		}
		listeners = append(listeners, l)
		ports[i] = port
		used[port] = struct{}{}
	}
	return ports, nil
}

func tryAllocatePort(host string, used map[int]struct{}) (int, net.Listener, error) {
	var lastErr error
	for range maxPortAllocationAttempts {
		port := safePortMin + rand.Intn(safePortMax-safePortMin)
		if _, ok := used[port]; ok {
			continue
		}

		l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
		if err != nil {
			lastErr = err
			continue
		}
		return port, l, nil
	}
	return 0, nil, cmp.Or(lastErr, fmt.Errorf("no unused free port found"))
}

// buildServerEnv returns the env vars that drive temporal-server's embedded
// config template (common/config/config_template_embedded.yaml). Inherits the
// current process env so PATH/TMPDIR etc. work, then overrides with the
// specific variables that map our Options into the template's env keys. Errors
// on unsupported drivers so Start surfaces misconfiguration before the
// expensive clone+build.
func buildServerEnv(p PersistenceOptions, c ClusterEndpoint, dynConfigPath, host string, ports [portCount]int) ([]string, error) {
	env := slices.Clone(os.Environ())
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

	// Active cluster config.
	if c.RPCAddress != "" {
		add("CLUSTER_RPC_ADDRESS", c.RPCAddress)
	}
	if c.HTTPAddress != "" {
		add("CLUSTER_HTTP_ADDRESS", c.HTTPAddress)
	}

	// Persistence config. Let the server validate unsupported drivers.
	driver := cmp.Or(p.Driver, "sqlite")
	add("DB", driver)
	if driver == "sqlite" {
		// Use in-memory SQLite.
		add("SQLITE_MODE", "memory")
		add("SQLITE_CACHE", "shared")
	}
	return env, nil
}

// dynamicConfigDefaults mirrors the dynamic-config the Temporal CLI dev-server
// applies plus the SDK testsuite's server-version-check disable.
func dynamicConfigDefaults(frontendHTTPAddr string) map[string]any {
	return map[string]any{
		// Match the Temporal CLI dev-server's higher limits.
		"history.persistenceMaxQPS":  45000,
		"frontend.persistenceMaxQPS": 10000,
		// Allows SDK/server minor version skew.
		"frontend.enableServerVersionCheck": false,
		// Older server versions fail Nexus tasks when the callback URL template is unset.
		"component.nexusoperations.callback.endpoint.template": fmt.Sprintf("http://%s/namespaces/{{.NamespaceName}}/nexus/callback", frontendHTTPAddr),
		"component.nexusoperations.useSystemCallbackURL":       true,
		// Eliminate cache races.
		"system.forceSearchAttributesCacheRefreshOnRead": true,
		"system.forceNexusEndpointRefreshOnRead":         true,
	}
}

type dynamicConfigEntry struct {
	Value       any            `yaml:"value"`
	Constraints map[string]any `yaml:"constraints"`
}

func writeDynamicConfig(workDir, frontendHTTPAddr string, overrides map[string]any) (string, error) {
	cfg := dynamicConfigDefaults(frontendHTTPAddr)
	maps.Copy(cfg, overrides)

	// Emit stable, readable YAML while letting yaml.Marshal handle arbitrary
	// scalar, slice, or map values from callers.
	var buf bytes.Buffer
	for _, k := range slices.Sorted(maps.Keys(cfg)) {
		out, err := yaml.Marshal(map[string][]dynamicConfigEntry{
			k: {{Value: cfg[k], Constraints: map[string]any{}}},
		})
		if err != nil {
			return "", fmt.Errorf("marshal dynamic-config %s: %w", k, err)
		}
		buf.Write(out)
	}

	path := filepath.Join(workDir, "dynamicconfig.yaml")
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return "", err
	}
	return path, nil
}
