package ebbandflow

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type FairnessTracker struct {
	mu        sync.RWMutex
	latencies map[string][]time.Duration // fairness key -> list of latencies
}

type OffenderReport struct {
	FairnessKey string  `json:"fairnessKey"`
	SampleCount int     `json:"sampleCount"`
	P95         float64 `json:"p95"`
}

type FairnessReport struct {
	KeyCount     int              `json:"keyCount"`
	TopOffenders []OffenderReport `json:"topOffenders"`
	P95          float64          `json:"p95"`
	HasViolation bool             `json:"hasViolation"`
}

func NewFairnessTracker() *FairnessTracker {
	return &FairnessTracker{
		latencies: make(map[string][]time.Duration),
	}
}

func (ft *FairnessTracker) Track(fairnessKey string, scheduleToStartLatency time.Duration) {
	if fairnessKey == "" {
		return
	}
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.latencies[fairnessKey] = append(ft.latencies[fairnessKey], scheduleToStartLatency)
}

// Clear clears all data.
func (ft *FairnessTracker) Clear() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.latencies = make(map[string][]time.Duration)
}

// GetReport returns a processed fairness report without clearing the data.
func (ft *FairnessTracker) GetReport(significantDiffThreshold float64) (*FairnessReport, error) {
	// Copy data while holding lock briefly
	ft.mu.RLock()
	if len(ft.latencies) < 2 {
		ft.mu.RUnlock()
		return nil, fmt.Errorf("need at least 2 fairness keys, got %d", len(ft.latencies))
	}

	// Make a deep copy of the latencies data.
	latenciesCopy := make(map[string][]time.Duration)
	for key, latencies := range ft.latencies {
		if len(latencies) > 0 {
			latenciesCopy[key] = make([]time.Duration, len(latencies))
			copy(latenciesCopy[key], latencies)
		}
	}
	ft.mu.RUnlock()
	// Now do all calculations without holding the lock.

	var allReports []OffenderReport
	p95Values := make(map[string]float64)

	for key, latencies := range latenciesCopy {
		if len(latencies) == 0 {
			continue
		}
		p95 := calculateP95(latencies)
		p95Values[key] = p95
		allReports = append(allReports, OffenderReport{
			FairnessKey: key,
			SampleCount: len(latencies),
			P95:         p95,
		})
	}

	if len(p95Values) < 2 {
		return nil, fmt.Errorf("need at least 2 keys with data, got %d", len(p95Values))
	}

	// Sort by P95 descending to get worst offenders.
	sort.Slice(allReports, func(i, j int) bool {
		return allReports[i].P95 > allReports[j].P95
	})
	topOffenders := allReports
	if len(topOffenders) > 5 {
		topOffenders = topOffenders[:5]
	}

	// Find min and max P95.
	var minP95, maxP95 float64
	first := true
	for _, p95 := range p95Values {
		if first {
			minP95, maxP95 = p95, p95
			first = false
		} else {
			if p95 < minP95 {
				minP95 = p95
			}
			if p95 > maxP95 {
				maxP95 = p95
			}
		}
	}

	// Calculate average P95 across all keys.
	var totalP95 float64
	for _, p95 := range p95Values {
		totalP95 += p95
	}
	avgP95 := totalP95 / float64(len(p95Values))

	hasViolation := maxP95 > minP95*significantDiffThreshold

	return &FairnessReport{
		KeyCount:     len(p95Values),
		TopOffenders: topOffenders,
		P95:          avgP95,
		HasViolation: hasViolation,
	}, nil
}

func calculateP95(latencies []time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(math.Ceil(0.95 * float64(len(sorted))))
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	if index < 0 {
		index = 0
	}

	return float64(sorted[index].Milliseconds())
}
