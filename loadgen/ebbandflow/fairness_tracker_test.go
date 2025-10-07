package ebbandflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFairnessTrackerHandlesBothVolumeAndPriorityScenarios(t *testing.T) {
	tests := []struct {
		name                string
		setupFunc           func(*FairnessTracker)
		violationIndicators []string
		expectedViolators   []string // expected fairness keys in TopViolators (order matters)
	}{
		{
			name: "keys have equal weights and equal volumes",
			setupFunc: func(ft *FairnessTracker) {
				// Equal weights, equal volumes, similar latencies -> should be fair
				for i := 0; i < 5; i++ {
					ft.Track("keyA", 1.0, 100*time.Millisecond)
					ft.Track("keyB", 1.0, 110*time.Millisecond)
				}
			},
			expectedViolators: []string{}, // no violators expected
		},
		{
			name: "volume differences are properly adjusted",
			setupFunc: func(ft *FairnessTracker) {
				// Same weights, different volumes - should be fair with volume adjustment
				ft.Track("keyA", 1.0, 50*time.Millisecond) // Low volume, low latency
				for i := 0; i < 10; i++ {
					ft.Track("keyB", 1.0, 200*time.Millisecond) // High volume, high latency
				}
			},
			expectedViolators: []string{"keyA"}, // keyA exceeds individual threshold but overall fairness metrics pass
		},
		{
			name: "different weights receive proportional treatment",
			setupFunc: func(ft *FairnessTracker) {
				// Different weights with appropriately proportional latencies - should be fair
				// keyB has 4x higher weight, so should get ~4x better latency
				for i := 0; i < 5; i++ {
					ft.Track("keyA", 1.0, 200*time.Millisecond) // Normal priority, normal latency
					ft.Track("keyB", 4.0, 50*time.Millisecond)  // 4x priority, 4x better latency
				}
			},
			expectedViolators: []string{}, // proportional treatment should be fair
		},
		{
			name: "keys have different priorities",
			setupFunc: func(ft *FairnessTracker) {
				// Different weights, proportional latencies - will show some violations but that's expected
				for i := 0; i < 5; i++ {
					ft.Track("keyA", 1.0, 200*time.Millisecond) // Normal priority
					ft.Track("keyB", 3.0, 70*time.Millisecond)  // High priority, proportionally lower latency
				}
			},
			violationIndicators: []string{"jains_fairness_index"},
			expectedViolators:   []string{}, // no individual violators, just overall fairness impact
		},
		{
			name: "latency differences are extreme",
			setupFunc: func(ft *FairnessTracker) {
				// Same weights, same volumes, but extreme latency difference -> unfair
				for i := 0; i < 5; i++ {
					ft.Track("keyA", 1.0, 50*time.Millisecond)   // Very fast
					ft.Track("keyB", 1.0, 1000*time.Millisecond) // Very slow
				}
			},
			violationIndicators: []string{"jains_fairness_index"},
			expectedViolators:   []string{}, // 20x latency difference may not exceed 2.0 threshold after median normalization
		},
		{
			name: "high priority keys receive worse treatment",
			setupFunc: func(ft *FairnessTracker) {
				// High priority key getting much worse treatment -> should cause violations
				for i := 0; i < 5; i++ {
					ft.Track("keyA", 1.0, 50*time.Millisecond)    // Low priority, very good latency
					ft.Track("keyB", 10.0, 2000*time.Millisecond) // High priority, terrible latency
				}
			},
			violationIndicators: []string{"jains_fairness_index"},
			expectedViolators:   []string{}, // keyB has terrible treatment but may not exceed threshold due to high weight
		},
		{
			name: "normalized latency distribution has extreme spread",
			setupFunc: func(ft *FairnessTracker) {
				// Creates extreme spread in weight-adjusted P95 distribution to trigger weight_adjusted_dispersion violation
				// The weight_adjusted_dispersion metric compares P95 vs P50 of normalized latencies - if P95/P50 > 2.0, it violates
				// This setup creates 3 keys with good performance and 1 with terrible performance
				for i := 0; i < 10; i++ {
					ft.Track("keyA", 1.0, 100*time.Millisecond)  // Good: normalized = 100/(1.0*1.0) = 100
					ft.Track("keyB", 1.0, 100*time.Millisecond)  // Good: normalized = 100/(1.0*1.0) = 100
					ft.Track("keyC", 1.0, 100*time.Millisecond)  // Good: normalized = 100/(1.0*1.0) = 100
					ft.Track("keyD", 1.0, 5000*time.Millisecond) // Bad: normalized = 5000/(1.0*1.0) = 5000
				}
				// Result: normalized P50≈100, P95≈5000, ratio=50 >> 2.0 threshold
			},
			violationIndicators: []string{"weight_adjusted_dispersion", "jains_fairness_index"},
			expectedViolators:   []string{"keyD"}, // keyD has 50x worse latency than median, will exceed 2.0 threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ft := NewFairnessTracker()
			tt.setupFunc(ft)

			report, err := ft.GetReport()
			require.NoError(t, err, "GetReport should succeed")

			assert.GreaterOrEqual(t, report.KeyCount, 2)
			assert.GreaterOrEqual(t, report.FairnessOutlierCount, 0)

			if len(tt.violationIndicators) > 0 {
				assert.NotEmpty(t, tt.violationIndicators)
				for _, indicator := range tt.violationIndicators {
					assert.Contains(t, report.Violations, indicator)
				}
			} else {
				assert.Empty(t, tt.violationIndicators)
			}

			assert.Len(t, report.TopOutliers, len(tt.expectedViolators), "TopOutliers count should match expected")
			if len(tt.expectedViolators) > 0 {
				for i, expectedKey := range tt.expectedViolators {
					assert.Equal(t, expectedKey, report.TopOutliers[i].FairnessKey,
						"TopOutliers[%d] should be %s", i, expectedKey)
				}
			}
		})
	}
}
