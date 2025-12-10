package metrics

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/process"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Metrics is a component for insrumenting an application with Prometheus metrics.
type Metrics struct {
	Server       *http.Server
	Registry     *prometheus.Registry
	Cache        map[string]any
	mutex        sync.Mutex
	PromInstance *PrometheusInstance
}

// Handler returns a new Temporal-client-compatible metrics handler.
func (m *Metrics) NewHandler() client.MetricsHandler {
	return &metricsHandler{
		metrics: m,
	}
}

// Shutdown the Prometheus HTTP server and local Prometheus process if they were set up.
func (m *Metrics) Shutdown(ctx context.Context, logger *zap.SugaredLogger) error {
	// Shutdown prometheus process if running
	if m.PromInstance != nil {
		m.PromInstance.Shutdown(ctx, logger)
	}

	// Shutdown HTTP server
	if m.Server != nil {
		return m.Server.Shutdown(ctx)
	}
	return nil
}

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
	if c, ok := h.metrics.Cache[name]; ok {
		ctr, ok = c.(*prometheus.CounterVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		ctr = prometheus.NewCounterVec(
			prometheus.CounterOpts{Name: name},
			h.labels,
		)
		h.metrics.Registry.MustRegister(ctr)
		h.metrics.Cache[name] = ctr
	}

	return metricsCounter{ctr.WithLabelValues(h.values...)}
}

func (h *metricsHandler) Gauge(name string) client.MetricsGauge {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	var gauge *prometheus.GaugeVec
	if c, ok := h.metrics.Cache[name]; ok {
		gauge, ok = c.(*prometheus.GaugeVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		gauge = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{Name: name},
			h.labels,
		)
		h.metrics.Registry.MustRegister(gauge)
		h.metrics.Cache[name] = gauge
	}

	return metricsGauge{gauge.WithLabelValues(h.values...)}
}

func (h *metricsHandler) Timer(name string) client.MetricsTimer {
	h.metrics.mutex.Lock()
	defer h.metrics.mutex.Unlock()

	var timer *prometheus.HistogramVec
	if c, ok := h.metrics.Cache[name]; ok {
		timer, ok = c.(*prometheus.HistogramVec)
		if !ok {
			panic(fmt.Errorf("duplicate metric with different type: %s", name))
		}
	} else {
		// TODO: buckets
		timer = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: name}, h.labels)
		h.metrics.Registry.MustRegister(timer)
		h.metrics.Cache[name] = timer
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

// processCollector is a cross-platform Prometheus collector that uses gopsutil
// to collect CPU and memory metrics. The standard ProcessCollector from Prometheus
// only works on Linux (requires /proc).
type processCollector struct {
	process        *process.Process
	cpuDesc        *prometheus.Desc
	memDesc        *prometheus.Desc
	memPercentDesc *prometheus.Desc
}

func NewProcessCollector() (*processCollector, error) {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, err
	}
	return &processCollector{
		process: p,
		cpuDesc: prometheus.NewDesc(
			"process_cpu_percent",
			"CPU usage as a percentage (100 = 1 core).",
			nil, nil,
		),
		memDesc: prometheus.NewDesc(
			"process_resident_memory_bytes",
			"Resident memory size in bytes.",
			nil, nil,
		),
		memPercentDesc: prometheus.NewDesc(
			"process_memory_percent",
			"Memory usage as a percentage of total system memory.",
			nil, nil,
		),
	}, nil
}

func (c *processCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.cpuDesc
	ch <- c.memDesc
}

func (c *processCollector) Collect(ch chan<- prometheus.Metric) {
	// CPU percent
	if cpuPercent, err := c.process.Percent(0); err == nil {
		ch <- prometheus.MustNewConstMetric(c.cpuDesc, prometheus.GaugeValue, cpuPercent)
	}

	// Resident memory (RSS)
	if memInfo, err := c.process.MemoryInfo(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.memDesc, prometheus.GaugeValue, float64(memInfo.RSS))
	}

	// Percent of total system memory
	if memPercent, err := c.process.MemoryPercent(); err == nil {
		ch <- prometheus.MustNewConstMetric(c.memPercentDesc, prometheus.GaugeValue, float64(memPercent))
	}
}
