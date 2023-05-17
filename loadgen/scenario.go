package loadgen

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type Scenario struct {
	Description string
	Executor    Executor
}

// Executor for a scenario.
type Executor interface {
	// Run the scenario
	Run(context.Context, RunOptions) error
}

// HasDefaultConfiguration is an interface executors can implement to show their
// default configuration.
type HasDefaultConfiguration interface {
	GetDefaultConfiguration() RunConfiguration
}

var registeredScenarios = make(map[string]*Scenario)

// MustRegisterScenario registers a scenario in the global static registry.
// Panics if registration fails.
// The file name of the caller is be used as the scenario name.
func MustRegisterScenario(scenario Scenario) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		panic("Could not infer caller when registering a nameless scenario")
	}
	scenarioName := strings.Replace(filepath.Base(file), ".go", "", 1)
	_, found := registeredScenarios[scenarioName]
	if found {
		panic(fmt.Errorf("duplicate scenario with name: %s", scenarioName))
	}
	registeredScenarios[scenarioName] = &scenario
}

// GetScenarios gets a copy of registered scenarios
func GetScenarios() map[string]*Scenario {
	ret := make(map[string]*Scenario, len(registeredScenarios))
	for k, v := range registeredScenarios {
		ret[k] = v
	}
	return ret
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
	// Configuration options passed by user if any.
	Configuration RunConfiguration
	// ScenarioOptions are options passed from the command line. Do not mutate these.
	ScenarioOptions map[string]string
}

func (r *RunOptions) ScenarioOptionInt(name string, defaultValue int) int {
	v := r.ScenarioOptions[name]
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return i
}

const DefaultIterations = 10
const DefaultMaxConcurrent = 10

type RunConfiguration struct {
	// Number of iterations to run of this scenario (mutually exclusive with Duration).
	Iterations int
	// Duration limit of this scenario (mutually exclusive with Iterations). If
	// neither iterations nor duration is set, default is DefaultIterations.
	Duration time.Duration
	// Maximum number of instances of the Execute method to run concurrently.
	// Default is DefaultMaxConcurrent.
	MaxConcurrent int
}

func (r *RunConfiguration) ApplyDefaults() {
	if r.Iterations == 0 && r.Duration == 0 {
		r.Iterations = DefaultIterations
	}
	if r.MaxConcurrent == 0 {
		r.MaxConcurrent = DefaultMaxConcurrent
	}
}

// Run represents a scenario run.
type Run struct {
	// Used for Workflow ID and task queue name generation.
	ID string
	// The name of the scenario used for the run.
	ScenarioName string
	// Temporal client to use for executing workflows, etc.
	Client client.Client
	// Each call to the Execute method gets a distinct `IterationInTest`. This
	// will never be 0.
	IterationInTest int
	Logger          *zap.SugaredLogger
	// Do not mutate this
	RunOptions *RunOptions
}

// TaskQueueForRun returns a default task queue name for the given scenario name and run ID.
func TaskQueueForRun(scenarioName, runID string) string {
	return fmt.Sprintf("%s:%s", scenarioName, runID)
}

func (r *Run) DefaultKitchenSinkWorkflowOptions() KitchenSinkWorkflowOptions {
	return KitchenSinkWorkflowOptions{
		StartOptions: client.StartWorkflowOptions{
			TaskQueue:                                TaskQueueForRun(r.ScenarioName, r.ID),
			ID:                                       fmt.Sprintf("w-%s-%d", r.ID, r.IterationInTest),
			WorkflowExecutionErrorWhenAlreadyStarted: true,
		},
	}
}

type KitchenSinkWorkflowOptions struct {
	Params       kitchensink.WorkflowParams
	StartOptions client.StartWorkflowOptions
}

// ExecuteKitchenSinkWorkflow starts the generic "kitchen sink" workflow and waits for its completion ignoring its result.
func (r *Run) ExecuteKitchenSinkWorkflow(ctx context.Context, options *KitchenSinkWorkflowOptions) error {
	r.Logger.Debugf("Executing workflow with options: %v", options.StartOptions)
	execution, err := r.Client.ExecuteWorkflow(ctx, options.StartOptions, "kitchenSink", options.Params)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, nil); err != nil {
		return fmt.Errorf("error executing workflow (ID: %s, run ID: %s): %w", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
