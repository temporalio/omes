package app

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	otelMetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/instrument/asyncfloat64"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type metricsHandler struct {
	meter otelMetric.Meter
	tags  map[string]string
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
	ctr, err := h.meter.SyncInt64().Counter(name)
	// There's no way to return an error in this interface, we'll assume it's okay to panic
	if err != nil {
		panic(err)
	}
	return metricsCounter{ctr, h.kvs()}
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	gauge, err := h.meter.AsyncFloat64().Gauge(name)
	// There's no way to return an error in this interface, we'll assume it's okay to panic
	if err != nil {
		panic(err)
	}
	return metricsGauge{gauge, h.kvs()}
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	ctr, err := h.meter.SyncInt64().Histogram(name)
	// There's no way to return an error in this interface, we'll assume it's okay to panic
	if err != nil {
		panic(err)
	}
	return metricsTimer{ctr, h.kvs()}
}

type metricsCounter struct {
	syncint64.Counter
	attributes []attribute.KeyValue
}

// Inc (increment) the counter value.
func (m metricsCounter) Inc(incr int64) {
	m.Add(context.Background(), incr, m.attributes...)
}

type metricsGauge struct {
	asyncfloat64.Gauge
	attributes []attribute.KeyValue
}

// Update the gauge with a new observation.
func (m metricsGauge) Update(x float64) {
	m.Observe(context.Background(), x, m.attributes...)
}

type metricsTimer struct {
	syncint64.Histogram
	attributes []attribute.KeyValue
}

// Record a duration.
func (m metricsTimer) Record(duration time.Duration) {
	m.Histogram.Record(context.Background(), int64(duration), m.attributes...)
}

type PrometheusOptions struct {
	// Address for the Prometheus HTTP listener.
	// Default to listen on all interfaces and port 9090.
	ListenAddress string
	// HTTP path for serving metrics.
	// Default /metrics
	HandlerPath string
}

type Metrics struct {
	server        *http.Server
	MeterProvider *metric.MeterProvider
}

func MustInitMetrics(options *PrometheusOptions, logger *zap.SugaredLogger) *Metrics {
	exporter, err := prometheus.New()
	if err != nil {
		logger.Fatalf("Failed to instantiate a prometheus exporter: %v", err)
	}
	meterProvider := metric.NewMeterProvider(metric.WithReader(exporter))
	server := mustInitPrometheusServer(options, logger)
	return &Metrics{
		server:        server,
		MeterProvider: meterProvider,
	}
}

func (m *Metrics) Handler() *metricsHandler {
	// TODO: figure out the meter name
	return &metricsHandler{
		meter: m.MeterProvider.Meter("temporal"),
		tags:  make(map[string]string),
	}
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	return m.server.Shutdown(ctx)
}

func mustInitPrometheusServer(options *PrometheusOptions, logger *zap.SugaredLogger) *http.Server {
	address := options.ListenAddress
	if address == "" {
		address = ":9090"
	}
	handlerPath := options.HandlerPath
	if handlerPath == "" {
		handlerPath = "/metrics"
	}

	handler := http.NewServeMux()
	handler.Handle(handlerPath, promhttp.Handler())

	server := &http.Server{Addr: address, Handler: handler}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatalf("Failed to initialize Prometheus HTTP listener on %s: %v", address, err)
	}

	go func() {
		err := server.Serve(listener)
		if err != http.ErrServerClosed {
			logger.Fatalf("Fatal error in Prometheus HTTP server: %v", err)
		}
	}()

	return server
}
