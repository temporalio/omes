package ebbandflow

import (
	"context"
	"fmt"
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
		"coefficientOfVariation", fmt.Sprintf("%.3f", report.CoefficientOfVariation),
		"atkinsonIndex", fmt.Sprintf("%.3f", report.AtkinsonIndex),
		"weightAdjustedFairness", fmt.Sprintf("%.3f", report.WeightAdjustedFairness),
	}

	if report.ViolationSummary == "" {
		logger.Info("Fairness passed", commonFields...)
	} else {
		errorFields := append(commonFields, "violationSummary", report.ViolationSummary)
		for i, offender := range report.TopViolators {
			violatorSummary := fmt.Sprintf("key=%s p95=%.2fms weight=%.1f weightAdjustedP95=%.2fms severity=%.2f",
				offender.FairnessKey,
				offender.P95,
				offender.Weight,
				offender.WeightAdjustedP95,
				offender.ViolationSeverity)
			errorFields = append(errorFields, fmt.Sprintf("topViolator%d", i+1), violatorSummary)
		}
		logger.Error("Fairness violation", errorFields...)
	}

	// Emit metrics.
	metricsHandler.Gauge("ebbandflow_fairness_jains_index").Update(report.JainsFairnessIndex)
	metricsHandler.Gauge("ebbandflow_fairness_coefficient_variation").Update(report.CoefficientOfVariation)
	metricsHandler.Gauge("ebbandflow_fairness_atkinson_index").Update(report.AtkinsonIndex)
	metricsHandler.Gauge("ebbandflow_fairness_weight_adjusted").Update(report.WeightAdjustedFairness)
	metricsHandler.Gauge("ebbandflow_fairness_key_count").Update(float64(report.KeyCount))
	if report.ViolationSummary != "" {
		metricsHandler.Counter("ebbandflow_fairness_violation_total").Inc(1)
	}

	return nil
}
