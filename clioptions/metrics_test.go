package clioptions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func eventually(t *testing.T, timeout time.Duration, fn func() error) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if lastErr = fn(); lastErr == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met within %v: %v", timeout, lastErr)
}

func TestMetricsServer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	ctx := context.Background()

	t.Run("SDK metrics server has no process metrics", func(t *testing.T) {
		opts := &MetricsOptions{
			PrometheusListenAddress: ":19090",
		}

		metrics := opts.MustCreateMetrics(ctx, logger)
		defer metrics.Shutdown(ctx, logger, "", "", "")

		eventually(t, 2*time.Second, func() error {
			sdkMetrics, err := tryFetchMetrics("http://localhost:19090/metrics")
			if err != nil {
				return err
			}
			if strings.Contains(sdkMetrics, "process_cpu_percent") {
				return errors.New("SDK server should NOT have process_cpu_percent")
			}
			if strings.Contains(sdkMetrics, "process_resident_memory_bytes") {
				return errors.New("SDK server should NOT have process_resident_memory_bytes")
			}
			return nil
		})
	})

	t.Run("SDK server shutdown", func(t *testing.T) {
		opts := &MetricsOptions{
			PrometheusListenAddress: ":19091",
		}

		metrics := opts.MustCreateMetrics(ctx, logger)

		eventually(t, 2*time.Second, func() error {
			_, err := tryFetchMetrics("http://localhost:19091/metrics")
			return err
		})

		err := metrics.Shutdown(ctx, logger, "", "", "")
		if err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}

		eventually(t, 2*time.Second, func() error {
			_, err := http.Get("http://localhost:19091/metrics")
			if err == nil {
				return errors.New("SDK server should be stopped after shutdown")
			}
			return nil
		})
	})
}

func TestProcessMetricsSidecar(t *testing.T) {
	logger := zap.NewNop().Sugar()
	ctx := context.Background()

	t.Run("sidecar has process metrics", func(t *testing.T) {
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19092",
			os.Getpid(),
			"v1.24.0",
			"test-build-123",
			"go",
		)
		defer sidecar.Shutdown(ctx)

		eventually(t, 2*time.Second, func() error {
			processMetrics, err := tryFetchMetrics("http://localhost:19092/metrics")
			if err != nil {
				return err
			}
			if !strings.Contains(processMetrics, "process_cpu_percent") {
				return errors.New("Sidecar should have process_cpu_percent")
			}
			if !strings.Contains(processMetrics, "process_resident_memory_bytes") {
				return errors.New("Sidecar should have process_resident_memory_bytes")
			}
			if strings.Contains(processMetrics, "temporal_") {
				return errors.New("Sidecar should NOT have temporal_ metrics")
			}
			return nil
		})
	})

	t.Run("info endpoint returns sdk_version, build_id, and language", func(t *testing.T) {
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19093",
			os.Getpid(),
			"v1.24.0",
			"test-build-456",
			"python",
		)
		defer sidecar.Shutdown(ctx)

		eventually(t, 2*time.Second, func() error {
			resp, err := http.Get("http://localhost:19093/info")
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.Header.Get("Content-Type") != "application/json" {
				return errors.New("Expected Content-Type application/json")
			}

			var info InfoResponse
			if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
				return err
			}

			if info.SDKVersion != "v1.24.0" {
				return errors.New("Expected sdk_version v1.24.0")
			}
			if info.BuildID != "test-build-456" {
				return errors.New("Expected build_id test-build-456")
			}
			if info.Language != "python" {
				return errors.New("Expected language python")
			}
			return nil
		})
	})

	t.Run("sidecar shutdown", func(t *testing.T) {
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19094",
			os.Getpid(),
			"v1.24.0",
			"test-build",
			"typescript",
		)

		eventually(t, 2*time.Second, func() error {
			_, err := tryFetchMetrics("http://localhost:19094/metrics")
			return err
		})

		err := sidecar.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}

		eventually(t, 2*time.Second, func() error {
			_, err := http.Get("http://localhost:19094/metrics")
			if err == nil {
				return errors.New("Sidecar should be stopped after shutdown")
			}
			return nil
		})
	})
}

func tryFetchMetrics(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
