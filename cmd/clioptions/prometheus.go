package clioptions

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
)

type PrometheusInstanceOptions struct {
	// Address to run the Prometheus instance
	Address string
	// Path to Prometheus config file for starting a local Prometheus instance.
	// If empty, no local Prometheus will be started.
	ConfigPath string
	// If true, create a TSDB snapshot on shutdown.
	Snapshot bool
	// Path to export worker metrics as JSON on shutdown.
	// Includes process metrics (CPU/memory), task latencies, polling metrics, and throughput.
	// If empty, no export will be performed.
	ExportWorkerMetricsPath string
	// Worker job to export.
	ExportWorkerMetricsJob string
	// Step interval when sampling timeseries metrics for the JSON output format.
	// If not provided a default interval of 15s will be used.
	// (only used if ExportWorkerMetricsPath is provided)
	ExportMetricsStep time.Duration

	fs *pflag.FlagSet
}

func (p *PrometheusInstanceOptions) FlagSet(prefix string) *pflag.FlagSet {
	p.fs = pflag.NewFlagSet(prefix+"prom_instance_options", pflag.ExitOnError)
	p.fs.StringVar(&p.Address, prefix+"prom-instance-addr", "", "Prometheus instance address")
	p.fs.StringVar(&p.ConfigPath, prefix+"prom-instance-config", "", "Start a local Prometheus instance with the specified config file (default: prom-config.yml)")
	p.fs.Lookup(prefix + "prom-instance-config").NoOptDefVal = "prom-config.yml"
	p.fs.BoolVar(&p.Snapshot, prefix+"prom-snapshot", false, "Create a TSDB snapshot on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsPath, prefix+"prom-export-worker-metrics", "", "Export worker process metrics as JSONL to the specified file on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsJob, prefix+"prom-export-worker-job", "omes-worker", "Name of the worker job to export metrics for")
	p.fs.DurationVar(&p.ExportMetricsStep, prefix+"prom-export-metrics-step", 15*time.Second, "Step interval to sample timeseries metrics")
	return p.fs
}

type PrometheusInstance struct {
	opts *PrometheusInstanceOptions
	// Derived prometheus command based on given options
	prometheusCmd *exec.Cmd
	// Prometheus API
	api v1.API
}

func (p *PrometheusInstanceOptions) IsConfigured() bool {
	return p.Address != "" && p.ConfigPath != ""
}

func (p *PrometheusInstanceOptions) StartPrometheusInstance(ctx context.Context, logger *zap.SugaredLogger) *PrometheusInstance {
	cmd := exec.CommandContext(ctx, "prometheus",
		"--config.file="+p.ConfigPath,
		"--web.enable-admin-api", // Required for snapshot API
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logger.Fatalf("Failed to start Prometheus with config %s: %v", p.ConfigPath, err)
	}

	logger.Infof("Started local Prometheus instance with config: %s (PID: %d)", p.ConfigPath, cmd.Process.Pid)

	client, err := api.NewClient(api.Config{Address: p.Address})
	if err != nil {
		logger.Fatalf("Failed to create Prometheus client: %v", err)
	}

	instance := &PrometheusInstance{
		opts:          p,
		prometheusCmd: cmd,
		api:           v1.NewAPI(client),
	}

	if err := instance.waitForReady(ctx, 30*time.Second); err != nil {
		logger.Fatalf("Prometheus failed to become ready: %v", err)
	}

	return instance
}

