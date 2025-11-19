package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/process"
)

const DefaultSampleInterval = 5 * time.Second

// Sample represents a single resource measurement at a point in time.
type Sample struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUPercent  float64   `json:"cpuPercent"`
	MemoryBytes uint64    `json:"memoryBytes"`
}

// Summary contains aggregate statistics over the course of the run.
type Summary struct {
	PeakCPUPercent     float64 `json:"peakCpuPercent"`
	AverageCPUPercent  float64 `json:"averageCpuPercent"`
	PeakMemoryBytes    uint64  `json:"peakMemoryBytes"`
	AverageMemoryBytes uint64  `json:"averageMemoryBytes"`
}

// Metrics is the top-level structure written to the JSON file.
type Metrics struct {
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime"`
	Samples   []Sample  `json:"samples"`
	Summary   Summary   `json:"summary"`
}

// Tracker tracks CPU and memory usage during worker execution.
type Tracker struct {
	samples    []Sample
	mu         sync.RWMutex
	interval   time.Duration
	cancel     context.CancelFunc
	done       chan struct{}
	proc       *process.Process
	runID      string
	outputPath string
	startTime  time.Time
}

// NewTracker creates a new resource tracker.
func NewTracker(runID string, outputDir string) (*Tracker, error) {
	proc, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, fmt.Errorf("failed to get process: %w", err)
	}

	// Build full path with filename
	outputPath := filepath.Join(outputDir, fmt.Sprintf("worker-metrics-%s.json", runID))

	return &Tracker{
		samples:    make([]Sample, 0, 100),
		interval:   DefaultSampleInterval,
		done:       make(chan struct{}),
		proc:       proc,
		runID:      runID,
		outputPath: outputPath,
		startTime:  time.Now(),
	}, nil
}

// Start begins collecting resource usage samples in the background.
func (t *Tracker) Start(ctx context.Context) {
	ctx, t.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(t.interval)
		defer ticker.Stop()
		defer close(t.done)

		// Take an initial sample immediately
		t.takeSample()

		for {
			select {
			case <-ctx.Done():
				// Take a final sample before stopping
				t.takeSample()
				return
			case <-ticker.C:
				t.takeSample()
			}
		}
	}()
}

// Stop stops collecting resource usage samples and writes the JSON file.
func (t *Tracker) Stop() error {
	if t.cancel != nil {
		t.cancel()
	}
	<-t.done
	return t.writeMetrics()
}

// takeSample collects a single resource usage measurement.
func (t *Tracker) takeSample() {
	var sample Sample
	sample.Timestamp = time.Now()

	// Get CPU usage (percentage of one core)
	if cpuPercent, err := t.proc.CPUPercent(); err == nil {
		sample.CPUPercent = cpuPercent
	}

	// Get memory usage (RSS - Resident Set Size)
	if memInfo, err := t.proc.MemoryInfo(); err == nil {
		sample.MemoryBytes = memInfo.RSS
	}

	t.mu.Lock()
	t.samples = append(t.samples, sample)
	t.mu.Unlock()
}

// computeSummary calculates aggregate statistics from samples.
func (t *Tracker) computeSummary() Summary {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.samples) == 0 {
		return Summary{}
	}

	var summary Summary
	var totalCPU float64
	var totalMemory uint64

	for _, sample := range t.samples {
		// Track peaks
		if sample.CPUPercent > summary.PeakCPUPercent {
			summary.PeakCPUPercent = sample.CPUPercent
		}
		if sample.MemoryBytes > summary.PeakMemoryBytes {
			summary.PeakMemoryBytes = sample.MemoryBytes
		}

		// Accumulate for averages
		totalCPU += sample.CPUPercent
		totalMemory += sample.MemoryBytes
	}

	// Calculate averages
	count := len(t.samples)
	summary.AverageCPUPercent = totalCPU / float64(count)
	summary.AverageMemoryBytes = totalMemory / uint64(count)

	return summary
}

// OutputPath returns the path where metrics will be written.
func (t *Tracker) OutputPath() string {
	return t.outputPath
}

// writeMetrics writes the collected metrics to a JSON file.
func (t *Tracker) writeMetrics() error {
	t.mu.RLock()
	samples := make([]Sample, len(t.samples))
	copy(samples, t.samples)
	t.mu.RUnlock()

	metrics := Metrics{
		StartTime: t.startTime,
		EndTime:   time.Now(),
		Samples:   samples,
		Summary:   t.computeSummary(),
	}

	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(t.outputPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}
