package clioptions

import (
	"context"
	"fmt"
	"maps"
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
	maps.Copy(mergedTags, tags)

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

func (h *metricsHandler) Counter(name string) client.MetricsCounter {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	var ctr *prometheus.CounterVec
	if c, ok := h.metrics.cache[name]; ok {
		ctr, ok = c.(*prometheus.CounterVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		ctr = prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: name},
			h.labels,
		)
		h.metrics.registry.MustRegister(ctr)
		h.metrics.cache[name] = ctr
	}

	return metricsCounter{ctr.WithLabelValues(h.values...)}
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	var gauge *prometheus.GaugeVec
	if c, ok := h.metrics.cache[name]; ok {
		gauge, ok = c.(*prometheus.GaugeVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		gauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: name},
			h.labels,
		)
		h.metrics.registry.MustRegister(gauge)
		h.metrics.cache[name] = gauge
	}

	return metricsGauge{gauge.WithLabelValues(h.values...)}
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	var timer *prometheus.HistogramVec
	if c, ok := h.metrics.cache[name]; ok {
		timer, ok = c.(*prometheus.HistogramVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		// TODO: buckets
		timer = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, h.labels)
		h.metrics.registry.MustRegister(timer)
		h.metrics.cache[name] = timer
	}

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

	fs         *pflag.FlagSet
	usedPrefix string
}

// Metrics is a component for insrumenting an application with Promethues metrics.
type Metrics struct {
	server   *http.Server
	registry *prometheus.Registry
	cache    map[string]any
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
		cache:    make(map[string]any),
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

// FlagSet adds the relevant flags to populate the options struct and returns a pflag.FlagSet.
func (m *MetricsOptions) FlagSet(prefix string) *pflag.FlagSet {
	if m.fs != nil {
		if prefix != m.usedPrefix {
			panic("prefix mismatch")
		}
		return m.fs
	}
	m.usedPrefix = prefix
	m.fs = pflag.NewFlagSet("metrics_options", pflag.ExitOnError)
	m.fs.StringVar(&m.PrometheusListenAddress, prefix+"prom-listen-address", "", "Prometheus listen address")
	m.fs.StringVar(&m.PrometheusHandlerPath, prefix+"prom-handler-path", "/metrics", "Prometheus handler path")
	return m.fs
}
