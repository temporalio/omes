package starter

import (
	"fmt"
	"sync"

	"go.temporal.io/sdk/client"
)

// ClientPool caches SDK clients by user-provided key for the lifetime of the process.
// It is safe for concurrent use.
type ClientPool struct {
	mu      sync.Mutex
	clients map[string]client.Client
}

// NewClientPool creates an empty client pool.
func NewClientPool() *ClientPool {
	return &ClientPool{
		clients: make(map[string]client.Client),
	}
}

// GetOrDial returns a cached client for key or dials and caches a new one.
// The caller owns key selection. Use a stable key to reuse the same client.
func (p *ClientPool) GetOrDial(key string, opts client.Options) (client.Client, error) {
	if key == "" {
		return nil, fmt.Errorf("client pool key cannot be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if c := p.clients[key]; c != nil {
		return c, nil
	}

	c, err := client.Dial(opts)
	if err != nil {
		return nil, err
	}
	p.clients[key] = c
	return c, nil
}
