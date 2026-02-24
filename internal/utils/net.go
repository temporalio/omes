package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
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

// WaitForReady polls a URL until it returns 200 or timeout is reached.
func WaitForReady(ctx context.Context, url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			return nil
		}
		if resp != nil {
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
	return fmt.Errorf("timeout waiting for %s", url)
}
