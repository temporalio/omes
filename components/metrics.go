package components

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type metricsHandler struct {
	registry *prometheus.Registry
	tags     map[string]string
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
	return &metricsHandler{registry: h.registry, tags: mergedTags}
}

func (h *metricsHandler) mustRegisterIgnoreDuplicate(c prometheus.Collector) {
	err := h.registry.Register(c)
	var alreadyRegisteredError prometheus.AlreadyRegisteredError
	if err != nil && !errors.As(err, &alreadyRegisteredError) {
		panic(err)
	}
}

func (h *metricsHandler) Counter(name string) client.MetricsCounter {
	ctr := prometheus.NewCounter(prometheus.CounterOpts{Name: name, ConstLabels: prometheus.Labels(h.tags)})
	h.mustRegisterIgnoreDuplicate(ctr)
	return metricsCounter{ctr}
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	gauge := prometheus.NewGauge(prometheus.GaugeOpts{Name: name, ConstLabels: prometheus.Labels(h.tags)})
	h.mustRegisterIgnoreDuplicate(gauge)
	return metricsGauge{gauge}
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	// TODO: buckets
	timer := prometheus.NewHistogram(prometheus.HistogramOpts{Name: name, ConstLabels: prometheus.Labels(h.tags)})
	h.mustRegisterIgnoreDuplicate(timer)
	return metricsTimer{timer}
}

type metricsCounter struct {
	prom prometheus.Counter
}

// Inc (increment) the counter value.
func (m metricsCounter) Inc(incr int64) {
	m.prom.Add(float64(incr))
}

type metricsGauge struct {
	prom prometheus.Gauge
}

// Update the gauge with a new observation.
func (m metricsGauge) Update(x float64) {
	m.prom.Set(x)
}

type metricsTimer struct {
	prom prometheus.Histogram
}

// Record a duration.
func (m metricsTimer) Record(duration time.Duration) {
	m.prom.Observe(duration.Seconds())
}

type PrometheusOptions struct {
	// Address for the Prometheus HTTP listener.
	// Default to listen on all interfaces and port 9090.
	// TODO: should empty be treated as server disabled?
	ListenAddress string
	// HTTP path for serving metrics.
	// Default /metrics
	HandlerPath string
}

type Metrics struct {
	server   *http.Server
	registry *prometheus.Registry
}

func MustInitMetrics(options *PrometheusOptions, logger *zap.SugaredLogger) *Metrics {
	registry := prometheus.NewRegistry()
	var server *http.Server
	if options.ListenAddress != "" {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		server = mustInitPrometheusServer(options, logger, registry)
	}
	return &Metrics{
		server:   server,
		registry: registry,
	}
}

func (m *Metrics) Handler() *metricsHandler {
	return &metricsHandler{
		registry: m.registry,
		tags:     make(map[string]string),
	}
}

func (m *Metrics) Shutdown(ctx context.Context) error {
	// server might be nil if no listen address was provided
	if m.server == nil {
		return nil
	}
	return m.server.Shutdown(ctx)
}

func mustInitPrometheusServer(options *PrometheusOptions, logger *zap.SugaredLogger, registry *prometheus.Registry) *http.Server {
	address := options.ListenAddress
	handlerPath := options.HandlerPath
	if handlerPath == "" {
		handlerPath = "/metrics"
	}

	handler := http.NewServeMux()
	handler.Handle(handlerPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

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
