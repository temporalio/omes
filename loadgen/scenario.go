package loadgen

import (
	"context"
	"fmt"
	"maps"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/nexus/v1"
	"go.temporal.io/api/operatorservice/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"

	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

var sanitizeRunID = regexp.MustCompile(`[^a-zA-Z0-9-]`)

const OmesExecutionIDSearchAttribute = "OmesExecutionID"

type Scenario struct {
	Description string
	// Options declares the options a user may set with `--option <name>=<value>`,
	// using the same typed pflag registrars as omes's own CLI flags:
	//
	//	Options: func(o *loadgen.OptionSet) {
	//		o.Int("children-per-workflow", 30, "Child workflows per iteration.")
	//		o.MarkRequired("task-queue-count")
	//	}
	//
	// Declarations are the single source of truth for an option's type and
	// default: omes rejects unknown names and malformed values before the run
	// starts, `list-scenarios` shows what the scenario accepts, and the Option*
	// accessors read from them. A scenario that declares nothing accepts no
	// options.
	Options func(*OptionSet)
	// DefaultConfiguration is the scenario's own default run configuration,
	// shown by `list-scenarios` and used for any field the user does not
	// override. It supersedes the [HasDefaultConfiguration] executor interface.
	DefaultConfiguration *RunConfiguration
	ExecutorFn           func() Executor
}

// GetDefaultConfiguration prefers the scenario's declared configuration and
// falls back to the executor's [HasDefaultConfiguration] implementation,
// reporting whether either was present.
func (s *Scenario) GetDefaultConfiguration() (RunConfiguration, bool) {
	if s.DefaultConfiguration != nil {
		return *s.DefaultConfiguration, true
	}
	if s.ExecutorFn != nil {
		if iface, _ := s.ExecutorFn().(HasDefaultConfiguration); iface != nil {
			return iface.GetDefaultConfiguration(), true
		}
	}
	return RunConfiguration{}, false
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
	// Implementations should pass a reference to a state variable to the loader function and assign to their internal state.
	// Callers should call this function before invoking the executor's Run method.
	LoadState(loader func(any) error) error
	// Snapshot returns a snapshot of the executor's internal state. The returned value must be serializable.
	//
	// The serialization format should be supported by the caller of this function.
	// Callers may call this function periodically to get a snapshot of the executor's state.
	Snapshot() any
}

// Optional interface that can be implemented by an [Executor] to make it configurable.
type Configurable interface {
	// Configure the executor with the given scenario info.
	//
	// Call this method if you want to ensure that all required configuration parameters
	// are present and valid without actually running the executor.
	Configure(ScenarioInfo) error
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
// The file name of the caller is used as the scenario name.
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
	maps.Copy(ret, registeredScenarios)
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
	// ExecutionID is a randomly generated ID that uniquely identifies this particular
	// execution of the scenario. Combined with RunID, it ensures no two executions collide.
	ExecutionID string
	// Metrics component for registering new metrics.
	MetricsHandler client.MetricsHandler
	// A zap logger.
	Logger *zap.SugaredLogger
	// A Temporal client.
	Client client.Client
	// Temporal client options
	ClientOptions clioptions.ClientOptions
	// Configuration info passed by user if any.
	Configuration RunConfiguration
	// Options holds the scenario's options: each declared default, overwritten by
	// whatever the user passed with `--option`. Read them with the Option*
	// accessors. Do not mutate.
	Options *OptionSet
	// The namespace that was used when connecting the client.
	Namespace string
	// Path to the root of the omes dir
	RootPath string
	// ExportOptions contains export-related configuration
	ExportOptions ExportOptions
}

// ExportOptions contains configuration for exporting scenario data.
type ExportOptions struct {
	// Directory to export histories (empty = disabled)
	ExportHistoriesDir string
	// Status filter: "failed", "terminated", "failed,terminated", "all"
	ExportHistoriesFilter string
}

// The Option* accessors return the option's value: what the user passed, or the
// default the scenario declared. Values are validated before the run starts, so
// these cannot fail on user input.
//
// Reading an option the scenario did not declare is a bug in the scenario, not
// something a user can cause. It logs and returns the zero value.

func (s *ScenarioInfo) OptionInt(name string) int {
	v, err := s.options().GetInt(name)
	if err != nil {
		s.undeclaredOption(name, err)
	}
	return v
}

func (s *ScenarioInfo) OptionFloat64(name string) float64 {
	v, err := s.options().GetFloat64(name)
	if err != nil {
		s.undeclaredOption(name, err)
	}
	return v
}

func (s *ScenarioInfo) OptionBool(name string) bool {
	v, err := s.options().GetBool(name)
	if err != nil {
		s.undeclaredOption(name, err)
	} else if s.options().unresolvedFeature(name) {
		s.unresolvedFeatureOption(name)
	}
	return v
}

func (s *ScenarioInfo) OptionDuration(name string) time.Duration {
	v, err := s.options().GetDuration(name)
	if err != nil {
		s.undeclaredOption(name, err)
	}
	return v
}

func (s *ScenarioInfo) OptionString(name string) string {
	v, err := s.options().GetString(name)
	if err != nil {
		s.undeclaredOption(name, err)
	}
	return v
}

// OptionUserSpecified reports whether name was explicitly supplied via
// `--option name=value`, as opposed to reading its declared (or feature-resolved)
// default. Use it when the presence of a value matters, not just its content, as
// with the standalone-nexus/nexus-enabled interaction in throughput_stress.
func (s *ScenarioInfo) OptionUserSpecified(name string) bool {
	return s.options().UserSpecified(name)
}

// options returns the resolved set, or an empty one so a ScenarioInfo built
// without options (in tests, say) reads zero values rather than panicking.
func (s *ScenarioInfo) options() *OptionSet {
	if s.Options == nil {
		s.Options = newOptionSet("")
	}
	return s.Options
}

// logf logs at info level, tolerating a ScenarioInfo built without a logger —
// library callers assemble one by hand and need not set every field.
func (s *ScenarioInfo) logf(format string, args ...any) {
	if s.Logger != nil {
		s.Logger.Infof(format, args...)
	} else {
		clioptions.BackupLogger.Printf(format, args...)
	}
}

// unresolvedFeatureOption reports a feature option read before any namespace
// probe ran, which leaves it false and quietly drops the feature from the load.
// It is an error rather than a warning because a run generating less load than
// intended otherwise looks exactly like one that worked. Callers driving a
// scenario as a library, rather than through the omes CLI, must call
// [ResolveFeatureOptions] once the client is dialed.
func (s *ScenarioInfo) unresolvedFeatureOption(name string) {
	msg := fmt.Sprintf("scenario %q read feature option %q before resolving capabilities for "+
		"namespace %q, so it reads false whether or not the option is supported; "+
		"call loadgen.ResolveFeatureOptions after dialing to fix this",
		s.ScenarioName, name, s.Namespace)
	if s.Logger != nil {
		s.Logger.Error(msg)
	} else {
		clioptions.BackupLogger.Println(msg)
	}
}

func (s *ScenarioInfo) undeclaredOption(name string, err error) {
	msg := fmt.Sprintf("scenario %q read option %q, which it does not declare: %v",
		s.ScenarioName, name, err)
	if s.Logger != nil {
		s.Logger.Error(msg)
	} else {
		clioptions.BackupLogger.Println(msg)
	}
}

const DefaultIterations = 10
const DefaultMaxConcurrentIterations = 10
const DefaultMaxIterationAttempts = 1
const BaseIterationRetryBackoff = 1 * time.Second
const MaxIterationRetryBackoff = 60 * time.Second

type RunConfiguration struct {
	// Number of iterations to run of this scenario (mutually exclusive with Duration).
	Iterations int
	// StartFromIteration is the iteration to start from when resuming a run.
	// This is used to skip iterations that have already been run.
	// Default is zero. If Iterations is set, too, must be less than or equal to Iterations.
	StartFromIteration int
	// MaxIterationAttempts is the maximum number of attempts to run the scenario.
	// Default (and minimum) is 1.
	MaxIterationAttempts int
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
	// IgnoreAlreadyStarted, if set, will not error when a workflow with the same ID already exists.
	// Default is false.
	IgnoreAlreadyStarted bool
	// OnCompletion, if set, is invoked after each successful iteration completes.
	OnCompletion func(context.Context, *Run)
	// HandleExecuteError, if set, is called when Execute returns an error, allowing transformation of errors.
	HandleExecuteError func(context.Context, *Run, error) error
	// ContinueOnIterationFailure, if set, keeps the run going after an iteration
	// terminally fails (i.e. exhausts MaxIterationAttempts) instead of aborting on
	// the first failure. The failed iteration is reported via OnIterationFailure
	// and new iterations keep starting until the duration or iteration limit is
	// reached. This is the mechanism for failure-tolerant runs; the policy for how
	// many failures are acceptable belongs to the caller. Default is false.
	ContinueOnIterationFailure bool
	// OnIterationFailure, if set, is invoked when an iteration terminally fails
	// (after exhausting retries). It is the hook callers use to tally failures
	// when running with ContinueOnIterationFailure.
	OnIterationFailure func(context.Context, *Run, error)
}

func (r *RunConfiguration) ApplyDefaults() {
	if r.Iterations == 0 && r.Duration == 0 {
		r.Iterations = DefaultIterations
	}
	if r.MaxConcurrent == 0 {
		r.MaxConcurrent = DefaultMaxConcurrentIterations
	}
	if r.MaxIterationAttempts == 0 {
		r.MaxIterationAttempts = DefaultMaxIterationAttempts
	}
}

func (r RunConfiguration) Validate() error {
	if r.Duration < 0 {
		return fmt.Errorf("Duration cannot be negative")
	}
	if r.Iterations > 0 {
		if r.Duration > 0 {
			return fmt.Errorf("iterations and duration are mutually exclusive")
		}
		if r.StartFromIteration > r.Iterations {
			return fmt.Errorf("StartFromIteration %d is greater than Iterations %d",
				r.StartFromIteration, r.Iterations)
		}
	}
	return nil
}

// Run represents an individual scenario run (many may be in a single instance (of possibly many) of a scenario).
type Run struct {
	// Do not mutate this, this is shared across the entire scenario
	*ScenarioInfo

	// Each run should have a unique iteration.
	Iteration int
	Logger    *zap.SugaredLogger

	// tracks how many attempts have been made for this iteration
	attemptCount int
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
			"KS_Keyword":                   enums.INDEXED_VALUE_TYPE_KEYWORD,
			"KS_Int":                       enums.INDEXED_VALUE_TYPE_INT,
			OmesExecutionIDSearchAttribute: enums.INDEXED_VALUE_TYPE_KEYWORD,
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

// TaskQueueForRun returns the task queue name for the given run ID.
func TaskQueueForRun(runID string) string {
	return "omes-" + runID
}

// EnsureNexusEndpoint returns the Nexus endpoint for this run, creating it if it
// does not already exist. Call this when Nexus is enabled without a named endpoint.
//
// Creating an endpoint changes the namespace, and a capability probe can enable
// Nexus without the user asking for it, so a created endpoint is logged
// distinctly from a reused one: on a shared or production namespace, that line is
// how an operator sees that this run added something.
func (s *ScenarioInfo) EnsureNexusEndpoint(ctx context.Context) (string, error) {
	endpoint, created, err := ensureNexusEndpoint(ctx, s.Client, s.Namespace, s.RunID)
	if err != nil {
		return "", err
	}
	if created {
		s.logf("Created Nexus endpoint %q in namespace %q for this run", endpoint, s.Namespace)
	}
	return endpoint, nil
}

// NexusEndpointForRun returns a sanitized Nexus endpoint name for the given run ID.
func NexusEndpointForRun(runID string) string {
	sanitized := sanitizeRunID.ReplaceAllString(strings.ReplaceAll(runID, "/", "-"), "")
	return "test-nexus-endpoint-" + sanitized
}

// ensureNexusEndpoint creates a Nexus endpoint for the given run, or returns nil if it already exists.
// ensureNexusEndpoint returns the run's endpoint name and whether this call was
// the one that created it.
func ensureNexusEndpoint(ctx context.Context, cl client.Client, namespace, runID string) (string, bool, error) {
	endpointName := NexusEndpointForRun(runID)
	taskQueue := TaskQueueForRun(runID)
	_, err := cl.OperatorService().CreateNexusEndpoint(ctx,
		&operatorservice.CreateNexusEndpointRequest{
			Spec: &nexus.EndpointSpec{
				Name: endpointName,
				Target: &nexus.EndpointTarget{
					Variant: &nexus.EndpointTarget_Worker_{
						Worker: &nexus.EndpointTarget_Worker{
							Namespace: namespace,
							TaskQueue: taskQueue,
						},
					},
				},
			},
		})
	if err != nil {
		if status.Code(err) == codes.AlreadyExists {
			return endpointName, false, nil
		}
		return "", false, err
	}
	return endpointName, true, nil
}

func (r *Run) TaskQueue() string {
	return TaskQueueForRun(r.RunID)
}

// DefaultStartWorkflowOptions gets default start workflow info.
func (r *Run) DefaultStartWorkflowOptions() client.StartWorkflowOptions {
	return client.StartWorkflowOptions{
		TaskQueue:                                TaskQueueForRun(r.RunID),
		ID:                                       fmt.Sprintf("w-%s-%s-%d", r.RunID, r.ExecutionID, r.Iteration),
		WorkflowExecutionErrorWhenAlreadyStarted: !r.Configuration.IgnoreAlreadyStarted,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(OmesExecutionIDSearchAttribute).ValueSet(r.ExecutionID),
		),
	}
}

// DefaultKitchenSinkWorkflowOptions gets the default kitchen sink workflow info.
func (r *Run) DefaultKitchenSinkWorkflowOptions() KitchenSinkWorkflowOptions {
	return KitchenSinkWorkflowOptions{StartOptions: r.DefaultStartWorkflowOptions()}
}

// ShouldRetry determines if another attempt should be made. It returns the backoff duration to wait
// before retrying and a boolean indicating whether a retry should occur.
func (r *Run) ShouldRetry(err error) (time.Duration, bool) {
	r.attemptCount++
	if r.attemptCount >= r.Configuration.MaxIterationAttempts {
		return 0, false
	}
	backoff := min(MaxIterationRetryBackoff, BaseIterationRetryBackoff*time.Duration(1<<uint(r.attemptCount-1)))
	return backoff, true
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
		Namespace:       r.Namespace,
		WorkflowOptions: options.StartOptions,
		WorkflowType:    "kitchenSink",
		WorkflowInput:   options.Params.GetWorkflowInput(),
	}
	startErr := executor.Start(ctx, options.Params.WithStartAction)
	if startErr != nil {
		return fmt.Errorf("failed to start kitchen sink workflow: %w", startErr)
	}

	var clientActionsErrPtr atomic.Pointer[error]
	clientSeq := options.Params.ClientSequence
	if clientSeq != nil && len(clientSeq.ActionSets) > 0 {
		go func() {
			err := executor.ExecuteClientSequence(cancelCtx, clientSeq)
			if err != nil {
				clientActionsErrPtr.Store(&err)
				r.Logger.Error("Client actions failed: ", clientActionsErrPtr)
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
	if clientActionsErr := clientActionsErrPtr.Load(); clientActionsErr != nil {
		return fmt.Errorf("kitchen sink client actions failed: %w", *clientActionsErr)
	}
	return nil
}

// ExecuteAnyWorkflow wraps calls to the client executing workflows to include some logging,
// returning an error if the execution fails.
func (r *Run) ExecuteAnyWorkflow(ctx context.Context, options client.StartWorkflowOptions, workflow any, valuePtr any, args ...any) error {
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
