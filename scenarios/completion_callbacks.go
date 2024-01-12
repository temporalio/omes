package scenarios

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"time"

	"github.com/facebookgo/clock"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/temporalio/omes/common"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/multierr"
	"go.uber.org/zap"
)

// This file contains code for a scenario to test workflow completion callbacks. You can test it by running:
// go run ./cmd run-scenario-with-worker --language go --scenario completion_callbacks --option hostName=localhost,startingPort=9000,numCallbackHosts=10,maxDelay=1s,maxErrorProbability=0.1,lambda=1,halfLife=1,dryRun=false --iterations 10 --max-concurrent 10

// CompletionCallbackScenarioOptions are the options for the CompletionCallbackScenario.
type CompletionCallbackScenarioOptions struct {
	// Logger must be non-nil.
	Logger *zap.SugaredLogger
	// SdkClient is the client to use to start workflows. This is required.
	SdkClient client.Client
	// Clock is for retry delays. This is required.
	Clock clock.Clock
	// StartingPort is the port of the first host to use for callbacks. Each host will use a different port starting
	// from this value. This is required and must be in [1024, 65535].
	StartingPort int
	// NumCallbackHosts is the number of hosts to use for callbacks. This is required and must be > 0.
	NumCallbackHosts int
	// CallbackHostName is the host name to use for the callback URL. Defaults to "localhost". Do not include the port.
	CallbackHostName string
	// DryRun determines whether this is a dry run. If the value is true, the scenario will not actually execute
	// workflows, but will instead just log what it would have done.
	DryRun bool
	// Lambda is the λ parameter for the exponential distribution function used to determine which host to use for a
	// given workflow. The default value is 1. A value of 0 means that all hosts have the same priority of being
	// selected. The higher the value, the more likely it is that the first host will be selected. This must be >= 0.
	Lambda float64
	// HalfLife is τ for the exponential decay function used to determine the delay and error probability for a given
	// host. The default value is 1. This means that the delay and error probability will be halved for each subsequent
	// host. This must be > 0. Set it to a very large value to make all hosts have the same delay and error probability.
	// Set it to a very small value to make only the first host have a very large delay and error probability.
	HalfLife float64
	// MaxDelay is the maximum delay to use for a callback. The actual delay will be this value times the exponential
	// decay function of the host index. This must be >= 0.
	MaxDelay time.Duration
	// MaxErrorProbability is the maximum probability that a callback will fail. This is used to simulate a callback
	// that fails to be delivered. The actual probability of failure will be this value times the exponential decay
	// function of the host index. This must be in [0, 1].
	MaxErrorProbability float64
	// AttachWorkflowID determines whether the workflow ID should be attached to the callback URL. This is useful for
	// debugging. The default value is true.
	AttachWorkflowID bool
}

// CompletionCallbackScenario is a scenario where each iteration executes a single workflow that has a completion
// callback attached targeting one of a given set of addresses. After each iteration, we query all workflows to verify
// that all callbacks have been delivered.
type CompletionCallbackScenario struct {
	options *CompletionCallbackScenarioOptions
}

type completionCallbackScenarioExecutor struct{}

const (
	// OptionKeyStartingPort determines CompletionCallbackScenarioOptions.StartingPort.
	OptionKeyStartingPort = "startingPort"
	// OptionKeyNumCallbackHosts determines CompletionCallbackScenarioOptions.NumCallbackHosts.
	OptionKeyNumCallbackHosts = "numCallbackHosts"
	// OptionKeyCallbackHostName determines CompletionCallbackScenarioOptions.CallbackHostName.
	OptionKeyCallbackHostName = "hostName"
	// OptionKeyDryRun determines CompletionCallbackScenarioOptions.DryRun.
	OptionKeyDryRun = "dryRun"
	// OptionKeyLambda determines CompletionCallbackScenarioOptions.Lambda.
	OptionKeyLambda = "lambda"
	// OptionKeyHalfLife determines CompletionCallbackScenarioOptions.HalfLife.
	OptionKeyHalfLife = "halfLife"
	// OptionKeyMaxDelay determines CompletionCallbackScenarioOptions.MaxDelay.
	OptionKeyMaxDelay = "maxDelay"
	// OptionKeyMaxErrorProbability determines CompletionCallbackScenarioOptions.MaxErrorProbability.
	OptionKeyMaxErrorProbability = "maxErrorProbability"
	// OptionKeyAttachWorkflowID determines CompletionCallbackScenarioOptions.AttachWorkflowID.
	OptionKeyAttachWorkflowID = "attachWorkflowID"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "For this scenario, Iterations is not supported and Duration is required. We run a single" +
			" iteration which will spawn a number of workflows, execute them, and verify that all callbacks are" +
			" eventually delivered.",
		Executor: newExecutor(),
	})
}

