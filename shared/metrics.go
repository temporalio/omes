package shared

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	metrics "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.temporal.io/sdk/client"
)

var meter = metrics.Meter("omes")

type metricsHandler struct {
	tags map[string]string
}

func NewMetricsHandler() *metricsHandler {
	return &metricsHandler{tags: make(map[string]string)}
}

func (h *metricsHandler) kvs() []attribute.KeyValue {
	kvs := make([]attribute.KeyValue, len(h.tags))
	for k, v := range h.tags {
		kvs = append(kvs, attribute.KeyValue{Key: attribute.Key(k), Value: attribute.StringValue(v)})
	}
	return kvs
}

func (h *metricsHandler) WithTags(tags map[string]string) client.MetricsHandler {
	// Make enough space for the handlers tags which are populated first
	mergedTags := make(map[string]string, len(h.tags))
	for t, v := range h.tags {
		mergedTags[t] = v
	}
	for t, v := range tags {
		mergedTags[t] = v
	}
	return &metricsHandler{tags: mergedTags}
}

func (h *metricsHandler) Counter(name string) client.MetricsCounter {
	ctr, err := meter.SyncInt64().Counter(name)
	// TODO: there's no way to return an error in this interface...
	if err != nil {
		panic(err)
	}
	return metricsCounter{ctr, h.kvs()}
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	gauge, err := meter.AsyncFloat64().Gauge(name)
	// TODO: there's no way to return an error in this interface...
	if err != nil {
		panic(err)
	}
	return metricsGauge{gauge, h.kvs()}
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	ctr, err := meter.SyncInt64().Histogram(name)
	// TODO: there's no way to return an error in this interface...
	if err != nil {
		panic(err)
	}
	return metricsTimer{ctr, h.kvs()}
}

type metricsCounter struct {
	syncint64.Counter
	attributes []attribute.KeyValue
}

// Inc increments the counter value.
func (m metricsCounter) Inc(incr int64) {
	m.Add(context.Background(), incr, m.attributes...)
}

type metricsGauge struct {
	asyncfloat64.Gauge
	attributes []attribute.KeyValue
}

// Update implements metrics.Gauge
func (m metricsGauge) Update(x float64) {
	m.Observe(context.Background(), x, m.attributes...)
}

type metricsTimer struct {
	syncint64.Histogram
	attributes []attribute.KeyValue
}

func (m metricsTimer) Record(duration time.Duration) {
	m.Histogram.Record(context.Background(), int64(duration), m.attributes...)
}
