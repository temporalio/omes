package starter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.temporal.io/sdk/client"
)

const metricsShutdownTimeout = 5 * time.Second

// setupPrometheusMetrics configures SDK metrics emission and serves /metrics
// when PromListenAddress is provided.
func setupPrometheusMetrics(config *WorkerConfig) (func(), error) {
	if config.PromListenAddress == "" {
		return func() {}, nil
	}

	registry := prometheus.NewRegistry()
	config.ConnectionOptions.MetricsHandler = newPromMetricsHandler(registry)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    config.PromListenAddress,
		Handler: mux,
	}
	listener, err := net.Listen("tcp", config.PromListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on %s: %w", config.PromListenAddress, err)
	}

	go func() {
		_ = server.Serve(listener)
	}()

	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), metricsShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(ctx)
	}
	return cleanup, nil
}

type promMetricsState struct {
	mu       sync.Mutex
	registry *prometheus.Registry
	cache    map[string]any
}

type promMetricsHandler struct {
	state  *promMetricsState
	labels []string
	values []string
}

func newPromMetricsHandler(registry *prometheus.Registry) client.MetricsHandler {
	return &promMetricsHandler{
		state: &promMetricsState{
			registry: registry,
			cache:    make(map[string]any),
		},
	}
}

func (h *promMetricsHandler) WithTags(tags map[string]string) client.MetricsHandler {
	merged := make(map[string]string, len(h.labels)+len(tags))
	for i, label := range h.labels {
		merged[label] = h.values[i]
	}
	for k, v := range tags {
		merged[k] = v
	}

	labels := make([]string, 0, len(merged))
	for k := range merged {
		labels = append(labels, k)
	}
	sort.Strings(labels)

	values := make([]string, len(labels))
	for i, label := range labels {
		values[i] = merged[label]
	}

	return &promMetricsHandler{
		state:  h.state,
		labels: labels,
		values: values,
	}
}

func (h *promMetricsHandler) Counter(name string) client.MetricsCounter {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	var ctr *prometheus.CounterVec
	if c, ok := h.state.cache[name]; ok {
		var okType bool
		ctr, okType = c.(*prometheus.CounterVec)
		if !okType {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		ctr = prometheus.NewCounterVec(prometheus.CounterOpts{Name: name}, h.labels)
		h.state.registry.MustRegister(ctr)
		h.state.cache[name] = ctr
	}

	return promMetricsCounter{prom: ctr.WithLabelValues(h.values...)}
}

func (h *promMetricsHandler) Gauge(name string) client.MetricsGauge {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	var gauge *prometheus.GaugeVec
	if c, ok := h.state.cache[name]; ok {
		var okType bool
		gauge, okType = c.(*prometheus.GaugeVec)
		if !okType {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		gauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: name}, h.labels)
		h.state.registry.MustRegister(gauge)
		h.state.cache[name] = gauge
	}

	return promMetricsGauge{prom: gauge.WithLabelValues(h.values...)}
}

func (h *promMetricsHandler) Timer(name string) client.MetricsTimer {
	h.state.mu.Lock()
	defer h.state.mu.Unlock()

	var timer *prometheus.HistogramVec
	if c, ok := h.state.cache[name]; ok {
		var okType bool
		timer, okType = c.(*prometheus.HistogramVec)
		if !okType {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		timer = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, h.labels)
		h.state.registry.MustRegister(timer)
		h.state.cache[name] = timer
	}

	return promMetricsTimer{prom: timer.WithLabelValues(h.values...)}
}

type promMetricsCounter struct {
	prom prometheus.Counter
}

func (m promMetricsCounter) Inc(incr int64) {
	m.prom.Add(float64(incr))
}

type promMetricsGauge struct {
	prom prometheus.Gauge
}

func (m promMetricsGauge) Update(x float64) {
	m.prom.Set(x)
}

type promMetricsTimer struct {
	prom prometheus.Observer
}

func (m promMetricsTimer) Record(duration time.Duration) {
	m.prom.Observe(duration.Seconds())
}
