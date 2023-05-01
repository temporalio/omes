package loadgen

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type metricsHandler struct {
	registry *prometheus.Registry
	tags     map[string]string
}

var _ client.MetricsHandler = (*metricsHandler)(nil)

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
	prom prometheus.Histogram
}

// Record records a duration.
func (m metricsTimer) Record(duration time.Duration) {
	m.prom.Observe(duration.Seconds())
}

// MetricsOptions for setting up Prometheus metrics.
type MetricsOptions struct {
	// Address for the Prometheus HTTP listener.
	// If empty, the listener will not be started.
	PrometheusListenAddress string `flag:"prom-listen-address"`
	// HTTP path for serving metrics.
	// Default "/metrics".
	PrometheusHandlerPath string `flag:"prom-handler-path"`
}

// Metrics is a component for insrumenting an application with Promethues metrics.
type Metrics struct {
	server   *http.Server
	registry *prometheus.Registry
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
	}
}

// Handler returns a new Temporal-client-compatible metrics handler.
func (m *Metrics) NewHandler() client.MetricsHandler {
	return &metricsHandler{
		registry: m.registry,
		tags:     make(map[string]string),
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
	fs.StringVar(&m.PrometheusListenAddress, fmt.Sprintf("%s%s", prefix,
		OptionToFlagName(m, "PrometheusListenAddress")), "", "Prometheus listen address")
	fs.StringVar(&m.PrometheusHandlerPath, fmt.Sprintf("%s%s", prefix,
		OptionToFlagName(m, "PrometheusHandlerPath")), "/metrics", "Prometheus handler path")
}
