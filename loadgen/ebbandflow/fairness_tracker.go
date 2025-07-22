package ebbandflow

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// 2.0 means P95 can be up to 2x the P50 before violation.
const significantDiffThreshold = 2.0

type FairnessTracker struct {
	mu        sync.RWMutex
	latencies map[string][]time.Duration // fairness key -> list of latencies
	weights   map[string]float32         // fairness key -> single weight
}

type ViolatorReport struct {
	FairnessKey       string  `json:"fairnessKey"`
	P95               float64 `json:"p95"`
	Weight            float64 `json:"weight"`
	WeightAdjustedP95 float64 `json:"weightAdjustedP95"`
	ViolationSeverity float64 `json:"violationSeverity"`
}

type FairnessReport struct {
	KeyCount               int               `json:"keyCount"`
	WeightAdjustedFairness float64           `json:"weightAdjustedFairness"`
	JainsFairnessIndex     float64           `json:"jainsFairnessIndex"`
	CoefficientOfVariation float64           `json:"coefficientOfVariation"`
	AtkinsonIndex          float64           `json:"atkinsonIndex"`
	Violations             map[string]string `json:"violations"` // key is fairness indicator
	TopViolators           []ViolatorReport  `json:"topViolators"`
}

func NewFairnessTracker() *FairnessTracker {
	return &FairnessTracker{
		latencies: make(map[string][]time.Duration),
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
	ft.latencies[fairnessKey] = append(ft.latencies[fairnessKey], scheduleToStartLatency)
	ft.weights[fairnessKey] = fairnessWeight
}

// Clear clears all data.
func (ft *FairnessTracker) Clear() {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.latencies = make(map[string][]time.Duration)
	ft.weights = make(map[string]float32) // assume weights never change
}

// GetReport returns a processed fairness report without clearing the data.
func (ft *FairnessTracker) GetReport() (*FairnessReport, error) {
	ft.mu.Lock()
	defer ft.mu.Unlock()

	if len(ft.latencies) < 2 {
		return nil, fmt.Errorf("need at least 2 fairness keys, got %d", len(ft.latencies))
	}

	p95Values := make(map[string]float64)
	sampleCounts := make(map[string]int)
	weights := make(map[string]float64)

	for key, latencies := range ft.latencies {
		if len(latencies) == 0 {
			continue
		}
		// Convert durations to float64 milliseconds
		values := make([]float64, len(latencies))
		for i, latency := range latencies {
			values[i] = float64(latency.Milliseconds())
		}
		sort.Float64s(values)
		p95 := calculatePercentile(values, 0.95)
		p95Values[key] = p95
		sampleCounts[key] = len(latencies)

		// Get the weight for this key
		if weight, exists := ft.weights[key]; exists {
			weights[key] = float64(weight)
		} else {
			weights[key] = 1.0 // default weight
		}
	}

	if len(p95Values) < 2 {
		return nil, fmt.Errorf("need at least 2 keys with data, got %d", len(p95Values))
	}

	// Extract P95 values for distribution analysis
	p95Slice := make([]float64, 0, len(p95Values))
	for _, p95 := range p95Values {
		p95Slice = append(p95Slice, p95)
	}
	sort.Float64s(p95Slice)

	// Calculate weight-adjusted P95 values for all metrics.
	weightAdjustedP95s := make(map[string]float64)
	for key, p95 := range p95Values {
		if weight := weights[key]; weight > 0 {
			weightAdjustedP95s[key] = p95 / weight
		}
	}

	// Approach 1: Weight-Adjusted Proportional Fairness
	// Normalizes each key's P95 by weight (P95/weight), then checks if the distribution
	// of normalized values is flat. Higher weight keys should get lower latencies, so
	// after normalization all keys should have similar values. We compare P95-of-normalized
	// vs P50-of-normalized: if P95 >> P50, some keys are getting much worse treatment
	// than the median even after accounting for their weight entitlement.
	weightAdjustedFairness, weightAdjustedViolation := calculateWeightAdjustedFairness(weightAdjustedP95s, significantDiffThreshold)

	// Approach 2: Jain's Fairness Index (Weight-Adjusted)
	// This is a standard networking fairness metric that measures how evenly distributed
	// the weight-adjusted P95 latencies are across all fairness keys. It returns a value between 0 and 1:
	// - 1.0 = perfect fairness (all keys have identical weight-adjusted latencies)
	// - 0.8-1.0 = good fairness (small variations acceptable)
	// - < 0.8 = significant unfairness (some keys much worse than others)
	// Now accounts for task weights by using weight-adjusted P95 values.
	jainsFairnessIndex := calculateJainsFairnessIndex(weightAdjustedP95s)
	jainsViolation := jainsFairnessIndex < 0.8

	// Approach 3: Coefficient of Variation (Weight-Adjusted)
	// Measures relative spread (stddev/mean) of weight-adjusted P95 latencies. Higher CV indicates
	// more variability in latencies across keys, suggesting unfair treatment.
	// CV > 0.5 typically indicates significant unfairness.
	coefficientOfVariation := calculateCoefficientOfVariation(weightAdjustedP95s)
	cvViolation := coefficientOfVariation > 0.5

	// Approach 4: Atkinson Index (Weight-Adjusted)
	// Economics inequality measure focusing on worst-off keys. Uses parameter ε=1
	// to give equal weight to proportional gaps. Higher values indicate more inequality.
	// Values > 0.3 suggest significant unfairness with some keys being severely disadvantaged.
	// Now uses weight-adjusted P95 values to account for task priorities.
	atkinsonIndex := calculateAtkinsonIndex(weightAdjustedP95s, 1.0)
	atkinsonViolation := atkinsonIndex > 0.3

	// Approach 5: Latency Envelope Analysis
	// Checks if any key has P99/P95 ratio > 3.0, indicating "fat tail" distributions
	// that suggest intermittent starvation or inconsistent scheduler behavior.
	latencyEnvelopeViolation, latencyEnvelopeDesc := ft.checkLatencyEnvelopeWithDesc(p95Values, 3.0)

	// Identify top 5 violators
	topViolators := ft.identifyTopViolators(p95Values, weights, p95Slice, significantDiffThreshold)

	// Create violations map
	violationMap := make(map[string]string)
	if weightAdjustedViolation {
		desc := fmt.Sprintf("Weight-adjusted fairness: %.2f > %.2f threshold", weightAdjustedFairness, significantDiffThreshold)
		violationMap["weight_adjusted_fairness"] = desc
	}
	if jainsViolation {
		desc := fmt.Sprintf("Jain's fairness index: %.3f < 0.800 threshold", jainsFairnessIndex)
		violationMap["jains_fairness_index"] = desc
	}
	if cvViolation {
		desc := fmt.Sprintf("Coefficient of variation: %.3f > 0.500 threshold", coefficientOfVariation)
		violationMap["coefficient_of_variation"] = desc
	}
	if atkinsonViolation {
		desc := fmt.Sprintf("Atkinson index: %.3f > 0.300 threshold", atkinsonIndex)
		violationMap["atkinson_index"] = desc
	}
	if latencyEnvelopeViolation {
		violationMap["latency_envelope"] = latencyEnvelopeDesc
	}

	return &FairnessReport{
		KeyCount:               len(p95Values),
		WeightAdjustedFairness: weightAdjustedFairness,
		JainsFairnessIndex:     jainsFairnessIndex,
		CoefficientOfVariation: coefficientOfVariation,
		AtkinsonIndex:          atkinsonIndex,
		Violations:             violationMap,
		TopViolators:           topViolators,
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

// identifyTopViolators finds the top 5 keys with worst fairness violations
// Must be called with ft.mu held
func (ft *FairnessTracker) identifyTopViolators(
	p95Values map[string]float64,
	weights map[string]float64,
	p95Slice []float64,
	threshold float64,
) []ViolatorReport {
	if len(p95Values) < 2 {
		return nil
	}

	// Calculate median P95 for comparison
	medianP95 := calculatePercentile(p95Slice, 0.50)
	if medianP95 == 0 {
		return nil
	}

	var violators []ViolatorReport

	for key, p95 := range p95Values {
		weight := weights[key]
		weightAdjustedP95 := p95 / weight

		// Calculate violation severity based on how much this key exceeds the median
		// For weighted keys, we expect LOWER latencies, so adjust the severity calculation
		// Higher weight should result in lower relative severity for the same absolute latency
		violationSeverity := (p95 / medianP95) * (1.0 / weight)

		// Only include keys that significantly exceed the median
		if violationSeverity > threshold {
			violators = append(violators, ViolatorReport{
				FairnessKey:       key,
				P95:               p95,
				Weight:            weight,
				WeightAdjustedP95: weightAdjustedP95,
				ViolationSeverity: violationSeverity,
			})
		}
	}

	// Sort by violation severity (worst first)
	sort.Slice(violators, func(i, j int) bool {
		return violators[i].ViolationSeverity > violators[j].ViolationSeverity
	})

	// Return top 5
	if len(violators) > 5 {
		violators = violators[:5]
	}

	return violators
}

// calculateWeightAdjustedFairness computes weight-adjusted fairness metric and violation status
func calculateWeightAdjustedFairness(weightAdjustedP95s map[string]float64, threshold float64) (float64, bool) {
	if len(weightAdjustedP95s) < 2 {
		return 1.0, false
	}

	weightAdjustedSlice := make([]float64, 0, len(weightAdjustedP95s))
	for _, weightAdjustedP95 := range weightAdjustedP95s {
		weightAdjustedSlice = append(weightAdjustedSlice, weightAdjustedP95)
	}

	if len(weightAdjustedSlice) < 2 {
		return 1.0, false
	}

	sort.Float64s(weightAdjustedSlice)
	p50OfWeightAdjusted := calculatePercentile(weightAdjustedSlice, 0.50)
	p95OfWeightAdjusted := calculatePercentile(weightAdjustedSlice, 0.95)

	var fairnessRatio float64
	if p50OfWeightAdjusted > 0 {
		fairnessRatio = p95OfWeightAdjusted / p50OfWeightAdjusted
	} else {
		fairnessRatio = 1.0
	}

	violation := fairnessRatio > threshold
	return fairnessRatio, violation
}

// calculateCoefficientOfVariation calculates CV = stddev/mean of P95 values
func calculateCoefficientOfVariation(p95Values map[string]float64) float64 {
	if len(p95Values) < 2 {
		return 0
	}

	// Calculate mean
	var sum float64
	for _, p95 := range p95Values {
		sum += p95
	}
	mean := sum / float64(len(p95Values))

	if mean == 0 {
		return 0
	}

	// Calculate variance
	var sumSquaredDiffs float64
	for _, p95 := range p95Values {
		diff := p95 - mean
		sumSquaredDiffs += diff * diff
	}
	variance := sumSquaredDiffs / float64(len(p95Values))
	stddev := math.Sqrt(variance)

	return stddev / mean
}

// calculateAtkinsonIndex calculates Atkinson inequality index with parameter epsilon
func calculateAtkinsonIndex(p95Values map[string]float64, epsilon float64) float64 {
	if len(p95Values) < 2 {
		return 0
	}

	var sum float64
	var logSum float64
	n := float64(len(p95Values))

	for _, p95 := range p95Values {
		if p95 <= 0 {
			return 0 // Cannot compute with zero or negative values
		}
		sum += p95
		logSum += math.Log(p95)
	}

	if sum == 0 {
		return 0
	}

	arithmeticMean := sum / n
	geometricMean := math.Exp(logSum / n)

	if arithmeticMean == 0 {
		return 0
	}

	return 1 - (geometricMean / arithmeticMean)
}

// checkLatencyEnvelope checks if any key has P99/P95 ratio exceeding threshold
func (ft *FairnessTracker) checkLatencyEnvelope(p95Values map[string]float64, threshold float64) bool {
	for key := range p95Values {
		latencies, exists := ft.latencies[key]
		if !exists || len(latencies) < 10 { // Need enough samples for P99
			continue
		}

		// Convert durations to float64 milliseconds
		values := make([]float64, len(latencies))
		for i, latency := range latencies {
			values[i] = float64(latency.Milliseconds())
		}
		sort.Float64s(values)

		p95 := calculatePercentile(values, 0.95)
		p99 := calculatePercentile(values, 0.99)

		if p95 > 0 && (p99/p95) > threshold {
			return true // Fat tail detected
		}
	}
	return false
}

// checkLatencyEnvelopeWithDesc checks if any key has P99/P95 ratio exceeding threshold and returns descriptive text
func (ft *FairnessTracker) checkLatencyEnvelopeWithDesc(p95Values map[string]float64, threshold float64) (bool, string) {
	for key := range p95Values {
		latencies, exists := ft.latencies[key]
		if !exists || len(latencies) < 10 { // Need enough samples for P99
			continue
		}

		// Convert durations to float64 milliseconds
		values := make([]float64, len(latencies))
		for i, latency := range latencies {
			values[i] = float64(latency.Milliseconds())
		}
		sort.Float64s(values)

		p95 := calculatePercentile(values, 0.95)
		p99 := calculatePercentile(values, 0.99)

		if p95 > 0 && (p99/p95) > threshold {
			ratio := p99 / p95
			return true, fmt.Sprintf("Latency envelope violation: key '%s' P99/P95 ratio %.2f > %.1f threshold (P95=%.1fms, P99=%.1fms)",
				key, ratio, threshold, p95, p99)
		}
	}
	return false, "No latency envelope violations detected"
}
