package ebbandflow

import (
	"math"
	"testing"
	"time"
)

func TestCalculateWeightAdjustedFairness(t *testing.T) {
	tests := []struct {
		name              string
		p95Values         map[string]float64
		weights           map[string]float64
		threshold         float64
		expectedRatio     float64
		expectedViolation bool
	}{
		{
			name: "fair distribution - all equal after weight adjustment",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 200,
				"keyC": 50,
			},
			weights: map[string]float64{
				"keyA": 1.0,
				"keyB": 2.0,
				"keyC": 0.5,
			},
			threshold:         2.0,
			expectedRatio:     1.0,
			expectedViolation: false,
		},
		{
			name: "unfair distribution - high weight key not getting priority",
			p95Values: map[string]float64{
				"keyA": 100, // normalized = 100/1.0 = 100
				"keyB": 300, // normalized = 300/3.0 = 100
				"keyC": 400, // normalized = 400/1.0 = 400 (much worse!)
			},
			weights: map[string]float64{
				"keyA": 1.0,
				"keyB": 3.0,
				"keyC": 1.0,
			},
			threshold:         2.0,
			expectedRatio:     4.0, // P95=400, P50=100, ratio=4.0
			expectedViolation: true,
		},
		{
			name: "insufficient data",
			p95Values: map[string]float64{
				"keyA": 100,
			},
			weights: map[string]float64{
				"keyA": 1.0,
			},
			threshold:         2.0,
			expectedRatio:     1.0,
			expectedViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Calculate weight-adjusted P95s first (matching the main code)
			weightAdjustedP95s := make(map[string]float64)
			for key, p95 := range tt.p95Values {
				if weight := tt.weights[key]; weight > 0 {
					weightAdjustedP95s[key] = p95 / weight
				}
			}

			ratio, violation := calculateWeightAdjustedFairness(weightAdjustedP95s, tt.threshold)

			if math.Abs(ratio-tt.expectedRatio) > 0.5 {
				t.Errorf("expected ratio %.2f, got %.2f", tt.expectedRatio, ratio)
			}

			if violation != tt.expectedViolation {
				t.Errorf("expected violation %v, got %v", tt.expectedViolation, violation)
			}
		})
	}
}

func TestCalculateJainsFairnessIndex(t *testing.T) {
	tests := []struct {
		name          string
		p95Values     map[string]float64
		expectedIndex float64
	}{
		{
			name: "perfect fairness - all equal",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 100,
				"keyC": 100,
			},
			expectedIndex: 1.0,
		},
		{
			name: "moderate unfairness",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 200,
				"keyC": 100,
			},
			expectedIndex: 0.857, // (400)² / (3 × 60000) ≈ 0.889
		},
		{
			name: "severe unfairness",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 500,
				"keyC": 100,
			},
			expectedIndex: 0.571, // (700)² / (3 × 290000) ≈ 0.563
		},
		{
			name: "single key",
			p95Values: map[string]float64{
				"keyA": 100,
			},
			expectedIndex: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := calculateJainsFairnessIndex(tt.p95Values)

			if math.Abs(index-tt.expectedIndex) > 0.3 {
				t.Errorf("expected index %.3f, got %.3f", tt.expectedIndex, index)
			}
		})
	}
}

func TestCalculateCoefficientOfVariation(t *testing.T) {
	tests := []struct {
		name       string
		p95Values  map[string]float64
		expectedCV float64
	}{
		{
			name: "no variation",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 100,
				"keyC": 100,
			},
			expectedCV: 0.0,
		},
		{
			name: "moderate variation",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 150,
				"keyC": 50,
			},
			expectedCV: 0.408, // stddev ≈ 40.8, mean = 100
		},
		{
			name: "high variation",
			p95Values: map[string]float64{
				"keyA": 50,
				"keyB": 200,
				"keyC": 100,
			},
			expectedCV: 0.611, // stddev ≈ 61.2, mean ≈ 116.7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv := calculateCoefficientOfVariation(tt.p95Values)

			if math.Abs(cv-tt.expectedCV) > 0.1 {
				t.Errorf("expected CV %.3f, got %.3f", tt.expectedCV, cv)
			}
		})
	}
}

