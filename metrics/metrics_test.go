package metrics

import (
	"reflect"
	"testing"
)

func TestMetricsHandlerSortsTags(t *testing.T) {
	handler := (&Metrics{}).NewHandler().WithTags(map[string]string{
		"status_code": "Unavailable",
		"scenario":    "test-scenario",
		"outcome":     "failed",
	}).(*metricsHandler)

	wantLabels := []string{"outcome", "scenario", "status_code"}
	wantValues := []string{"failed", "test-scenario", "Unavailable"}
	if !reflect.DeepEqual(wantLabels, handler.labels) {
		t.Fatalf("labels = %v, want %v", handler.labels, wantLabels)
	}
	if !reflect.DeepEqual(wantValues, handler.values) {
		t.Fatalf("values = %v, want %v", handler.values, wantValues)
	}
}