// ExponentialSample returns a sample from an exponential distribution with the given lambda.
// The cdf of the exponential distribution is:
//
//	cdf(x) = 1 - lambda * exp(-lambda * x)
//
// Here, the probability that the returned value is equal to `i` is:
// 0 if i < 0 or i > n
// (cdf(i+1)-cdf(i))/cdf(n) if 0 <= i < n
//
// The `u` parameter should be a uniform random number in [0, 1).
func ExponentialSample(n int, lambda float64, u float64) (int, error) {
	if u < 0 || u >= 1 {
		return 0, errors.Errorf("u must be in [0, 1)")
	}
	totalProbability := 1 - math.Exp(-lambda*float64(n))
	for i := 1; i < n; i++ {
		cdf := 1 - math.Exp(-lambda*float64(i))
		if u <= cdf/totalProbability {
			return i - 1, nil
		}
	}
	return n - 1, nil
}

// NewCompletionCallbackScenario creates a new CompletionCallbackScenario or an error if the options are invalid.
func NewCompletionCallbackScenario(opts *CompletionCallbackScenarioOptions) (*CompletionCallbackScenario, error) {
	err := validateOptions(opts)
	if err != nil {
		return nil, err
	}
	return &CompletionCallbackScenario{
		options: opts,
	}, nil
}

// RunIteration runs a single iteration of this scenario.
func (s *CompletionCallbackScenario) RunIteration(
	ctx context.Context,
	startWorkflowOptions client.StartWorkflowOptions,
) error {
	workflowID := uuid.New()
	u, err := generateURLFromOptions(s.options, workflowID)
	if err != nil {
		return err
	}

	s.options.Logger.Debugw("Using callback URL", "url", u.String())

	if s.options.DryRun {
		return nil
	}

	completionCallbacks := []*commonpb.Callback{{
		Variant: &commonpb.Callback_Nexus_{
			Nexus: &commonpb.Callback_Nexus{
				Url: u.String(),
			},
		},
	}}
	startWorkflowOptions.CompletionCallbacks = completionCallbacks
	startWorkflowOptions.ID = workflowID
	input := &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			kitchensink.NoOpSingleActivityActionSet(),
		},
	}
	workflowRun, err := s.options.SdkClient.ExecuteWorkflow(ctx, startWorkflowOptions, common.WorkflowNameKitchenSink, input)
	if err != nil {
		return errors.Wrap(err, "failed to execute workflow")
	}
	err = workflowRun.Get(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to get workflow result")
	}

	return s.verifyCallbackSucceeded(ctx, workflowRun, completionCallbacks)
}

func newExecutor() loadgen.Executor {
	return completionCallbackScenarioExecutor{}
}

func (completionCallbackScenarioExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	opts := loadOptions(info.ScenarioOptions)
	opts.Clock = clock.New()
	opts.Logger = info.Logger
	opts.SdkClient = info.Client
	l := &loadgen.GenericExecutor{
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			opts.SdkClient = run.Client
			opts.Logger = run.Logger
			scenario, err := NewCompletionCallbackScenario(opts)
			if err != nil {
				return err
			}
			return scenario.RunIteration(ctx, run.DefaultStartWorkflowOptions())
		},
	}
	return l.Run(ctx, info)
}

