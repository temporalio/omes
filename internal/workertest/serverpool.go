package workertest

import (
	"fmt"
	"maps"
	"reflect"
	"sync"
	"testing"

	"github.com/temporalio/omes/devserver"
)

type pooledTestDevServer struct {
	server        *devserver.Server
	dynamicConfig map[string]any
	namespaces    map[string]struct{}
}

type testDevServerGroupKey struct {
	test  *testing.T
	group string
}

type testDevServerPool struct {
	sync.Mutex
	servers map[testDevServerGroupKey]*pooledTestDevServer
}

var groupedTestDevServers = testDevServerPool{
	servers: make(map[testDevServerGroupKey]*pooledTestDevServer),
}

func (p *testDevServerPool) acquire(
	t *testing.T,
	group string,
	namespace string,
	opts devserver.Options,
) (*devserver.Server, error) {
	p.Lock()
	defer p.Unlock()

	key := testDevServerGroupKey{test: t, group: group}
	if pooled := p.servers[key]; pooled != nil {
		if !maps.EqualFunc(pooled.dynamicConfig, opts.DynamicConfigValues, reflect.DeepEqual) {
			return nil, fmt.Errorf(
				"dev server group %q already uses different dynamic config",
				group,
			)
		}
		if _, exists := pooled.namespaces[namespace]; exists {
			return nil, fmt.Errorf(
				"dev server group %q already uses namespace %q",
				group,
				namespace,
			)
		}
		if err := pooled.server.RegisterNamespace(t.Context(), namespace); err != nil {
			return nil, err
		}
		pooled.namespaces[namespace] = struct{}{}
		return pooled.server, nil
	}

	server, err := devserver.Start(t.Context(), opts)
	if err != nil {
		return nil, err
	}
	p.servers[key] = &pooledTestDevServer{
		server:        server,
		dynamicConfig: maps.Clone(opts.DynamicConfigValues),
		namespaces:    map[string]struct{}{namespace: {}},
	}
	t.Cleanup(func() {
		p.Lock()
		delete(p.servers, key)
		p.Unlock()
		_ = server.Stop()
	})
	return server, nil
}
