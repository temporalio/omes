package clioptions

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/metrics"
	"go.uber.org/zap"
)

// InfoResponse is returned by the /info endpoint on the process metrics server.
// Only contains fields that run-scenario doesn't already know.
type InfoResponse struct {
	SDKVersion string `json:"sdk_version"`
	BuildID    string `json:"build_id"`
	Language   string `json:"language"`
}

// StartProcessMetricsSidecar starts a process metrics server that monitors an external PID.
// This is called by run.go after starting the SDK worker subprocess.
// It serves /metrics (CPU/memory for the worker PID) and /info (worker metadata).
func StartProcessMetricsSidecar(
	logger *zap.SugaredLogger,
	address string,
	workerPID int,
	sdkVersion string,
	buildID string,
	language string,
) *http.Server {
	registry := prometheus.NewRegistry()
	procCollector, err := metrics.NewProcessCollector(workerPID)
	if err != nil {
		logger.Fatalf("Unable to setup process collector for PID %d: %v", workerPID, err)
	}
	registry.MustRegister(procCollector)

	handler := http.NewServeMux()
	handler.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	handler.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(InfoResponse{
			SDKVersion: sdkVersion,
			BuildID:    buildID,
			Language:   language,
		})
	})

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
	// MetricsVersionTag is the SDK version/ref to report in metrics.
	// This is used by the sidecar's /info endpoint and is NOT passed to the worker.
	// If empty, falls back to the --version flag value.
	MetricsVersionTag           string
	prometheusInstanceOptions   PrometheusInstanceFlags

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
	m.fs.StringVar(&m.MetricsVersionTag, prefix+"metrics-version-tag", "", "SDK version/ref to report in metrics (sidecar only, not passed to worker)")
	m.fs.AddFlagSet(m.prometheusInstanceOptions.FlagSet(prefix))
	return m.fs
}

// MustCreateMetrics sets up Prometheus based metrics and starts an HTTP server
// for serving SDK metrics.
func (m *MetricsOptions) MustCreateMetrics(ctx context.Context, logger *zap.SugaredLogger) *metrics.Metrics {
	registry := prometheus.NewRegistry()
	var server *http.Server

	if m.PrometheusListenAddress != "" {
		server = m.mustInitPrometheusServer(logger, registry)
	}

	var promInstance *metrics.PrometheusInstance
	if m.prometheusInstanceOptions.IsConfigured() {
		promInstance = m.prometheusInstanceOptions.StartPrometheusInstance(ctx, logger)
	}
	return &metrics.Metrics{
		Server:       server,
		Registry:     registry,
		Cache:        make(map[string]any),
		PromInstance: promInstance,
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

type PrometheusInstanceFlags struct {
	metrics.PrometheusInstanceOptions
	fs *pflag.FlagSet
}

func (p *PrometheusInstanceFlags) FlagSet(prefix string) *pflag.FlagSet {
	if p.fs != nil {
		return p.fs
	}
	p.fs = pflag.NewFlagSet(prefix+"prom_instance_options", pflag.ExitOnError)
	p.fs.StringVar(&p.Address, prefix+"prom-instance-addr", "", "Prometheus instance address")
	p.fs.StringVar(&p.ConfigPath, prefix+"prom-instance-config", "prom-config.yml", "Start a local Prometheus instance with the specified config file")
	p.fs.BoolVar(&p.Snapshot, "prom-snapshot", false, "Create a TSDB snapshot on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsPath, prefix+"prom-export-worker-metrics", "", "Export worker process metrics to the specified file on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsJob, prefix+"prom-export-worker-job", "omes-worker", "Name of the worker job to export SDK metrics for")
	p.fs.StringVar(&p.ExportProcessMetricsJob, prefix+"prom-export-process-job", "omes-worker-process", "Name of the process metrics job to export")
	p.fs.DurationVar(&p.ExportMetricsStep, prefix+"prom-export-metrics-step", 15*time.Second, "Step interval to sample timeseries metrics")
	p.fs.StringVar(&p.ExportWorkerInfoAddress, prefix+"prom-export-worker-info-address", "", "Address to fetch /info from during export (e.g., localhost:9091)")
	return p.fs
}
