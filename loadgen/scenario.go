package loadgen

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"

	"github.com/temporalio/omes/loadgen/kitchensink"
)

type Scenario struct {
	Description string
	Executor    Executor
}

// Executor for a scenario.
type Executor interface {
	// Run the scenario
	Run(context.Context, ScenarioInfo) error
}

// Optional interface that can be implemented by an [Executor] to allow it to be resumable.
type Resumable interface {
	// LoadState loads a snapshot into the executor's internal state.
	//
	// Implementations should pass a reference to a state variable the loader function and assign to their internal state.
	// Callers should call this function before invoking the executor's Run method.
	LoadState(loader func(any) error) error
	// Snapshot returns a snapshot of the executor's internal state. The returned value must be serializable.
	//
	// The serialization format should be supported by the caller of this function.
	// Callers may call this function periodically to get a snapshot of the executor's state.
	Snapshot() any
}

// ExecutorFunc is an [Executor] implementation for a function
type ExecutorFunc func(context.Context, ScenarioInfo) error

// Run implements [Executor.Run].
func (e ExecutorFunc) Run(ctx context.Context, info ScenarioInfo) error { return e(ctx, info) }

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
	// Configuration info passed by user if any.
	Configuration RunConfiguration
	// ScenarioOptions are info passed from the command line. Do not mutate these.
	ScenarioOptions map[string]string
	// The namespace that was used when connecting the client.
	Namespace string
	// Path to the root of the omes dir
	RootPath string
	// StartFromIteration is the iteration to start from when resuming a scenario.
	// This is used to skip iterations that have already been run.
	StartFromIteration int
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

func (s *ScenarioInfo) ScenarioOptionBool(name string, defaultValue bool) bool {
	v := s.ScenarioOptions[name]
	return v == "true"
}

func (s *ScenarioInfo) ScenarioOptionDuration(name string, defaultValue time.Duration) time.Duration {
	v := s.ScenarioOptions[name]
	if v == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		panic(err)
	}
	return d
}

const DefaultIterations = 10
const DefaultMaxConcurrent = 10

type RunConfiguration struct {
	// Number of iterations to run of this scenario (mutually exclusive with Duration).
	Iterations int
	// Duration limit of this scenario (mutually exclusive with Iterations). If neither iterations
	// nor duration is set, default is DefaultIterations. When the Duration is elapsed, no new
	// iterations will be started, but we will wait for any currently running iterations to
	// complete.
	Duration time.Duration
	// Maximum number of instances of the Execute method to run concurrently.
	// Default is DefaultMaxConcurrent.
	MaxConcurrent int
	// MaxIterationsPerSecond is the maximum number of iterations to run per second.
	// Default is zero, meaning unlimited.
	MaxIterationsPerSecond float64
	// Timeout is the maximum amount of time we'll wait for the scenario to finish running.
	// If the timeout is hit any pending executions will be cancelled and the scenario will exit
	// with an error. The default is unlimited.
	Timeout time.Duration
	// Do not register the default search attributes used by scenarios. If the SAs are not registered
	// by the run, they must be registered by some other method. This is needed because cloud cells
	// cannot use the SDK to register SAs, instead the SAs must be registered through the control plane.
	// Default is false.
	DoNotRegisterSearchAttributes bool
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
	*ScenarioInfo
	// Each run should have a unique iteration.
	Iteration int
	Logger    *zap.SugaredLogger
}

// NewRun creates a new run.
func (s *ScenarioInfo) NewRun(iteration int) *Run {
	return &Run{
		ScenarioInfo: s,
		Iteration:    iteration,
		Logger:       s.Logger.With("iteration", iteration),
	}
}

