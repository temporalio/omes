package utils

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/temporalio/omes/cmd/clioptions"
)

// DefaultEntryFile returns the default entry file for a language.
func DefaultEntryFile(lang clioptions.Language) string {
	switch lang {
	case clioptions.LangPython:
		return "main.py"
	case clioptions.LangTypeScript:
		return "main.ts"
	default:
		return ""
	}
}

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

// WaitForReady polls a URL until it returns 200 or timeout is reached.
func WaitForReady(ctx context.Context, url string, timeout time.Duration) error {
	return WaitForReadyWithErrCh(ctx, url, timeout, nil)
}

// WaitForReadyWithErrCh polls a URL with optional background error channel monitoring.
func WaitForReadyWithErrCh(ctx context.Context, url string, timeout time.Duration, errCh <-chan error) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 5 * time.Second}

	for time.Now().Before(deadline) {
		// Check for startup error
		if errCh != nil {
			select {
			case err := <-errCh:
				if err != nil {
					return fmt.Errorf("process failed to start: %w", err)
				}
			default:
			}
		}

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
