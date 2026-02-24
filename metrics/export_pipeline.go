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
	"go.uber.org/zap"
)

const (
	// JobWorkerApp is the Prometheus scrape job name for worker SDK/application metrics.
	JobWorkerApp = "omes_worker_app"
	// JobWorkerProcess is the Prometheus scrape job name for worker process sidecar metrics.
	JobWorkerProcess = "omes_worker_process"
	// DefaultStepInterval is the default step interval to scrape Prometheus if none was provided
	DefaultStepInterval = 5 * time.Second
	// exportPipelineParquetBatchSize controls parquet flush size for this export path.
	exportPipelineParquetBatchSize = 5000
)

// RawSample is a transport-neutral metrics sample used by the export pipeline.
type RawSample struct {
	Timestamp time.Time
	Metric    string
	Value     float64
	// Labels is shared across samples for a single series. Treat as read-only.
	Labels map[string]string
}

// PromQuery defines one Prometheus query range expression and optional output name.
// If Name is empty, ExportFromPrometheus will fall back to the series __name__ label.
type PromQuery struct {
	Name  string
	Query string
}

// PromQueryConfig configures a Prometheus query-range export pass.
type PromQueryConfig struct {
	Address string
	Start   time.Time
	End     time.Time
	Step    time.Duration
	Queries []PromQuery
	Logger  *zap.SugaredLogger
}

// SampleHandler receives each raw sample from Prometheus. Handlers must treat
// RawSample.Labels as read-only.
type SampleHandler func(context.Context, RawSample) error

// MetricLineMetadata carries run metadata to stamp onto mapped MetricLine rows.
type MetricLineMetadata struct {
	Scenario            string
	RunID               string
	RunFamily           string
	SDKVersion          string
	BuildID             string
	Language            string
	Environment         string
	RunConfigProfile    string
	WorkerConfigProfile string
}

// ExportFromPrometheus queries Prometheus over the configured window and invokes
// the callback for each sample from worker app/process jobs.
func ExportFromPrometheus(ctx context.Context, cfg PromQueryConfig, handle SampleHandler) error {
	if handle == nil {
		return fmt.Errorf("sample handler is required")
	}
	applyPromQueryDefaults(&cfg)
	if err := validatePromQueryConfig(&cfg); err != nil {
		return err
	}

	client, err := api.NewClient(api.Config{Address: cfg.Address})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	queries := cfg.Queries
	if len(queries) == 0 {
		// Fallback query, get all raw metrics from jobs
		queries = []PromQuery{
			{Query: fmt.Sprintf(`{job=~"%s|%s"}`, JobWorkerApp, JobWorkerProcess)},
		}
	}

	promAPI := v1.NewAPI(client)
	skippedUnnamedSeries := 0

	for _, q := range queries {
		result, _, err := promAPI.QueryRange(ctx, q.Query, v1.Range{
			Start: cfg.Start,
			End:   cfg.End,
			Step:  cfg.Step,
		})
		if err != nil {
			return fmt.Errorf("failed Prometheus query_range for %q: %w", q.Name, err)
		}

		matrix, ok := result.(model.Matrix)
		if !ok {
			return fmt.Errorf("unexpected Prometheus query result type %T", result)
		}

		for _, series := range matrix {
			metricName := q.Name
			if metricName == "" {
				metricName = string(series.Metric[model.MetricNameLabel])
			}
			if metricName == "" {
				skippedUnnamedSeries++
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
					return err
				}
				s := RawSample{
					Timestamp: sample.Timestamp.Time(),
					Metric:    metricName,
					Value:     float64(sample.Value),
					Labels:    labels,
				}
				if err := handle(ctx, s); err != nil {
					return err
				}
			}
		}
	}
	if skippedUnnamedSeries > 0 && cfg.Logger != nil {
		cfg.Logger.Warnf(
			"Skipped %d Prometheus series due to empty metric name",
			skippedUnnamedSeries,
		)
	}

	return nil
}

// ExportMetricLineParquetFromPrometheus is a convenience path for parquet
// export flow: create parquet handler, export from Prometheus, close writer.
func ExportMetricLineParquetFromPrometheus(
	ctx context.Context,
	queryCfg PromQueryConfig,
	outputPath string,
	meta MetricLineMetadata,
) error {
	if outputPath == "" {
		return fmt.Errorf("output path is required")
	}

	writer, err := newParquetMetricLineWriter(outputPath)
	if err != nil {
		return err
	}

	handler := func(ctx context.Context, sample RawSample) error {
		if sample.Metric == "" || math.IsNaN(sample.Value) {
			return nil
		}

		if err := writer.writeMetricLine(MetricLine{
			Timestamp:           sample.Timestamp,
			Metric:              sample.Metric,
			Value:               sample.Value,
			SDKVersion:          meta.SDKVersion,
			BuildID:             meta.BuildID,
			Language:            meta.Language,
			Environment:         meta.Environment,
			Scenario:            meta.Scenario,
			RunID:               meta.RunID,
			RunFamily:           meta.RunFamily,
			RunConfigProfile:    meta.RunConfigProfile,
			WorkerConfigProfile: meta.WorkerConfigProfile,
		}); err != nil {
			return err
		}
		return nil
	}
	exportErr := ExportFromPrometheus(ctx, queryCfg, handler)
	closeErr := writer.Close()
	if exportErr != nil {
		return fmt.Errorf("failed to export parquet metrics: %w", exportErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close parquet export: %w", closeErr)
	}
	return nil
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
		buffer: make([]MetricLine, 0, exportPipelineParquetBatchSize),
	}, nil
}

func (w *parquetMetricLineWriter) writeMetricLine(line MetricLine) error {
	if w.closed {
		return fmt.Errorf("writer is closed")
	}
	w.buffer = append(w.buffer, line)
	if len(w.buffer) < exportPipelineParquetBatchSize {
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

func applyPromQueryDefaults(cfg *PromQueryConfig) {
	if cfg.End.IsZero() {
		cfg.End = time.Now()
	}
	if cfg.Step <= 0 {
		cfg.Step = DefaultStepInterval
	}
	if len(cfg.Queries) == 0 {
		// Fallback query, get all raw metrics from jobs
		cfg.Queries = []PromQuery{
			{Query: fmt.Sprintf(`{job=~"%s|%s"}`, JobWorkerApp, JobWorkerProcess)},
		}
	}
}

func validatePromQueryConfig(cfg *PromQueryConfig) error {
	if cfg.Address == "" {
		return fmt.Errorf("prometheus address is required")
	}
	if cfg.Start.IsZero() {
		return fmt.Errorf("start time is required")
	}
	if !cfg.Start.Before(cfg.End) {
		return fmt.Errorf("invalid time range: start=%v end=%v", cfg.Start, cfg.End)
	}
	for idx, q := range cfg.Queries {
		if q.Query == "" {
			return fmt.Errorf("queries[%d].query is required", idx)
		}
	}

	return nil
}
