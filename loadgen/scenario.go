package loadgen

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

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
	// PreScenario is called once before the scenario begins execution. Use it for one-time setup,
	// like adding a custom search attribute.
	PreScenario(context.Context, ScenarioInfo) error
	// Run the scenario
	Run(context.Context, ScenarioInfo) error
	// PostScenario is called after the scenario has finished running. Use it for things like
	// verifying that visibility records are as expected, etc.
	PostScenario(context.Context, ScenarioInfo) error
}

// ExecutorFunc is an [Executor] implementation for a function
type ExecutorFunc func(context.Context, RunOptions) error

// Run implements [Executor.Run].
func (e ExecutorFunc) Run(ctx context.Context, opts RunOptions) error { return e(ctx, opts) }

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

// ScenarioInfo contains information about the scenario under execution.
type ScenarioInfo struct {
	// Name of the scenario (inferred from the file name)
	ScenarioName string
	// Run ID of the current scenario run, used to generate a unique task queue
	// and workflow ID prefix. This is a single value for the whole scenario, and
	// not a Workflow RunId.
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
	// The namespace that was used when connecting the client.
	Namespace string
	// Unique salt for this scenario run. Use to provide values that are unique but stable for
	// the duration of the scenario run.
	Salt uuid.UUID
}

func (s *ScenarioInfo) ScenarioOptionInt(name string, defaultValue int) int {
	v := s.ScenarioOptions[name]
	if v == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		panic(err)
	}
	return i
}

// UniqueRunID returns a unique RunID for this scenario run. Since RunID is user-specified and may
// be non-unique, you can use this for a stable (for the run) and guaranteed unique ID.
func (s *ScenarioInfo) UniqueRunID() string {
	return fmt.Sprintf("%s-%s", s.RunID, s.Salt.String())
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

// Run represents an individual scenario run (many may be in a single instance (of possibly many) of a scenario).
type Run struct {
	// Do not mutate this, this is shared across the entire scenario
	*RunOptions
	// Each run should have a unique iteration.
	Iteration int
	Logger    *zap.SugaredLogger
}

// NewRun creates a new run.
func (r *RunOptions) NewRun(iteration int) *Run {
	return &Run{
		RunOptions: r,
		Iteration:  iteration,
		Logger:     r.Logger.With("iteration", iteration),
	}
}

// TaskQueueForRun returns a default task queue name for the given scenario name and run ID.
func TaskQueueForRun(scenarioName, runID string) string {
	return fmt.Sprintf("%s:%s", scenarioName, runID)
}

func (r *Run) TaskQueue() string {
	return TaskQueueForRun(r.ScenarioName, r.ID)
}

// DefaultStartWorkflowOptions gets default start workflow options.
func (r *Run) DefaultStartWorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                TaskQueueForRun(r.ScenarioName, r.RunID),
		ID:                                       fmt.Sprintf("w-%s-%d", r.RunID, r.Iteration),
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}


// DefaultKitchenSinkWorkflowOptions gets the default kitchen sink workflow options.
func (r *Run) DefaultKitchenSinkWorkflowOptions() KitchenSinkWorkflowOptions {
	return KitchenSinkWorkflowOptions{StartOptions: r.DefaultStartWorkflowOptions()}
}

type KitchenSinkWorkflowOptions struct {
	Params       kitchensink.WorkflowParams
	StartOptions client.StartWorkflowOptions
}

// ExecuteKitchenSinkWorkflow starts the generic "kitchen sink" workflow and waits for its
// completion ignoring its result.
func (r *Run) ExecuteKitchenSinkWorkflow(ctx context.Context, options *KitchenSinkWorkflowOptions) error {
	return r.ExecuteAnyWorkflow(ctx, options.StartOptions, "kitchenSink", nil, options.Params)
}

// ExecuteAnyWorkflow wraps calls to the client executing workflows to include some logging,
// returning an error if the execution fails.
func (r *Run) ExecuteAnyWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, valuePtr interface{}, args ...interface{}) error {
	r.Logger.Debugf("Executing workflow %s with options: %v", workflow, options)
	execution, err := r.Client.ExecuteWorkflow(ctx, options, workflow, args...)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, valuePtr); err != nil {
		return fmt.Errorf("workflow execution failed (ID: %s, run ID: %s): %w", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
