package clioptions

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestMetricsServer(t *testing.T) {
	logger := zap.NewNop().Sugar()
	ctx := context.Background()

	t.Run("SDK metrics server has no process metrics", func(t *testing.T) {
		opts := &MetricsOptions{
			PrometheusListenAddress: ":19090",
		}

		metrics := opts.MustCreateMetrics(ctx, logger)
		defer metrics.Shutdown(ctx, logger, "", "")

		time.Sleep(100 * time.Millisecond)

		// SDK server should NOT have process metrics
		sdkMetrics := fetchMetrics(t, "http://localhost:19090/metrics")
		if strings.Contains(sdkMetrics, "process_cpu_percent") {
			t.Error("SDK server should NOT have process_cpu_percent")
		}
		if strings.Contains(sdkMetrics, "process_resident_memory_bytes") {
			t.Error("SDK server should NOT have process_resident_memory_bytes")
		}
	})

	t.Run("SDK server shutdown", func(t *testing.T) {
		opts := &MetricsOptions{
			PrometheusListenAddress: ":19091",
		}

		metrics := opts.MustCreateMetrics(ctx, logger)
		time.Sleep(100 * time.Millisecond)

		// Verify server is running
		fetchMetrics(t, "http://localhost:19091/metrics")

		// Shutdown
		err := metrics.Shutdown(ctx, logger, "", "")
		if err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}

		// Give time for shutdown
		time.Sleep(100 * time.Millisecond)

		// Verify server is stopped
		_, err = http.Get("http://localhost:19091/metrics")
		if err == nil {
			t.Error("SDK server should be stopped after shutdown")
		}
	})
}

func TestProcessMetricsSidecar(t *testing.T) {
	logger := zap.NewNop().Sugar()
	ctx := context.Background()

	t.Run("sidecar has process metrics", func(t *testing.T) {
		// Start sidecar monitoring current process
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19092",
			os.Getpid(),
			"v1.24.0",
			"test-build-123",
		)
		defer sidecar.Shutdown(ctx)

		time.Sleep(100 * time.Millisecond)

		// Sidecar SHOULD have process metrics
		processMetrics := fetchMetrics(t, "http://localhost:19092/metrics")
		if !strings.Contains(processMetrics, "process_cpu_percent") {
			t.Error("Sidecar should have process_cpu_percent")
		}
		if !strings.Contains(processMetrics, "process_resident_memory_bytes") {
			t.Error("Sidecar should have process_resident_memory_bytes")
		}

		// Sidecar should NOT have SDK metrics (temporal_*)
		if strings.Contains(processMetrics, "temporal_") {
			t.Error("Sidecar should NOT have temporal_ metrics")
		}
	})

	t.Run("info endpoint returns sdk_version and build_id", func(t *testing.T) {
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19093",
			os.Getpid(),
			"v1.24.0",
			"test-build-456",
		)
		defer sidecar.Shutdown(ctx)

		time.Sleep(100 * time.Millisecond)

		resp, err := http.Get("http://localhost:19093/info")
		if err != nil {
			t.Fatalf("Failed to fetch /info: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
		}

		var info InfoResponse
		if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
			t.Fatalf("Failed to decode /info response: %v", err)
		}

		if info.SDKVersion != "v1.24.0" {
			t.Errorf("Expected sdk_version v1.24.0, got %s", info.SDKVersion)
		}
		if info.BuildID != "test-build-456" {
			t.Errorf("Expected build_id test-build-456, got %s", info.BuildID)
		}
	})

	t.Run("sidecar shutdown", func(t *testing.T) {
		sidecar := StartProcessMetricsSidecar(
			logger,
			":19094",
			os.Getpid(),
			"v1.24.0",
			"test-build",
		)

		time.Sleep(100 * time.Millisecond)

		// Verify server is running
		fetchMetrics(t, "http://localhost:19094/metrics")

		// Shutdown
		err := sidecar.Shutdown(ctx)
		if err != nil {
			t.Errorf("Shutdown returned error: %v", err)
		}

		// Give time for shutdown
		time.Sleep(100 * time.Millisecond)

		// Verify server is stopped
		_, err = http.Get("http://localhost:19094/metrics")
		if err == nil {
			t.Error("Sidecar should be stopped after shutdown")
		}
	})
}

func fetchMetrics(t *testing.T, url string) string {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to fetch %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	return string(body)
}
