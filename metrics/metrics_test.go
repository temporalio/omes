package metrics

import (
	"fmt"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

// TestHandlerLabelsMatchTags verifies every exported label carries the value of the tag with the same name, even when
// many handlers with the same tag set share one metric.
func TestHandlerLabelsMatchTags(t *testing.T) {
	m := &Metrics{Registry: prometheus.NewRegistry(), Cache: map[string]any{}}
	root := m.NewHandler()

	// Each handler gets its own tag map, so map iteration order differs between them and a positional mix-up between
	// label names and values shows up within a few iterations.
	for i := range 50 {
		tags := map[string]string{"series": fmt.Sprint(i)}
		for _, k := range []string{"namespace", "operation", "task_queue", "workflow_type", "activity_type", "client_name", "worker_type"} {
			tags[k] = k + "-value"
		}
		h := root.WithTags(tags)
		h.Counter("test_counter").Inc(1)
		h.Gauge("test_gauge").Update(1)
		h.Timer("test_timer").Record(time.Millisecond)
	}

	families, err := m.Registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 3)
	for _, fam := range families {
		require.Len(t, fam.GetMetric(), 50, fam.GetName())
		for _, metric := range fam.GetMetric() {
			for _, lp := range metric.GetLabel() {
				if lp.GetName() == "series" {
					continue
				}
				require.Equal(t, lp.GetName()+"-value", lp.GetValue(), "metric %s", fam.GetName())
			}
		}
	}
}
