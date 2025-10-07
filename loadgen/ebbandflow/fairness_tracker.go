package ebbandflow

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"golang.org/x/exp/maps"
)

// 2.0 means P95 can be up to 2x the P50 before violation.
const significantDiffThreshold = 2.0

// Maximum number of latency samples to keep per key to prevent unbounded memory growth.
const maxSamplesPerKey = 1000

type FairnessTracker struct {
	mu        sync.RWMutex
	latencies map[string][]float64 // fairness key -> slice of latencies in milliseconds
	weights   map[string]float32   // fairness key -> single weight
}

type OutlierReport struct {
	FairnessKey       string  `json:"fairnessKey"`
	P95               float64 `json:"p95"`
	Weight            float64 `json:"weight"`
	WeightAdjustedP95 float64 `json:"weightAdjustedP95"`
	OutlierSeverity   float64 `json:"outlierSeverity"`
}

type FairnessReport struct {
	KeyCount             int               `json:"keyCount"`
	FairnessOutlierCount int               `json:"fairnessOutlierCount"`
	JainsFairnessIndex   float64           `json:"jainsFairnessIndex"`
	Violations           map[string]string `json:"violations"` // key is fairness indicator
	TopOutliers          []OutlierReport   `json:"topOutliers"`
}

func NewFairnessTracker() *FairnessTracker {
	return &FairnessTracker{
		latencies: make(map[string][]float64),
		weights:   make(map[string]float32),
	}
}

func (ft *FairnessTracker) Track(
	fairnessKey string,
	fairnessWeight float32,
	scheduleToStartLatency time.Duration,
) {
	if fairnessKey == "" {
		return
	}
	ft.mu.Lock()
	defer ft.mu.Unlock()

	latencySlice := ft.latencies[fairnessKey]

	// Add the new sample
	latencySlice = append(latencySlice, float64(scheduleToStartLatency.Milliseconds()))

	// Enforce bounded memory by keeping only the most recent samples
	if len(latencySlice) > maxSamplesPerKey {
		// Keep the most recent maxSamplesPerKey samples
		copy(latencySlice, latencySlice[len(latencySlice)-maxSamplesPerKey:])
		latencySlice = latencySlice[:maxSamplesPerKey]
	}

	ft.latencies[fairnessKey] = latencySlice
	ft.weights[fairnessKey] = fairnessWeight
}

// GetReport returns a processed fairness report without clearing the data.
func (ft *FairnessTracker) GetReport() (*FairnessReport, error) {
	ft.mu.RLock()
	if len(ft.latencies) < 2 {
		ft.mu.RUnlock()
		return nil, fmt.Errorf("need at least 2 fairness keys, got %d", len(ft.latencies))
	}

	latencyData := make(map[string][]float64, len(ft.latencies))
	weights := make(map[string]float64, len(ft.weights))
	for key, slice := range ft.latencies {
		if len(slice) == 0 {
			continue
		}
		values := make([]float64, len(slice))
		copy(values, slice)
		latencyData[key] = values

		// Get the weight for this key
		if weight, exists := ft.weights[key]; exists {
			weights[key] = float64(weight)
		} else {
			weights[key] = 1.0 // default weight
		}
	}
	ft.mu.RUnlock()

	p95Values := make(map[string]float64, len(latencyData))
	sampleCounts := make(map[string]int, len(latencyData))
	for key, values := range latencyData {
		sort.Float64s(values)
		p95 := calculatePercentile(values, 0.95)
		p95Values[key] = p95
		sampleCounts[key] = len(values)
	}

	if len(p95Values) < 2 {
		return nil, fmt.Errorf("need at least 2 keys with data, got %d", len(p95Values))
	}

	// Extract P95 values for distribution analysis
	p95Slice := maps.Values(p95Values)
	sort.Float64s(p95Slice)

	// Calculate volume adjustments based on task count relative to average.
	// This represents the "heaviness" - how much load each key is generating.
	var totalSamples int
	for _, count := range sampleCounts {
		totalSamples += count
	}
	avgSamplesPerKey := float64(totalSamples) / float64(len(sampleCounts))

	volumeMultipliers := make(map[string]float64, len(sampleCounts))
	for key, count := range sampleCounts {
		// Volume multiplier: how much more/less volume this key has vs average
		// Higher volume keys should naturally have higher latencies
		volumeMultipliers[key] = float64(count) / avgSamplesPerKey
	}

	// Calculate weight-adjusted P95 values accounting for both FairnessWeight and volume
	// Formula: p95 / (fairnessWeight * volumeMultiplier)
	// - fairnessWeight: intentional priority (higher = should get better treatment)
	// - volumeMultiplier: natural volume effect (higher = naturally expect worse latency)
	weightAdjustedP95s := make(map[string]float64, len(p95Values))
	for key, p95 := range p95Values {
		fairnessWeight := weights[key]
		volumeMultiplier := volumeMultipliers[key]

		// Ensure all keys are included with sensible defaults to avoid silently dropping keys
		if fairnessWeight <= 0 {
			fairnessWeight = 1.0
		}
		if volumeMultiplier <= 0 {
			volumeMultiplier = 1.0
		}

		// Divide by fairnessWeight: higher priority should have lower normalized latency
		// Divide by volumeMultiplier: higher volume naturally has higher latency, so normalize for that
		weightAdjustedP95s[key] = p95 / (fairnessWeight * volumeMultiplier)
	}

	// Count how many keys exceed the fairness threshold after weight adjustment
	offenderCount, weightAdjustedViolation := calculateWeightAdjustedDispersionOffenders(weightAdjustedP95s, significantDiffThreshold)

	// Calculate Jain's Fairness Index (0-1 scale, where 1.0 = perfect fairness)
	jainsFairnessIndex := calculateJainsFairnessIndex(weightAdjustedP95s)
	jainsViolation := jainsFairnessIndex < 0.8

	// Identify top 5 outliers
	topOutliers := identifyTopOutliers(p95Values, weights, volumeMultipliers, p95Slice, significantDiffThreshold)

	// Create violations map
	violationMap := make(map[string]string)
	if weightAdjustedViolation {
		desc := fmt.Sprintf("Weight-adjusted dispersion: %d offenders exceed %.2fx threshold", offenderCount, significantDiffThreshold)
		violationMap["weight_adjusted_dispersion"] = desc
	}
	if jainsViolation {
		desc := fmt.Sprintf("Jain's fairness index: %.3f < 0.800 threshold", jainsFairnessIndex)
		violationMap["jains_fairness_index"] = desc
	}

	return &FairnessReport{
		KeyCount:             len(p95Values),
		FairnessOutlierCount: offenderCount,
		JainsFairnessIndex:   jainsFairnessIndex,
		Violations:           violationMap,
		TopOutliers:          topOutliers,
	}, nil
}

