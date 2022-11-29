package scenarios

import (
	"context"
	"fmt"
	"time"
)

type Scenario struct {
	// A unique name within the registered set of scenarios
	Name string
	// Number of instances of the Execute method to run concurrently
	Concurrency uint32
	// Number of iterations to run of this scenario (mutually exclusive with Duration)
	Iterations uint32
	// Duration limit of this scenario (mutually exclusive with Iterations)
	Duration time.Duration
	// Function to execute a single iteration of this scenario
	Execute func(ctx context.Context, run *Run) error
}

func TaskQueueForRunID(scenario *Scenario, runID string) string {
	return fmt.Sprintf("%s:%s", scenario.Name, runID)
}
