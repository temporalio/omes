package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchWorkerInfo(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		// Create a test server that returns worker info
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/info" {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sdk_version": "v1.24.0",
				"build_id":    "test-build-456",
			})
		}))
		defer server.Close()

		// Extract host:port from URL (remove http://)
		address := strings.TrimPrefix(server.URL, "http://")

		info, err := fetchWorkerInfo(address)
		if err != nil {
			t.Fatalf("fetchWorkerInfo failed: %v", err)
		}

		if info.SDKVersion != "v1.24.0" {
			t.Errorf("Expected sdk_version 'v1.24.0', got '%s'", info.SDKVersion)
		}
		if info.BuildID != "test-build-456" {
			t.Errorf("Expected build_id 'test-build-456', got '%s'", info.BuildID)
		}
	})
}