// calculatePercentile calculates the specified percentile from a sorted slice of float64 values.
func calculatePercentile(sortedValues []float64, percentile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if percentile < 0 {
		percentile = 0
	}
	if percentile > 1 {
		percentile = 1
	}

	index := percentile * float64(len(sortedValues)-1)
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sortedValues[lower]
	}

	// Linear interpolation between the two nearest values
	weight := index - float64(lower)
	return sortedValues[lower]*(1-weight) + sortedValues[upper]*weight
}

// calculateJainsFairnessIndex calculates Jain's Fairness Index: (Σxi)² / (n × Σxi²)
// Returns a value between 0 and 1, where 1 indicates perfect fairness
func calculateJainsFairnessIndex(values map[string]float64) float64 {
	if len(values) == 0 {
		return 1.0
	}
	if len(values) == 1 {
		return 1.0
	}

	var sum, sumSquares float64
	for _, value := range values {
		sum += value
		sumSquares += value * value
	}

	if sumSquares == 0 {
		return 1.0
	}

	n := float64(len(values))
	return (sum * sum) / (n * sumSquares)
}

// identifyTopOutliers finds the top 5 keys with worst fairness outliers
func identifyTopOutliers(
	p95Values map[string]float64,
	weights map[string]float64,
	volumeMultipliers map[string]float64,
	p95Slice []float64,
	threshold float64,
) []OutlierReport {
	if len(p95Values) < 2 {
		return nil
	}

	// Calculate median P95 for comparison
	medianP95 := calculatePercentile(p95Slice, 0.50)
	if medianP95 == 0 {
		return nil
	}

	var outliers []OutlierReport

	for key, p95 := range p95Values {
		fairnessWeight := weights[key]
		volumeMultiplier := volumeMultipliers[key]

		// Guard against division by zero
		if fairnessWeight <= 0 {
			fairnessWeight = 1.0
		}
		if volumeMultiplier <= 0 {
			volumeMultiplier = 1.0
		}

		weightAdjustedP95 := p95 / (fairnessWeight * volumeMultiplier)

		// Calculate outlier severity based on how much this key exceeds the median
		// Accounts for both fairness weight (intentional priority) and volume multiplier (natural load effect)
		outlierSeverity := (p95 / medianP95) * (1.0 / (fairnessWeight * volumeMultiplier))

		// Only include keys that significantly exceed the median
		if outlierSeverity > threshold {
			outliers = append(outliers, OutlierReport{
				FairnessKey:       key,
				P95:               p95,
				Weight:            fairnessWeight,
				WeightAdjustedP95: weightAdjustedP95,
				OutlierSeverity:   outlierSeverity,
			})
		}
	}

	// Sort by outlier severity (worst first)
	sort.Slice(outliers, func(i, j int) bool {
		return outliers[i].OutlierSeverity > outliers[j].OutlierSeverity
	})

	// Return top 5
	if len(outliers) > 5 {
		outliers = outliers[:5]
	}
	return outliers
}

// calculateWeightAdjustedDispersionOffenders counts how many keys exceed the fairness threshold
func calculateWeightAdjustedDispersionOffenders(weightAdjustedP95s map[string]float64, threshold float64) (int, bool) {
	if len(weightAdjustedP95s) < 2 {
		return 0, false
	}

	i := 0
	weightAdjustedSlice := make([]float64, len(weightAdjustedP95s))
	for _, weightAdjustedP95 := range weightAdjustedP95s {
		weightAdjustedSlice[i] = weightAdjustedP95
		i++
	}

	if len(weightAdjustedSlice) < 2 {
		return 0, false
	}

	sort.Float64s(weightAdjustedSlice)
	medianWeightAdjusted := calculatePercentile(weightAdjustedSlice, 0.50)

	if medianWeightAdjusted <= 0 {
		return 0, false
	}

	// Count how many keys exceed threshold * median
	offenderCount := 0
	for _, weightAdjustedP95 := range weightAdjustedP95s {
		if weightAdjustedP95 > medianWeightAdjusted*threshold {
			offenderCount++
		}
	}

	violation := offenderCount > 0
	return offenderCount, violation
}
