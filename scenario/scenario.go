package scenario

import (
	"context"

	"github.com/temporalio/omes/components/metrics"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type RunOptions struct {
	// Name of the scenario (inferred from the file name)
	ScenarioName string
	// Run ID of the current scenario run, used to generate a unique task queue and workflow ID prefix.
	RunID string
	// Metrics component for registering new metrics.
	Metrics *metrics.Metrics
	// A zap logger.
	Logger *zap.SugaredLogger
	// A Temporal client.
	Client client.Client
}

// Executor for a scenario.
type Executor interface {
	// Run the scenario
	Run(context.Context, *RunOptions) error
}

type Scenario struct {
	Executor Executor
}
