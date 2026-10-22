package clioptions

import (
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/metrics"
	"go.uber.org/zap"
)

// StartProcessMetricsSidecar starts a process metrics server that monitors an external PID.
// This is called by run.go after starting the SDK worker subprocess.
// It serves /metrics (CPU/memory for the worker PID).
func StartProcessMetricsSidecar(logger *zap.SugaredLogger, address string, workerPID int) *http.Server {
	registry := prometheus.NewRegistry()
	procCollector, err := metrics.NewProcessCollector(workerPID)
	if err != nil {
		logger.Fatalf("Unable to setup process collector for PID %d: %v", workerPID, err)
	}
	registry.MustRegister(procCollector)

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	server := &http.Server{Addr: address, Handler: handler}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		logger.Fatalf("Failed to start process metrics sidecar on %s: %v", address, err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Process metrics sidecar error: %v", err)
		}
	}()

	logger.Infof("Process metrics sidecar started on %s (monitoring PID %d)", address, workerPID)
	return server
}

// MetricsOptions for setting up Prometheus metrics.
type MetricsOptions struct {
	// Address for the Prometheus HTTP listener.
	// If empty, the listener will not be started.
	PrometheusListenAddress string
	// HTTP path for serving metrics.
	// Default "/metrics".
	PrometheusHandlerPath string
	// Address for separate process metrics server (CPU/memory only).
	// If empty, process metrics will not be served separately.
	WorkerProcessMetricsAddress string

	fs         *pflag.FlagSet
	usedPrefix string
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
	m.fs.StringVar(&m.WorkerProcessMetricsAddress, prefix+"process-metrics-address", "", "Address for separate process metrics server (CPU/memory only)")
	return m.fs
}

// MustCreateMetrics sets up Prometheus based metrics and starts an HTTP server
// for serving SDK metrics.
func (m *MetricsOptions) MustCreateMetrics(logger *zap.SugaredLogger) *metrics.Metrics {
	registry := prometheus.NewRegistry()
	var server *http.Server

	if m.PrometheusListenAddress != "" {
		server = m.mustInitPrometheusServer(logger, registry)
	}

	return &metrics.Metrics{
		Server:   server,
		Registry: registry,
		Cache:    make(map[string]any),
	}
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
