package loadgen

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type Scenario struct {
	Executor Executor
}

// Executor for a scenario.
type Executor interface {
	// Run the scenario
	Run(context.Context, *RunOptions) error
}

var registeredScenarios = make(map[string]*Scenario)

// MustRegisterScenario registers a scenario in the global static registry.
// Panics if registration fails.
// The file name of the caller is be used as the scenario name.
func MustRegisterScenario(scenario *Scenario) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("Could not infer caller when registering a nameless scenario")
	}
	scenarioName := strings.Replace(filepath.Base(file), ".go", "", 1)
	_, found := registeredScenarios[scenarioName]
	if found {
		panic(fmt.Errorf("duplicate scenario with name: %s", scenarioName))
	}
	registeredScenarios[scenarioName] = scenario
}

// GetScenario gets a scenario by name from the global static registry.
func GetScenario(name string) *Scenario {
	return registeredScenarios[name]
}

type RunOptions struct {
	// Name of the scenario (inferred from the file name)
	ScenarioName string
	// Run ID of the current scenario run, used to generate a unique task queue and workflow ID prefix.
	RunID string
	// Metrics component for registering new metrics.
	MetricsHandler client.MetricsHandler
	// A zap logger.
	Logger *zap.SugaredLogger
	// A Temporal client.
	Client client.Client
}

// Run represents a scenario run.
type Run struct {
	// Used for Workflow ID and task queue name generation.
	ID string
	// The name of the scenario used for the run.
	ScenarioName string
	// Temporal client to use for executing workflows, etc.
	Client client.Client
	// Each call to the Execute method gets a distinct `IterationInTest`.
	IterationInTest int
	Logger          *zap.SugaredLogger
}

// TaskQueueForRun returns a default task queue name for the given scenario name and run ID.
func TaskQueueForRun(scenarioName, runID string) string {
	return fmt.Sprintf("%s:%s", scenarioName, runID)
}

// TaskQueue returns a default task queue name for this run based on the scenario name and the run ID.
func (r *Run) TaskQueue() string {
	return TaskQueueForRun(r.ScenarioName, r.ID)
}

// WorkflowOptions returns a default set of options that can be used in any scenario.
func (r *Run) WorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                r.TaskQueue(),
		ID:                                       fmt.Sprintf("w-%s-%d", r.ID, r.IterationInTest),
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}

// ExecuteKitchenSinkWorkflow starts the generic "kitchen sink" workflow and waits for its completion ignoring its result.
func (r *Run) ExecuteKitchenSinkWorkflow(ctx context.Context, params *kitchensink.WorkflowParams) error {
	opts := r.WorkflowOptions()
	r.Logger.Debugf("Executing workflow with options: %v", opts)
	execution, err := r.Client.ExecuteWorkflow(ctx, opts, "kitchenSink", params)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, nil); err != nil {
		return fmt.Errorf("error executing workflow (ID: %s, run ID: %s): %w", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
