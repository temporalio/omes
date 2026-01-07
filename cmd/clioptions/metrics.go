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

// InfoResponse is returned by the /info endpoint on worker metrics servers
type InfoResponse struct {
	SdkOptions      SdkOptions `json:"sdk_options"`
	ScenarioOptions ScenarioID `json:"scenario_options"`
	BuildID         string     `json:"build_id"`
}

// MetricsOptions for setting up Prometheus metrics.
type MetricsOptions struct {
	// Address for the Prometheus HTTP listener.
	// If empty, the listener will not be started.
	PrometheusListenAddress string
	// HTTP path for serving metrics.
	// Default "/metrics".
	PrometheusHandlerPath     string
	prometheusInstanceOptions PrometheusInstanceFlags

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
	m.fs.AddFlagSet(m.prometheusInstanceOptions.FlagSet(prefix))
	return m.fs
}

// MustCreateMetrics sets up Prometheus based metrics and starts an HTTP server
// for serving metrics.
func (m *MetricsOptions) MustCreateMetrics(ctx context.Context, logger *zap.SugaredLogger) *metrics.Metrics {
	return m.MustCreateWorkerMetrics(ctx, logger, SdkOptions{}, ScenarioID{}, "")
}

// MustCreateWorkerMetrics creates metrics with worker metadata for the /info endpoint.
// This is used by worker processes that need to expose worker information.
func (m *MetricsOptions) MustCreateWorkerMetrics(
	ctx context.Context,
	logger *zap.SugaredLogger,
	sdkOpts SdkOptions,
	scenarioOpts ScenarioID,
	buildID string,
) *metrics.Metrics {
	registry := prometheus.NewRegistry()
	var server *http.Server
	if m.PrometheusListenAddress != "" {
		procCollector, err := metrics.NewProcessCollector()
		if err != nil {
			logger.Fatalf("Unable to setup process collector for Prometheus: %v", err)
		}
		registry.MustRegister(procCollector)
		server = m.mustInitPrometheusServer(logger, registry, sdkOpts, scenarioOpts, buildID)
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

func (m *MetricsOptions) mustInitPrometheusServer(
	logger *zap.SugaredLogger,
	registry *prometheus.Registry,
	sdkOpts SdkOptions,
	scenarioOpts ScenarioID,
	buildID string,
) *http.Server {
	address := m.PrometheusListenAddress
	handlerPath := m.PrometheusHandlerPath
	if handlerPath == "" {
		handlerPath = "/metrics"
	}

	handler := http.NewServeMux()
	handler.Handle(handlerPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	// Add /info endpoint
	handler.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := InfoResponse{
			SdkOptions:      sdkOpts,
			ScenarioOptions: scenarioOpts,
			BuildID:         buildID,
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			logger.Errorf("failed to encode /info response: %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	})

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
	p.fs.StringVar(&p.ConfigPath, prefix+"prom-instance-config", "", "Start a local Prometheus instance with the specified config file (default: prom-config.yml)")
	p.fs.Lookup(prefix + "prom-instance-config").NoOptDefVal = "prom-config.yml"
	p.fs.BoolVar(&p.Snapshot, "prom-snapshot", false, "Create a TSDB snapshot on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsPath, prefix+"prom-export-worker-metrics", "", "Export worker process metrics to the specified file on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsJob, prefix+"prom-export-worker-job", "omes-worker", "Name of the worker job to export metrics for")
	p.fs.DurationVar(&p.ExportMetricsStep, prefix+"prom-export-metrics-step", 15*time.Second, "Step interval to sample timeseries metrics")
	return p.fs
}