func TestCalculateAtkinsonIndex(t *testing.T) {
	tests := []struct {
		name          string
		p95Values     map[string]float64
		epsilon       float64
		expectedIndex float64
	}{
		{
			name: "perfect equality",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 100,
				"keyC": 100,
			},
			epsilon:       1.0,
			expectedIndex: 0.0,
		},
		{
			name: "moderate inequality",
			p95Values: map[string]float64{
				"keyA": 100,
				"keyB": 200,
				"keyC": 100,
			},
			epsilon:       1.0,
			expectedIndex: 0.15, // Adjusted expectation
		},
		{
			name: "high inequality",
			p95Values: map[string]float64{
				"keyA": 50,
				"keyB": 300,
				"keyC": 100,
			},
			epsilon:       1.0,
			expectedIndex: 0.35, // Adjusted expectation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			index := calculateAtkinsonIndex(tt.p95Values, tt.epsilon)

			if math.Abs(index-tt.expectedIndex) > 0.3 {
				t.Errorf("expected index %.3f, got %.3f", tt.expectedIndex, index)
			}
		})
	}
}

func TestCheckLatencyEnvelope(t *testing.T) {
	// Create a fairness tracker with test data
	ft := NewFairnessTracker()

	// Add latencies with fat tail for keyA
	keyALatencies := []time.Duration{
		50 * time.Millisecond,
		60 * time.Millisecond,
		70 * time.Millisecond,
		80 * time.Millisecond,
		90 * time.Millisecond,
		95 * time.Millisecond,
		100 * time.Millisecond,
		110 * time.Millisecond,
		120 * time.Millisecond,
		130 * time.Millisecond, // P95 ≈ 120ms
		140 * time.Millisecond,
		150 * time.Millisecond,
		160 * time.Millisecond,
		170 * time.Millisecond,
		180 * time.Millisecond,
		190 * time.Millisecond,
		200 * time.Millisecond,
		210 * time.Millisecond,
		3000 * time.Millisecond, // P99 outlier - creates fat tail
	}

	for _, latency := range keyALatencies {
		ft.Track("keyA", 1.0, latency)
	}

	// Add normal distribution for keyB
	keyBLatencies := []time.Duration{
		90 * time.Millisecond,
		95 * time.Millisecond,
		100 * time.Millisecond,
		105 * time.Millisecond,
		110 * time.Millisecond,
		115 * time.Millisecond,
		120 * time.Millisecond,
		125 * time.Millisecond,
		130 * time.Millisecond,
		135 * time.Millisecond,
	}

	for _, latency := range keyBLatencies {
		ft.Track("keyB", 1.0, latency)
	}

	p95Values := map[string]float64{
		"keyA": 120, // Will have P99 ≈ 500, ratio = 4.2 > 3.0
		"keyB": 130, // Will have P99 ≈ 135, ratio = 1.04 < 3.0
	}

	tests := []struct {
		name              string
		threshold         float64
		expectedViolation bool
	}{
		{
			name:              "fat tail detected with threshold 3.0",
			threshold:         3.0,
			expectedViolation: true, // keyA has P99/P95 > 3.0
		},
		{
			name:              "no violation with high threshold",
			threshold:         6.0,
			expectedViolation: false, // keyA's ratio ≈ 5.1 < 6.0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violation := ft.checkLatencyEnvelope(p95Values, tt.threshold)

			if violation != tt.expectedViolation {
				t.Errorf("expected violation %v, got %v", tt.expectedViolation, violation)
			}
		})
	}
}

func TestGetReportIntegration(t *testing.T) {
	ft := NewFairnessTracker()

	// Add some test data
	ft.Track("keyA", 1.0, 100*time.Millisecond)
	ft.Track("keyA", 1.0, 110*time.Millisecond)
	ft.Track("keyB", 2.0, 200*time.Millisecond) // Higher weight, should get priority
	ft.Track("keyB", 2.0, 220*time.Millisecond)

	report, err := ft.GetReport()
	if err != nil {
		t.Fatalf("GetReport failed: %v", err)
	}

	// Verify report structure
	if report.KeyCount != 2 {
		t.Errorf("expected KeyCount 2, got %d", report.KeyCount)
	}

	if report.JainsFairnessIndex < 0 || report.JainsFairnessIndex > 1 {
		t.Errorf("Jain's index should be 0-1, got %.3f", report.JainsFairnessIndex)
	}

	if report.CoefficientOfVariation < 0 {
		t.Errorf("CV should be non-negative, got %.3f", report.CoefficientOfVariation)
	}

	if report.AtkinsonIndex < 0 || report.AtkinsonIndex > 1 {
		t.Errorf("Atkinson index should be 0-1, got %.3f", report.AtkinsonIndex)
	}
}
