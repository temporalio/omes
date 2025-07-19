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

	if report.KeyCount < 2 {
		logger.Warn("Not enough fairness keys to compare", "keyCount", report.KeyCount)
		return nil
	}

	// Log result.
	if !report.HasViolation {
		logger.Info("Fairness passed",
			"keyCount", report.KeyCount,
			"P95", fmt.Sprintf("%.2fms", report.P95))
	} else {
		logger.Error("Fairness violation",
			"keyCount", report.KeyCount,
			"P95", fmt.Sprintf("%.2fms", report.P95))

		for i, offender := range report.TopOffenders {
			logger.Info("Top offender",
				"rank", i+1,
				"key", offender.FairnessKey,
				"p95", fmt.Sprintf("%.2fms", offender.P95),
				"samples", offender.SampleCount)
		}
	}

	// Emit mertic.
	metricsHandler.Gauge("ebbandflow_fairness_p95").Update(report.P95)

	return nil
}
