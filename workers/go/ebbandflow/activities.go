package ebbandflow

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen/ebbandflow"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/activity"
)

type Activities struct{}

type ActivityExecutionResult struct {
	ScheduledTime   time.Time `json:"scheduledTime"`
	ActualStartTime time.Time `json:"actualStartTime"`
}

func (a Activities) MeasureLatencyActivity(
	ctx context.Context,
	activityAction *kitchensink.ExecuteActivityAction,
) (ActivityExecutionResult, error) {
	if delay := activityAction.GetDelay(); delay != nil {
		time.Sleep(delay.AsDuration())
	}

	activityInfo := activity.GetInfo(ctx)
	return ActivityExecutionResult{
		ScheduledTime:   activityInfo.ScheduledTime,
		ActualStartTime: activityInfo.StartedTime,
	}, nil
}

// ProcessFairnessReport processes fairness report data and emits logs and metrics
func (a Activities) ProcessFairnessReport(ctx context.Context, report ebbandflow.FairnessReport) error {
	logger := activity.GetLogger(ctx)
	metricsHandler := activity.GetMetricsHandler(ctx)

	// Log result.
	commonFields := []any{
		"keyCount", report.KeyCount,
		"jainsFairnessIndex", fmt.Sprintf("%.3f", report.JainsFairnessIndex),
		"fairnessOutlierCount", report.FairnessOutlierCount,
	}

	if len(report.Violations) == 0 {
		logger.Info("Fairness status: passed", commonFields...)
	} else {
		// Join violations for logging
		var violations []string
		for _, desc := range report.Violations {
			violations = append(violations, desc)
		}
		violationSummary := fmt.Sprintf("%d violations: [%s]", len(violations), strings.Join(violations, "; "))

		errorFields := append(commonFields, "violationSummary", violationSummary)
		for i, outlier := range report.TopOutliers {
			outlierSummary := fmt.Sprintf("key=%s p95=%.2fms weight=%.1f weightAdjustedP95=%.2fms severity=%.2f",
				outlier.FairnessKey,
				outlier.P95,
				outlier.Weight,
				outlier.WeightAdjustedP95,
				outlier.OutlierSeverity)
			errorFields = append(errorFields, fmt.Sprintf("topOutlier%d", i+1), outlierSummary)
		}
		logger.Error("Fairness status: violated", errorFields...)
	}

	// Emit metrics.
	metricsHandler.Gauge("ebbandflow_fairness_jains_index").Update(report.JainsFairnessIndex)
	metricsHandler.Gauge("ebbandflow_fairness_outlier_count").Update(float64(report.FairnessOutlierCount))
	metricsHandler.Gauge("ebbandflow_fairness_key_count").Update(float64(report.KeyCount))

	// Emit one violation metric per validation index with labels
	for validationIndex := range report.Violations {
		metricsHandler.WithTags(map[string]string{"validation_index": validationIndex}).
			Counter("ebbandflow_fairness_violation").Inc(1)
	}

	return nil
}
