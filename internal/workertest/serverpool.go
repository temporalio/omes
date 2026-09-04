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
	mu            sync.Mutex
	start         sync.Once
	server        *devserver.Server
	err           error
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
	key := testDevServerGroupKey{test: t, group: group}

	p.Lock()
	pooled := p.servers[key]
	if pooled == nil {
		pooled = &pooledTestDevServer{}
		p.servers[key] = pooled
	}
	p.Unlock()

	started := false
	pooled.start.Do(func() {
		started = true
		pooled.server, pooled.err = devserver.Start(t.Context(), opts)
		if pooled.err != nil {
			return
		}
		pooled.dynamicConfig = maps.Clone(opts.DynamicConfigValues)
		pooled.namespaces = map[string]struct{}{namespace: {}}
		t.Cleanup(func() {
			p.Lock()
			delete(p.servers, key)
			p.Unlock()
			_ = pooled.server.Stop()
		})
	})
	if pooled.err != nil {
		p.Lock()
		if p.servers[key] == pooled {
			delete(p.servers, key)
		}
		p.Unlock()
		return nil, pooled.err
	}
	if started {
		return pooled.server, nil
	}
	return pooled.acquireNamespace(t, group, namespace, opts.DynamicConfigValues)
}

func (p *pooledTestDevServer) acquireNamespace(
	t *testing.T,
	group string,
	namespace string,
	dynamicConfig map[string]any,
) (*devserver.Server, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !maps.EqualFunc(p.dynamicConfig, dynamicConfig, reflect.DeepEqual) {
		return nil, fmt.Errorf(
			"dev server group %q already uses different dynamic config",
			group,
		)
	}
	if _, exists := p.namespaces[namespace]; exists {
		return nil, fmt.Errorf(
			"dev server group %q already uses namespace %q",
			group,
			namespace,
		)
	}
	if err := p.server.RegisterNamespace(t.Context(), namespace); err != nil {
		return nil, err
	}
	p.namespaces[namespace] = struct{}{}
	return p.server, nil
}
