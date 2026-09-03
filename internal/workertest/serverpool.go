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
	test          *testing.T
	refs          int
}

type testDevServerPool struct {
	sync.Mutex
	servers map[string]*pooledTestDevServer
}

var groupedTestDevServers = testDevServerPool{
	servers: make(map[string]*pooledTestDevServer),
}

func (p *testDevServerPool) acquire(
	t *testing.T,
	cfg testEnvConfig,
	opts devserver.Options,
) (*devserver.Server, func(), error) {
	p.Lock()
	defer p.Unlock()

	if pooled := p.servers[cfg.devServerGroup]; pooled != nil {
		if pooled.test != t {
			return nil, nil, fmt.Errorf(
				"dev server group %q already belongs to another test",
				cfg.devServerGroup,
			)
		}
		if !reflect.DeepEqual(pooled.dynamicConfig, cfg.dynamicConfig) {
			return nil, nil, fmt.Errorf(
				"dev server group %q already uses different dynamic config",
				cfg.devServerGroup,
			)
		}
		if _, exists := pooled.namespaces[cfg.namespace]; exists {
			return nil, nil, fmt.Errorf(
				"dev server group %q already uses namespace %q",
				cfg.devServerGroup,
				cfg.namespace,
			)
		}
		pooled.namespaces[cfg.namespace] = struct{}{}
		pooled.refs++
		return pooled.server, func() { p.release(cfg.devServerGroup, pooled.server) }, nil
	}

	server, err := devserver.Start(t.Context(), opts)
	if err != nil {
		return nil, nil, err
	}
	p.servers[cfg.devServerGroup] = &pooledTestDevServer{
		server:        server,
		dynamicConfig: maps.Clone(cfg.dynamicConfig),
		namespaces:    map[string]struct{}{cfg.namespace: {}},
		test:          t,
		refs:          1,
	}
	return server, func() { p.release(cfg.devServerGroup, server) }, nil
}

func (p *testDevServerPool) release(id string, server *devserver.Server) {
	p.Lock()
	pooled := p.servers[id]
	if pooled == nil || pooled.server != server {
		p.Unlock()
		return
	}
	pooled.refs--
	if pooled.refs > 0 {
		p.Unlock()
		return
	}
	delete(p.servers, id)
	p.Unlock()
	_ = server.Stop()
}