func (s *ScenarioInfo) RegisterDefaultSearchAttributes(ctx context.Context) error {
	if s.Client == nil {
		// No client in some unit tests. Ideally this would be mocked but no mock operator service
		// client is readily available.
		return nil
	}
	// Ensure custom search attributes are registered that many scenarios rely on
	_, err := s.Client.OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
		SearchAttributes: map[string]enums.IndexedValueType{
			"KS_Keyword": enums.INDEXED_VALUE_TYPE_KEYWORD,
			"KS_Int":     enums.INDEXED_VALUE_TYPE_INT,
		},
		Namespace: s.Namespace,
	})
	// Throw an error if the attributes could not be registered, but ignore already exists errs
	alreadyExistsStrings := []string{
		"already exists",
		"attributes mapping unavailble",
	}
	if err != nil {
		isAlreadyExistsErr := false
		for _, s := range alreadyExistsStrings {
			if strings.Contains(err.Error(), s) {
				isAlreadyExistsErr = true
				break
			}
		}
		if !isAlreadyExistsErr {
			return fmt.Errorf("failed to register search attributes: %w", err)
		}
	}
	return nil
}

// TaskQueueForRun returns a default task queue name for the given scenario name and run ID.
func TaskQueueForRun(scenarioName, runID string) string {
	return fmt.Sprintf("%s:%s", scenarioName, runID)
}

func (r *Run) TaskQueue() string {
	return TaskQueueForRun(r.ScenarioName, r.RunID)
}

// DefaultStartWorkflowOptions gets default start workflow info.
func (r *Run) DefaultStartWorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                TaskQueueForRun(r.ScenarioName, r.RunID),
		ID:                                       fmt.Sprintf("w-%s-%d", r.RunID, r.Iteration),
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
}

// DefaultKitchenSinkWorkflowOptions gets the default kitchen sink workflow info.
func (r *Run) DefaultKitchenSinkWorkflowOptions() KitchenSinkWorkflowOptions {
	return KitchenSinkWorkflowOptions{StartOptions: r.DefaultStartWorkflowOptions()}
}

type KitchenSinkWorkflowOptions struct {
	Params       *kitchensink.TestInput
	StartOptions client.StartWorkflowOptions
}

// ExecuteKitchenSinkWorkflow starts the generic "kitchen sink" workflow and waits for its
// completion ignoring its result. Concurrently it will perform any client actions specified in
// kitchensink.TestInput.ClientSequence
func (r *Run) ExecuteKitchenSinkWorkflow(ctx context.Context, options *KitchenSinkWorkflowOptions) error {
	r.Logger.Debugf("Executing kitchen sink workflow with options: %v", options)
	cancelCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	executor := &kitchensink.ClientActionsExecutor{
		Client:          r.Client,
		WorkflowOptions: options.StartOptions,
		WorkflowType:    "kitchenSink",
		WorkflowInput:   options.Params.GetWorkflowInput(),
	}
	startErr := executor.Start(ctx, options.Params.WithStartAction)
	if startErr != nil {
		return fmt.Errorf("failed to start kitchen sink workflow: %w", startErr)
	}

	var clientActionsErr error
	clientSeq := options.Params.ClientSequence
	if clientSeq != nil && len(clientSeq.ActionSets) > 0 {
		go func() {
			clientActionsErr = executor.ExecuteClientSequence(cancelCtx, clientSeq)
			if clientActionsErr != nil {
				r.Logger.Error("Client actions failed: ", clientActionsErr)
				cancel()
				// TODO: Remove or change to "always terminate when exiting early" flag
				err := r.Client.TerminateWorkflow(
					ctx, options.StartOptions.ID, "", "client actions failed", nil)
				if err != nil {
					return
				}
			}
		}()
	}

	executeErr := executor.Handle.Get(cancelCtx, nil)
	if executeErr != nil {
		return fmt.Errorf("failed to execute kitchen sink workflow: %w", executeErr)
	}
	if clientActionsErr != nil {
		return fmt.Errorf("kitchen sink client actions failed: %w", clientActionsErr)
	}
	return nil
}

// ExecuteAnyWorkflow wraps calls to the client executing workflows to include some logging,
// returning an error if the execution fails.
func (r *Run) ExecuteAnyWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow interface{}, valuePtr interface{}, args ...interface{}) error {
	r.Logger.Debugf("Executing workflow %s with info: %v", workflow, options)
	execution, err := r.Client.ExecuteWorkflow(ctx, options, workflow, args...)
	if err != nil {
		return err
	}
	if err := execution.Get(ctx, valuePtr); err != nil {
		return fmt.Errorf("workflow execution failed (ID: %s, run ID: %s): %w", execution.GetID(), execution.GetRunID(), err)
	}
	return nil
}
