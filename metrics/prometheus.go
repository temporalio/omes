package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
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
	// Path to export worker metrics on shutdown.
	// Includes process metrics (CPU/memory), task latencies, polling metrics, and throughput.
	// If empty, no export will be performed.
	ExportWorkerMetricsPath string
	// Worker job to export.
	ExportWorkerMetricsJob string
	// Step interval when sampling timeseries metrics for export.
	// If not provided a default interval of 15s will be used.
	// (only used if ExportWorkerMetricsPath is provided)
	ExportMetricsStep time.Duration
	// Address to fetch worker info from during export (e.g., "localhost:9091").
	// If provided, /info will be fetched and used to populate build_id in export.
	ExportWorkerInfoAddress string
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

type PrometheusInstance struct {
	opts *PrometheusInstanceOptions
	// Derived prometheus command based on given options
	prometheusCmd *exec.Cmd
	// Prometheus API
	api v1.API
	// Time when the instance started
	startTime time.Time
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

func (i *PrometheusInstance) Shutdown(ctx context.Context, logger *zap.SugaredLogger, scenario, runID string) {
	// Export worker metrics if configured
	if i.opts.ExportWorkerMetricsPath != "" {
		err := i.exportWorkerMetrics(ctx, logger, scenario, runID)
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

func (i *PrometheusInstance) exportWorkerMetrics(ctx context.Context, logger *zap.SugaredLogger, scenario, runID string) error {
	start, end, err := i.getTimeRange()
	if err != nil {
		return fmt.Errorf("failed to get time range: %w", err)
	}

	file, err := os.Create(i.opts.ExportWorkerMetricsPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Fetch worker info if address is configured
	var workerInfo *WorkerInfo
	if i.opts.ExportWorkerInfoAddress != "" {
		workerInfo, err = fetchWorkerInfo(i.opts.ExportWorkerInfoAddress)
		if err != nil {
			logger.Warnf("Failed to fetch worker info from %s: %v (continuing without worker metadata)", i.opts.ExportWorkerInfoAddress, err)
		} else {
			logger.Infof("Fetched worker info: sdk_version=%s, build_id=%s", workerInfo.SDKVersion, workerInfo.BuildID)
		}
	}

	queries := i.buildMetricQueries()

	return i.exportWorkerMetricsParquet(ctx, file, queries, start, end, workerInfo, logger, scenario, runID)
}

func (i *PrometheusInstance) buildMetricQueries() []metricQuery {
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
		{"workflow_task_execution_latency_seconds", "temporal_workflow_task_execution_latency"},
		{"workflow_task_schedule_to_start_latency_seconds", "temporal_workflow_task_schedule_to_start_latency"},
		{"workflow_endtoend_latency_seconds", "temporal_workflow_endtoend_latency"},
		{"activity_execution_latency_seconds", "temporal_activity_execution_latency"},
		{"activity_schedule_to_start_latency_seconds", "temporal_activity_schedule_to_start_latency"},
	}
	for _, m := range histogramMetrics {
		queries = append(queries, histogramQuantileQuery(m.name, m.promName, job, 0.50, "p50"))
		queries = append(queries, histogramQuantileQuery(m.name, m.promName, job, 0.99, "p99"))
	}

	return queries
}

const parquetBatchSize = 5000

func (i *PrometheusInstance) exportWorkerMetricsParquet(
	ctx context.Context,
	file *os.File,
	queries []metricQuery,
	start, end time.Time,
	workerInfo *WorkerInfo,
	logger *zap.SugaredLogger,
	scenario, runID string,
) (err error) {
	writer := parquet.NewGenericWriter[MetricLine](file,
		parquet.Compression(&zstd.Codec{}),
	)
	defer func() {
		if closeErr := writer.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close parquet writer: %w", closeErr)
		}
	}()

	// Extract build_id from worker info (empty string if not available)
	var buildID string
	if workerInfo != nil {
		buildID = workerInfo.BuildID
	}

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
				BuildID:   buildID,
				Scenario:  scenario,
				RunID:     runID,
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

// WorkerInfo represents the response from the /info endpoint.
// Only contains fields that run-scenario doesn't already know.
type WorkerInfo struct {
	SDKVersion string `json:"sdk_version"`
	BuildID    string `json:"build_id"`
}

// fetchWorkerInfo fetches worker metadata from the /info endpoint.
func fetchWorkerInfo(address string) (*WorkerInfo, error) {
	resp, err := http.Get("http://" + address + "/info")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d from /info", resp.StatusCode)
	}

	var info WorkerInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("failed to decode /info response: %w", err)
	}
	return &info, nil
}

// MetricLine represents a single metric data point.
type MetricLine struct {
	Timestamp           time.Time `parquet:"timestamp,timestamp"`
	Metric              string    `parquet:"metric,dict"`
	Value               float64   `parquet:"value"`
	Environment         string    `parquet:"environment,dict"`
	BuildID             string    `parquet:"build_id,dict"`
	Scenario            string    `parquet:"scenario,dict"`
	RunID               string    `parquet:"run_id,dict"`
	RunConfigProfile    string    `parquet:"run_profile,dict"`
	WorkerConfigProfile string    `parquet:"worker_profile,dict"`
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
