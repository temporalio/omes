package devserver

import (
	"fmt"
	"math/rand"
	"net"
)

// Postgres stores cluster_membership.rpc_port as SMALLINT (max 32767), so
// every port must stay below that. Linux ephemeral ports start at 32768
// which would overflow — pick from a fixed safe window instead.
const (
	safePortMin = 10000
	safePortMax = 30000
)

// allocatePorts returns portCount consecutive free ports in the postgres-safe
// range (<32768). Picks a random base in [safePortMin, safePortMax-portCount)
// and probes; retries on collision.
func allocatePorts(host string) ([portCount]int, error) {
	for attempt := 0; attempt < 50; attempt++ {
		base := safePortMin + rand.Intn(safePortMax-safePortMin-portCount)
		if ports, ok := probeContiguous(host, base); ok {
			return ports, nil
		}
	}
	return [portCount]int{}, fmt.Errorf("allocatePorts: could not find free contiguous range")
}

// probeContiguous tries to bind portCount sequential ports starting at base.
// Returns the array and true on success; closes all listeners before returning
// so callers can rebind. Race window is small but real.
func probeContiguous(host string, base int) ([portCount]int, bool) {
	var ports [portCount]int
	listeners := make([]net.Listener, 0, portCount)
	defer func() {
		for _, l := range listeners {
			_ = l.Close()
		}
	}()
	for i := 0; i < portCount; i++ {
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", host, base+i))
		if err != nil {
			return [portCount]int{}, false
		}
		listeners = append(listeners, l)
		ports[i] = base + i
	}
	return ports, true
}
