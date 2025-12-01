package clioptions

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
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
	// Path to export worker metrics (CPU/memory) as JSON on shutdown.
	// If empty, no export will be performed.
	ExportWorkerMetricsPath string
	// Format to export worker metrics, default is parquet.
	ExportWorkerMetricsFormat string
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
	p.fs.StringVar(&p.ExportWorkerMetricsPath, prefix+"prom-export-worker-metrics", "", "Export worker process metrics to the specified file on shutdown")
	p.fs.StringVar(&p.ExportWorkerMetricsFormat, prefix+"prom-export-metrics-format", "parquet", "Format to export worker metrics (allowed values: 'parquet' or 'jsonl')")
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
	// Time when the instance started
	startTime time.Time
}

func (p *PrometheusInstanceOptions) IsConfigured() bool {
	return p.Address != "" && p.ConfigPath != ""
}

func (p *PrometheusInstanceOptions) StartPrometheusInstance(ctx context.Context, logger *zap.SugaredLogger) *PrometheusInstance {
	cmd := exec.CommandContext(ctx, "prometheus",
		"--config.file="+p.ConfigPath,
		"--web.enable-admin-api", // Required for snapshot API
		"--web.listen-address="+p.Address,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logger.Fatalf("Failed to start Prometheus with config %s: %v", p.ConfigPath, err)
	}

	logger.Infof("Started local Prometheus instance with config: %s (PID: %d)", p.ConfigPath, cmd.Process.Pid)

	client, err := api.NewClient(api.Config{Address: "http://" + p.Address})
	if err != nil {
		logger.Fatalf("Failed to create Prometheus client: %v", err)
	}

	instance := &PrometheusInstance{
		opts:          p,
		prometheusCmd: cmd,
		api:           v1.NewAPI(client),
		startTime:     time.Now(),
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
		err := i.exportWorkerMetrics(ctx, logger)
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

func (i *PrometheusInstance) exportWorkerMetrics(ctx context.Context, logger *zap.SugaredLogger) error {
	format := i.opts.ExportWorkerMetricsFormat
	if format != "parquet" && format != "jsonl" {
		return fmt.Errorf("unrecognized export format, expected either 'parquet' or 'jsonl', got %s", format)
	}

	start, end, err := i.getTimeRange()
	if err != nil {
		return fmt.Errorf("failed to get time range: %w", err)
	}

	file, err := os.Create(i.opts.ExportWorkerMetricsPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	queries := i.buildMetricQueries()

	if format == "jsonl" {
		return i.exportWorkerMetricsJSONL(ctx, file, queries, start, end, logger)
	}
	return i.exportWorkerMetricsParquet(ctx, file, queries, start, end, logger)
}

func (i *PrometheusInstance) buildMetricQueries() []metricQuery {
	job := i.opts.ExportWorkerMetricsJob

	queries := []metricQuery{
		{"process_cpu_percent", fmt.Sprintf(`process_cpu_percent{job="%s"}`, job)},
		{"process_memory_bytes", fmt.Sprintf(`process_resident_memory_bytes{job="%s"}`, job)},
		{"process_memory_percent", fmt.Sprintf(`process_memory_percent{job="%s"}`, job)},
	}

	return queries
}

func (i *PrometheusInstance) exportWorkerMetricsJSONL(
	ctx context.Context,
	file *os.File,
	queries []metricQuery,
	start, end time.Time,
	logger *zap.SugaredLogger,
) error {
	encoder := json.NewEncoder(file)
	metricsWithNaN := make(map[string]int)

	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("export cancelled: %w", err)
		}

		dataPoints, err := i.queryPrometheusRange(ctx, q.query, start, end, i.opts.ExportMetricsStep)
		if err != nil {
			return fmt.Errorf("failed to query %s: %w", q.name, err)
		}

		for _, dp := range dataPoints {
			if math.IsNaN(dp.Value) {
				metricsWithNaN[q.name]++
				continue
			}
			m := MetricLine{
				Timestamp: dp.Timestamp,
				Metric:    q.name,
				Value:     dp.Value,
			}
			if err := encoder.Encode(m); err != nil {
				return fmt.Errorf("failed to write metric %s: %w", q.name, err)
			}
		}
	}

	if len(metricsWithNaN) > 0 {
		logger.Warnf("Skipped NaN values for metrics (no data in time range): %v", metricsWithNaN)
	}

	return nil
}

const parquetBatchSize = 5000

func (i *PrometheusInstance) exportWorkerMetricsParquet(
	ctx context.Context,
	file *os.File,
	queries []metricQuery,
	start, end time.Time,
	logger *zap.SugaredLogger,
) (err error) {
	writer := parquet.NewGenericWriter[MetricLine](file,
		parquet.Compression(&zstd.Codec{}),
	)
	defer func() {
		if closeErr := writer.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close parquet writer: %w", closeErr)
		}
	}()

	metricsWithNaN := make(map[string]int)
	metrics := make([]MetricLine, 0, parquetBatchSize)

	flushBatch := func() error {
		if len(metrics) == 0 {
			return nil
		}
		if _, err := writer.Write(metrics); err != nil {
			return fmt.Errorf("failed to write parquet batch: %w", err)
		}
		metrics = metrics[:0]
		return nil
	}

	for _, q := range queries {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("export cancelled: %w", err)
		}

		dataPoints, err := i.queryPrometheusRange(ctx, q.query, start, end, i.opts.ExportMetricsStep)
		if err != nil {
			return fmt.Errorf("failed to query %s: %w", q.name, err)
		}

		for _, dp := range dataPoints {
			if math.IsNaN(dp.Value) {
				metricsWithNaN[q.name]++
				continue
			}
			metrics = append(metrics, MetricLine{
				Timestamp: dp.Timestamp,
				Metric:    q.name,
				Value:     dp.Value,
			})
			if len(metrics) >= parquetBatchSize {
				if err := flushBatch(); err != nil {
					return err
				}
			}
		}
	}

	// Flush remaining metrics
	if err := flushBatch(); err != nil {
		return err
	}

	if len(metricsWithNaN) > 0 {
		logger.Warnf("Skipped NaN values for metrics (no data in time range): %v", metricsWithNaN)
	}

	return nil
}

func (i *PrometheusInstance) getTimeRange() (start, end time.Time, err error) {
	// Use the instance start time and current time to only export metrics from this run
	start = i.startTime
	end = time.Now()

	if start.IsZero() || end.IsZero() || !start.Before(end) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid time range (start=%v, end=%v)", start, end)
	}

	return start, end, nil
}

// MetricLine represents a single metric data point in JSONL format.
type MetricLine struct {
	Timestamp time.Time `json:"timestamp" parquet:"timestamp,timestamp"`
	Metric    string    `json:"metric" parquet:"metric,dict"`
	Value     float64   `json:"value" parquet:"value"`
}

type metricQuery struct {
	name  string
	query string
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