// verifyCallbackSucceeded verifies that all callbacks have been delivered by querying the workflow and checking the
// Callbacks field. It will retry until all callbacks have been delivered or until the context is canceled.
func (s *CompletionCallbackScenario) verifyCallbackSucceeded(
	ctx context.Context,
	workflowRun client.WorkflowRun,
	completionCallbacks []*commonpb.Callback,
) error {
	retryDelay := time.Millisecond * 10
	for {
		execution, err := s.options.SdkClient.DescribeWorkflowExecution(ctx, workflowRun.GetID(), workflowRun.GetRunID())
		if err != nil {
			return errors.Wrap(err, "failed to describe workflow")
		}
		callbacks := execution.Callbacks
		if len(callbacks) != len(completionCallbacks) {
			callbacksString := ""
			for i, callback := range callbacks {
				callbacksString += fmt.Sprintf("%d: %t: %+v\n", i, callback == nil, callback)
			}
			return errors.Errorf("expected %d callbacks, got %d: %s", len(completionCallbacks), len(callbacks), callbacksString)
		}
		allSucceeded := true
		anyFailed := false
		for _, callback := range callbacks {
			if callback.State != enums.CALLBACK_STATE_SUCCEEDED {
				allSucceeded = false
			}
			if callback.State == enums.CALLBACK_STATE_BACKING_OFF {
				s.options.Logger.Infow("Callback backing off", "failure", callback.LastAttemptFailure)
			}
			if callback.State == enums.CALLBACK_STATE_FAILED {
				anyFailed = true
				s.options.Logger.Errorw("Callback failed", "failure", callback.LastAttemptFailure)
			}
		}
		if anyFailed {
			return errors.New("one or more callbacks failed")
		}
		if allSucceeded {
			return nil
		}
		timer := s.options.Clock.Timer(retryDelay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
		retryDelay *= 2
	}
}

// validateOptions validates the options for this scenario.
func validateOptions(options *CompletionCallbackScenarioOptions) error {
	var errs []error
	if options.StartingPort < 1024 || options.StartingPort >= 65535 {
		errs = append(errs, fmt.Errorf("%q is required and must be in [1024, 65535]", OptionKeyStartingPort))
	}
	if options.NumCallbackHosts <= 0 {
		errs = append(errs, fmt.Errorf("%q is required and must be > 0", OptionKeyNumCallbackHosts))
	}
	if options.CallbackHostName == "" {
		errs = append(errs, fmt.Errorf("%q is required", OptionKeyCallbackHostName))
	}
	if options.Lambda <= 0 {
		errs = append(errs, fmt.Errorf("%q must be > 0", OptionKeyLambda))
	}
	if options.HalfLife <= 0 {
		errs = append(errs, fmt.Errorf("%q must be > 0", OptionKeyHalfLife))
	}
	if options.MaxDelay < 0 {
		errs = append(errs, fmt.Errorf("%q must be >= 0s", OptionKeyMaxDelay))
	}
	if options.MaxErrorProbability < 0 || options.MaxErrorProbability > 1 {
		errs = append(errs, fmt.Errorf("%q must be in [0, 1]", OptionKeyMaxErrorProbability))
	}
	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}

	return nil
}

// loadOptions loads the options for this scenario from the given map.
func loadOptions(scenarioOptions map[string]string) *CompletionCallbackScenarioOptions {
	options := &CompletionCallbackScenarioOptions{
		StartingPort:        loadgen.ScenarioOptionInt(scenarioOptions, OptionKeyStartingPort, 0),
		NumCallbackHosts:    loadgen.ScenarioOptionInt(scenarioOptions, OptionKeyNumCallbackHosts, 0),
		CallbackHostName:    scenarioOptions[OptionKeyCallbackHostName],
		DryRun:              loadgen.ScenarioOptionBool(scenarioOptions, OptionKeyDryRun, false),
		Lambda:              loadgen.ScenarioOptionFloat64(scenarioOptions, OptionKeyLambda, 1.0),
		HalfLife:            loadgen.ScenarioOptionFloat64(scenarioOptions, OptionKeyHalfLife, 1.0),
		MaxDelay:            loadgen.ScenarioOptionDuration(scenarioOptions, OptionKeyMaxDelay, time.Second*5),
		MaxErrorProbability: loadgen.ScenarioOptionFloat64(scenarioOptions, OptionKeyMaxErrorProbability, 0.0),
		AttachWorkflowID:    loadgen.ScenarioOptionBool(scenarioOptions, OptionKeyAttachWorkflowID, true),
	}
	return options
}

// generateURLFromOptions generates a callback URL from the given options.
func generateURLFromOptions(options *CompletionCallbackScenarioOptions, workflowID string) (*url.URL, error) {
	hostIndex, err := ExponentialSample(options.NumCallbackHosts, options.Lambda, rand.Float64())
	if err != nil {
		return nil, err
	}

	// The decayLambda is different from the lambda parameter used to select the host.
	// https://en.wikipedia.org/wiki/Exponential_decay#Half-life
	decayLambda := math.Ln2 / options.HalfLife

	callbackDelay := time.Duration(options.MaxDelay.Seconds() * math.Exp(-decayLambda*float64(hostIndex)) * float64(time.Second))

	errorProbability := options.MaxErrorProbability * math.Exp(-decayLambda*float64(hostIndex))

	port := options.StartingPort + hostIndex

	q := url.Values{}
	q.Add("delay", fmt.Sprintf("%s", callbackDelay))
	q.Add("failure-probability", fmt.Sprintf("%f", errorProbability))
	if options.AttachWorkflowID {
		q.Add("workflow-id", workflowID)
	}
	u := &url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("%s:%d", options.CallbackHostName, port),
		RawQuery: q.Encode(),
	}
	return u, nil
}
