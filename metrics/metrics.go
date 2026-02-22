package metrics

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Metrics is a component for instrumenting an application with Prometheus metrics.
type Metrics struct {
	Server               *http.Server
	ProcessMetricsServer *http.Server // Separate server for process metrics (CPU/memory)
	Registry             *prometheus.Registry
	Cache                map[string]any
	mutex                sync.Mutex
	PromInstance         *PrometheusInstance
}

// Handler returns a new Temporal-client-compatible metrics handler.
func (m *Metrics) NewHandler() client.MetricsHandler {
	return &metricsHandler{
		metrics: m,
	}
}

// Shutdown the Prometheus HTTP server and local Prometheus process if they were set up.
// scenario, runID, and runFamily are passed to the export function for metrics metadata.
func (m *Metrics) Shutdown(ctx context.Context, logger *zap.SugaredLogger, scenario, runID, runFamily string) error {
	// Shutdown prometheus process if running
	if m.PromInstance != nil {
		m.PromInstance.Shutdown(ctx, logger, scenario, runID, runFamily)
	}

	var err error
	// Shutdown main HTTP server
	if m.Server != nil {
		if shutdownErr := m.Server.Shutdown(ctx); shutdownErr != nil {
			err = fmt.Errorf("main metrics server: %w", shutdownErr)
		}
	}
	// Shutdown process metrics server
	if m.ProcessMetricsServer != nil {
		if shutdownErr := m.ProcessMetricsServer.Shutdown(ctx); shutdownErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; process metrics server: %w", err, shutdownErr)
			} else {
				err = fmt.Errorf("process metrics server: %w", shutdownErr)
			}
		}
	}
	return err
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
	rootProcess     *process.Process
	pidProcessCache map[int32]*process.Process
	cpuDesc         *prometheus.Desc
	memDesc         *prometheus.Desc
	memPercentDesc  *prometheus.Desc
	mutex           sync.Mutex
}

func NewProcessCollector(pid int) (*processCollector, error) {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return nil, fmt.Errorf("failed to create process for PID %d: %w", pid, err)
	}
	return &processCollector{
		rootProcess:     p,
		pidProcessCache: map[int32]*process.Process{p.Pid: p},
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
	ch <- c.memPercentDesc
}

func (c *processCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	children, _ := c.rootProcess.Children()

	seen := map[int32]bool{
		c.rootProcess.Pid: true,
	}
	procs := make([]*process.Process, 0, len(children)+1)
	procs = append(procs, c.rootProcess)
	for _, child := range children {
		if child == nil || seen[child.Pid] {
			continue
		}
		seen[child.Pid] = true
		if _, ok := c.pidProcessCache[child.Pid]; !ok {
			c.pidProcessCache[child.Pid] = child
		}
		procs = append(procs, c.pidProcessCache[child.Pid])
	}
	for pid := range c.pidProcessCache {
		if !seen[pid] {
			delete(c.pidProcessCache, pid)
		}
	}

	var totalCPU float64
	var totalRSS uint64
	for _, p := range procs {
		if cpuPercent, err := p.Percent(0); err == nil {
			totalCPU += cpuPercent
		}
		if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
			totalRSS += memInfo.RSS
		}
	}

	ch <- prometheus.MustNewConstMetric(c.cpuDesc, prometheus.GaugeValue, totalCPU)
	ch <- prometheus.MustNewConstMetric(c.memDesc, prometheus.GaugeValue, float64(totalRSS))

	if vm, err := mem.VirtualMemory(); err == nil && vm.Total > 0 {
		memPercent := (float64(totalRSS) / float64(vm.Total)) * 100
		ch <- prometheus.MustNewConstMetric(c.memPercentDesc, prometheus.GaugeValue, memPercent)
	}
}
