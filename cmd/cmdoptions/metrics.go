package cmdoptions

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type metricsHandler struct {
	metrics *Metrics
	labels  []string
	values  []string
}

var _ client.MetricsHandler = (*metricsHandler)(nil)

func (h *metricsHandler) WithTags(tags map[string]string) client.MetricsHandler {
	// Make enough space for the handlers tags which are populated first
	mergedTags := make(map[string]string, len(h.labels))
	for i, l := range h.labels {
		mergedTags[l] = h.values[i]
	}
	for l, v := range tags {
		mergedTags[l] = v
	}

	var labels, values []string
	for l, v := range mergedTags {
		labels = append(labels, l)
		values = append(values, v)
	}

	return &metricsHandler{
		metrics: h.metrics,
		labels:  labels,
		values:  values,
	}
}

func (h *metricsHandler) getOrCreateCounter(name string) *prometheus.CounterVec {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	if c, ok := h.metrics.cache[name]; ok {
		if ctr, ok := c.(*prometheus.CounterVec); ok {
			return ctr
		}
		panic(fmt.Errorf("duplicate metric with different type: %s", name))
	}

	m := prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: name},
		h.labels,
	)
	h.metrics.registry.MustRegister(m)
	h.metrics.cache[name] = m

	return m
}

func (h *metricsHandler) Counter(name string) client.MetricsCounter {
	ctr := h.getOrCreateCounter(name)
	return metricsCounter{ctr.WithLabelValues(h.values...)}
}

func (h *metricsHandler) getOrCreateGauge(name string) *prometheus.GaugeVec {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	if c, ok := h.metrics.cache[name]; ok {
		if gauge, ok := c.(*prometheus.GaugeVec); ok {
			return gauge
		}
		panic(fmt.Errorf("duplicate metric with different type: %s", name))
	}

	m := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: name},
		h.labels,
	)
	h.metrics.registry.MustRegister(m)
	h.metrics.cache[name] = m

	return m
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	gauge := h.getOrCreateGauge(name)
	return metricsGauge{gauge.WithLabelValues(h.values...)}
}

func (h *metricsHandler) getOrCreateTimer(name string) *prometheus.HistogramVec {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	if c, ok := h.metrics.cache[name]; ok {
		if h, ok := c.(*prometheus.HistogramVec); ok {
			return h
		}
		panic(fmt.Errorf("duplicate metric with different type: %s", name))
	}

	m := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: name},
		h.labels,
	)
	h.metrics.registry.MustRegister(m)
	h.metrics.cache[name] = m

	return m
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	// TODO: buckets
	timer := h.getOrCreateTimer(name)
	return metricsTimer{timer.WithLabelValues(h.values...)}
}

type metricsCounter struct {
	prom prometheus.Counter
}

// Inc increments the counter value.
func (m metricsCounter) Inc(incr int64) {
	m.prom.Add(float64(incr))
}

type metricsGauge struct {
	prom prometheus.Gauge
}

// Update updates the gauge with a new observation.
func (m metricsGauge) Update(x float64) {
	m.prom.Set(x)
}

type metricsTimer struct {
	prom prometheus.Observer
}

// Record records a duration.
func (m metricsTimer) Record(duration time.Duration) {
	m.prom.Observe(duration.Seconds())
}

// MetricsOptions for setting up Prometheus metrics.
type MetricsOptions struct {
	// Address for the Prometheus HTTP listener.
	// If empty, the listener will not be started.
	PrometheusListenAddress string
	// HTTP path for serving metrics.
	// Default "/metrics".
	PrometheusHandlerPath string
}

// Metrics is a component for insrumenting an application with Promethues metrics.
type Metrics struct {
	server   *http.Server
	registry *prometheus.Registry
	cache    map[string]interface{}
	mutex    sync.Mutex
}

// MustCreateMetrics sets up Prometheus based metrics and starts an HTTP server
// for serving metrics.
func (m *MetricsOptions) MustCreateMetrics(logger *zap.SugaredLogger) *Metrics {
	registry := prometheus.NewRegistry()
	var server *http.Server
	if m.PrometheusListenAddress != "" {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
		server = m.mustInitPrometheusServer(logger, registry)
	}
	return &Metrics{
		server:   server,
		registry: registry,
		cache:    make(map[string]interface{}),
	}
}

// Handler returns a new Temporal-client-compatible metrics handler.
func (m *Metrics) NewHandler() client.MetricsHandler {
	return &metricsHandler{
		metrics: m,
	}
}

// Shutdown the Promethus HTTP server if one was set up.
func (m *Metrics) Shutdown(ctx context.Context) error {
	// server might be nil if no listen address was provided
	if m.server == nil {
		return nil
	}
	return m.server.Shutdown(ctx)
}

func (m *MetricsOptions) mustInitPrometheusServer(logger *zap.SugaredLogger, registry *prometheus.Registry) *http.Server {
	address := m.PrometheusListenAddress
	handlerPath := m.PrometheusHandlerPath
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

// AddCLIFlags adds the relevant flags to populate the options struct.
func (m *MetricsOptions) AddCLIFlags(fs *pflag.FlagSet, prefix string) {
	fs.StringVar(&m.PrometheusListenAddress, prefix+"prom-listen-address", "", "Prometheus listen address")
	fs.StringVar(&m.PrometheusHandlerPath, prefix+"prom-handler-path", "/metrics", "Prometheus handler path")
}

// ToFlags converts these options to string flags.
func (m *MetricsOptions) ToFlags() (flags []string) {
	if m.PrometheusListenAddress != "" {
		flags = append(flags, "--prom-listen-address", m.PrometheusListenAddress)
	}
	if m.PrometheusHandlerPath != "" {
		flags = append(flags, "--prom-handler-path", m.PrometheusHandlerPath)
	}
	return
}
