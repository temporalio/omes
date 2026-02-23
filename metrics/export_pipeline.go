package metrics

import (
	"context"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// JobWorkerApp is the Prometheus scrape job name for worker SDK/application metrics.
	JobWorkerApp = "omes_worker_app"
	// JobWorkerProcess is the Prometheus scrape job name for worker process sidecar metrics.
	JobWorkerProcess = "omes_worker_process"
	// DefaultStepInterval is the default step interval to scrape Prometheus if none was provided
	DefaultStepInterval = 5 * time.Second
)

// RawSample is a transport-neutral metrics sample used by the export pipeline.
type RawSample struct {
	Timestamp time.Time
	Metric    string
	Value     float64
	Labels    map[string]string
}

// PromQueryConfig configures a Prometheus query-range export pass.
type PromQueryConfig struct {
	Address string
	Start   time.Time
	End     time.Time
	Step    time.Duration
}

// ExportStats summarizes how many samples were read and handed to the callback.
type ExportStats struct {
	SamplesRead    int64
	SamplesHandled int64
}

// SampleHandler receives each raw sample from Prometheus.
type SampleHandler func(context.Context, RawSample) error

// MetricLineMetadata carries run metadata to stamp onto mapped MetricLine rows.
type MetricLineMetadata struct {
	Scenario   string
	RunID      string
	RunFamily  string
	SDKVersion string
	BuildID    string
	Language   string
}

// ExportFromPrometheus queries Prometheus over the configured window and invokes
// the callback for each sample from worker app/process jobs.
func ExportFromPrometheus(ctx context.Context, cfg PromQueryConfig, handle SampleHandler) (ExportStats, error) {
	if err := validatePromQueryConfig(&cfg); err != nil {
		return ExportStats{}, err
	}
	if handle == nil {
		return ExportStats{}, fmt.Errorf("sample handler is required")
	}

	client, err := api.NewClient(api.Config{Address: cfg.Address})
	if err != nil {
		return ExportStats{}, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	query := fmt.Sprintf(`{job=~"%s|%s"}`, JobWorkerApp, JobWorkerProcess)
	result, _, err := v1.NewAPI(client).QueryRange(ctx, query, v1.Range{
		Start: cfg.Start,
		End:   cfg.End,
		Step:  cfg.Step,
	})
	if err != nil {
		return ExportStats{}, fmt.Errorf("failed Prometheus query_range: %w", err)
	}

	matrix, ok := result.(model.Matrix)
	if !ok {
		return ExportStats{}, fmt.Errorf("unexpected Prometheus query result type %T", result)
	}

	var stats ExportStats
	for _, series := range matrix {
		metricName := string(series.Metric[model.MetricNameLabel])
		if metricName == "" {
			continue
		}

		labels := make(map[string]string, len(series.Metric))
		for key, value := range series.Metric {
			if key == model.MetricNameLabel {
				continue
			}
			labels[string(key)] = string(value)
		}

		for _, sample := range series.Values {
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			s := RawSample{
				Timestamp: sample.Timestamp.Time(),
				Metric:    metricName,
				Value:     float64(sample.Value),
				Labels:    labels,
			}
			stats.SamplesRead++
			if err := handle(ctx, s); err != nil {
				return stats, err
			}
			stats.SamplesHandled++
		}
	}

	return stats, nil
}

// ExportMetricLineParquetFromPrometheus is a convenience path for parquet
// export flow: create parquet handler, export from Prometheus, close writer.
func ExportMetricLineParquetFromPrometheus(
	ctx context.Context,
	queryCfg PromQueryConfig,
	outputPath string,
	meta MetricLineMetadata,
) (ExportStats, error) {
	if outputPath == "" {
		return ExportStats{}, fmt.Errorf("output path is required")
	}

	writer, err := newParquetMetricLineWriter(outputPath)
	if err != nil {
		return ExportStats{}, err
	}

	handler := func(ctx context.Context, sample RawSample) error {
		if sample.Metric == "" || math.IsNaN(sample.Value) {
			return nil
		}
		return writer.writeMetricLine(MetricLine{
			Timestamp:  sample.Timestamp,
			Metric:     sample.Metric,
			Value:      sample.Value,
			SDKVersion: meta.SDKVersion,
			BuildID:    meta.BuildID,
			Language:   meta.Language,
			Scenario:   meta.Scenario,
			RunID:      meta.RunID,
			RunFamily:  meta.RunFamily,
		})
	}
	stats, exportErr := ExportFromPrometheus(ctx, queryCfg, handler)
	closeErr := writer.Close()
	if exportErr != nil {
		return stats, fmt.Errorf("failed to export parquet metrics: %w", exportErr)
	}
	if closeErr != nil {
		return stats, fmt.Errorf("failed to close parquet export: %w", closeErr)
	}
	return stats, nil
}

type parquetMetricLineWriter struct {
	file   *os.File
	writer *parquet.GenericWriter[MetricLine]
	buffer []MetricLine
	closed bool
}

func newParquetMetricLineWriter(outputPath string) (*parquetMetricLineWriter, error) {
	file, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	return &parquetMetricLineWriter{
		file:   file,
		writer: parquet.NewGenericWriter[MetricLine](file, parquet.Compression(&zstd.Codec{})),
		buffer: make([]MetricLine, 0, parquetBatchSize),
	}, nil
}

func (w *parquetMetricLineWriter) writeMetricLine(line MetricLine) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}
	w.buffer = append(w.buffer, line)
	if len(w.buffer) < parquetBatchSize {
		return nil
	}
	return w.flush()
}

func (w *parquetMetricLineWriter) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true

	var outErr error
	if err := w.flush(); err != nil {
		outErr = err
	}
	if err := w.writer.Close(); err != nil && outErr == nil {
		outErr = fmt.Errorf("failed to close parquet writer: %w", err)
	}
	if err := w.file.Close(); err != nil && outErr == nil {
		outErr = fmt.Errorf("failed to close output file: %w", err)
	}
	return outErr
}

func (w *parquetMetricLineWriter) flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	if _, err := w.writer.Write(w.buffer); err != nil {
		return fmt.Errorf("failed to write parquet batch: %w", err)
	}
	w.buffer = w.buffer[:0]
	return nil
}

func validatePromQueryConfig(cfg *PromQueryConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("prometheus address is required")
	}
	if cfg.Start.IsZero() {
		return fmt.Errorf("start time is required")
	}
	if cfg.End.IsZero() {
		cfg.End = time.Now() // fallback to current time
	}
	if !cfg.Start.Before(cfg.End) {
		return fmt.Errorf("invalid time range: start=%v end=%v", cfg.Start, cfg.End)
	}
	if cfg.Step <= 0 {
		cfg.Step = DefaultStepInterval
	}
	return nil
}
