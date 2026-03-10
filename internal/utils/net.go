package utils

import (
	"context"
	"fmt"
	"net"
	"time"
)

// FindAvailablePort finds an available TCP port.
func FindAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port, nil
}

// DefaultPort assigns a random available port to *p if it is 0.
func DefaultPort(p *int) error {
	if *p == 0 {
		port, err := FindAvailablePort()
		if err != nil {
			return err
		}
		*p = port
	}
	return nil
}

// WaitUntil retries fn until timeout elapses or it succeeds.
func WaitUntil(ctx context.Context, fn func() error, timeout time.Duration, retryInterval time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
		}
	}
	return fmt.Errorf("ready check failed")
}