func (p *PrometheusInstance) waitForReady(ctx context.Context, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		_, err := p.api.Runtimeinfo(ctx)
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for Prometheus, latest err: %v", err)
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (i *PrometheusInstance) Shutdown(ctx context.Context, logger *zap.SugaredLogger) {
	// Export worker metrics as JSON if configured
	if i.opts.ExportWorkerMetricsPath != "" {
		err := i.exportWorkerMetrics(ctx)
		if err != nil {
			logger.Errorf("Failed to export worker metrics: %v", err)
		} else {
			logger.Infof("Worker metrics exported to: %s", i.opts.ExportWorkerMetricsPath)
		}
	}
	// Create a snapshot if configured
	if i.opts.Snapshot {
		snapshotName, err := i.createPrometheusSnapshot(ctx)
		if err != nil {
			logger.Errorf("Failed to snapshot Prometheus instance: %v", err)
		} else {
			logger.Infof("Prometheus snapshot captured, named: %s", snapshotName)
		}
	}

	err := i.prometheusCmd.Process.Signal(os.Interrupt)
	if err != nil {
		logger.Errorf("Failed to signal interrupt to Prometheus instance process: %v", err)
		return
	}

	done := make(chan error, 1)
	go func() { done <- i.prometheusCmd.Wait() }()

	select {
	case <-done:
		logger.Info("Prometheus shut down gracefully")
	case <-time.After(10 * time.Second):
		logger.Warn("Prometheus didn't shut down gracefully, killing")
		i.prometheusCmd.Process.Kill()
	}
}

func (i *PrometheusInstance) exportWorkerMetrics(ctx context.Context) error {
	start, end, err := i.getTimeRange(ctx)
	if err != nil {
		return fmt.Errorf("failed to get time range: %w", err)
	}

	file, err := os.Create(i.opts.ExportWorkerMetricsPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)

	job := i.opts.ExportWorkerMetricsJob

	// Process metrics
	queries := []metricQuery{
		{"process_cpu_percent", fmt.Sprintf(`process_cpu_percent{job="%s"}`, job)},
		{"process_memory_bytes", fmt.Sprintf(`process_resident_memory_bytes{job="%s"}`, job)},
		{"process_memory_percent", fmt.Sprintf(`process_memory_percent{job="%s"}`, job)},
	}

	// Polling/capacity metrics
	gaugeMetrics := []struct{ name, promName string }{
		{"num_pollers", "temporal_num_pollers"},
		{"worker_task_slots_available", "temporal_worker_task_slots_available"},
		{"worker_task_slots_used", "temporal_worker_task_slots_used"},
	}
	for _, m := range gaugeMetrics {
		queries = append(queries, gaugeQuery(m.name, m.promName, job))
	}

	// Latency histogram metrics
	histogramMetrics := []struct{ name, promName string }{
		{"workflow_task_execution_latency", "temporal_workflow_task_execution_latency"},
		{"workflow_task_schedule_to_start_latency", "temporal_workflow_task_schedule_to_start_latency"},
		{"workflow_endtoend_latency", "temporal_workflow_endtoend_latency"},
		{"activity_execution_latency", "temporal_activity_execution_latency"},
		{"activity_schedule_to_start_latency", "temporal_activity_schedule_to_start_latency"},
	}
	for _, m := range histogramMetrics {
		queries = append(queries, histogramQuantileQuery(m.name, m.promName, job, 0.50, "p50"))
		queries = append(queries, histogramQuantileQuery(m.name, m.promName, job, 0.99, "p99"))
	}

	for _, q := range queries {
		dataPoints, err := i.queryPrometheusRange(ctx, q.query, start, end, i.opts.ExportMetricsStep)
		if err != nil {
			return fmt.Errorf("failed to query %s: %w", q.name, err)
		}

		for _, dp := range dataPoints {
			err := encoder.Encode(MetricLine{
				Timestamp: dp.Timestamp,
				Metric:    q.name,
				Value:     dp.Value,
			})
			if err != nil {
				return fmt.Errorf("failed to write prometheus metric %s: %w", q.name, err)
			}
		}
	}

	return nil
}

func (i *PrometheusInstance) getTimeRange(ctx context.Context) (start, end time.Time, err error) {
	tsdb, err := i.api.TSDB(ctx)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	start = time.UnixMilli(int64(tsdb.HeadStats.MinTime))
	end = time.UnixMilli(int64(tsdb.HeadStats.MaxTime))

	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time range (start=%v, end=%v)", start, end)
	}

	return start, end, nil
}

// MetricLine represents a single metric data point in JSONL format.
type MetricLine struct {
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
}

type metricQuery struct {
	name  string
	query string
}

// Query builder helpers for different metric types.

func gaugeQuery(name, promName, job string) metricQuery {
	return metricQuery{name, fmt.Sprintf(`sum(%s{job="%s"})`, promName, job)}
}

func histogramQuantileQuery(name, promName, job string, quantile float64, percentile string) metricQuery {
	return metricQuery{
		name + "_" + percentile,
		fmt.Sprintf(`histogram_quantile(%.2f, sum(rate(%s_bucket{job="%s"}[1m])) by (le))`, quantile, promName, job),
	}
}

// MetricDataPoint represents a single data point with timestamp and value (internal use).
type MetricDataPoint struct {
	Timestamp time.Time
	Value     float64
}

func (i *PrometheusInstance) queryPrometheusRange(ctx context.Context, query string, start, end time.Time, step time.Duration) ([]MetricDataPoint, error) {
	result, _, err := i.api.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})
	if err != nil {
		return nil, err
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return nil, fmt.Errorf("unexpected result type")
	}

	var dataPoints []MetricDataPoint
	for _, series := range matrix {
		for _, sample := range series.Values {
			dataPoints = append(dataPoints, MetricDataPoint{
				Timestamp: sample.Timestamp.Time(),
				Value:     float64(sample.Value),
			})
		}
	}
	return dataPoints, nil
}

func (i *PrometheusInstance) createPrometheusSnapshot(ctx context.Context) (string, error) {
	result, err := i.api.Snapshot(ctx, false)
	if err != nil {
		return "", fmt.Errorf("failed to create snapshot: %w", err)
	}
	return result.Name, nil
}
