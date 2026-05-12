package devserver

import (
	"fmt"
	"net"
	"strconv"
)

type portSet struct {
	Host string

	FrontendGRPC       int
	FrontendMembership int
	FrontendHTTP       int
	HistoryGRPC        int
	HistoryMembership  int
	MatchingGRPC       int
	MatchingMembership int
	WorkerGRPC         int
	WorkerMembership   int
}

func (p portSet) frontendAddr() string {
	return net.JoinHostPort(p.Host, strconv.Itoa(p.FrontendGRPC))
}

// allocatePortSet returns 9 ports for the four temporal services. Behavior:
//   - portBase != 0: returns 9 sequential ports starting at portBase. Use this
//     when ports must stay below 32768 (postgres cluster_membership.rpc_port is
//     SMALLINT, and Linux ephemeral ports overflow it).
//   - portBase == 0: picks free ports by listening on :0 then closing — racy
//     but cheap. Honors a non-zero frontend port from bindAddress if set.
//
// bindAddress (when non-empty) parses as host:port; host (default 127.0.0.1)
// is used everywhere. Mixing portBase and a bindAddress port is rejected.
func allocatePortSet(bindAddress string, portBase int) (portSet, error) {
	host := "127.0.0.1"
	frontendPort := 0
	if bindAddress != "" {
		h, p, err := net.SplitHostPort(bindAddress)
		if err != nil {
			return portSet{}, fmt.Errorf("parse bind address %q: %w", bindAddress, err)
		}
		if h != "" {
			host = h
		}
		if p != "" && p != "0" {
			n, err := strconv.Atoi(p)
			if err != nil {
				return portSet{}, fmt.Errorf("parse port %q: %w", p, err)
			}
			frontendPort = n
		}
	}
	if portBase != 0 && frontendPort != 0 {
		return portSet{}, fmt.Errorf("cannot combine PortBase with an explicit bind port")
	}

	ports := make([]int, 9)
	if portBase != 0 {
		for i := range ports {
			ports[i] = portBase + i
		}
	} else {
		listeners := make([]net.Listener, 0, 9)
		defer func() {
			for _, l := range listeners {
				_ = l.Close()
			}
		}()
		for i := range ports {
			if i == 0 && frontendPort != 0 {
				ports[i] = frontendPort
				continue
			}
			l, err := net.Listen("tcp", host+":0")
			if err != nil {
				return portSet{}, fmt.Errorf("listen on free port: %w", err)
			}
			listeners = append(listeners, l)
			ports[i] = l.Addr().(*net.TCPAddr).Port
		}
	}

	return portSet{
		Host:               host,
		FrontendGRPC:       ports[0],
		FrontendMembership: ports[1],
		FrontendHTTP:       ports[2],
		HistoryGRPC:        ports[3],
		HistoryMembership:  ports[4],
		MatchingGRPC:       ports[5],
		MatchingMembership: ports[6],
		WorkerGRPC:         ports[7],
		WorkerMembership:   ports[8],
	}, nil
}
