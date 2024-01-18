package scenarios

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/facebookgo/clock"
	"github.com/pborman/uuid"
	"github.com/temporalio/omes/common"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// This file contains code for a scenario to test workflow completion callbacks. You can test it by running:
// go run ./cmd run-scenario-with-worker --language go --scenario completion_callbacks --option hostName=localhost,startingPort=9000,numCallbackHosts=10,maxDelay=1s,maxErrorProbability=0.1,lambda=1,halfLife=1,dryRun=false --iterations 10 --max-concurrent 10

// CompletionCallbackScenarioOptions are the options for the CompletionCallbackScenario.
type CompletionCallbackScenarioOptions struct {
	// RPS is the maximum number of requests per second to send. This is required and must be > 0.
	RPS int
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
	// CallbackHostName is the host name to use for the callback URL. Do not include the port.
	CallbackHostName string
	// DryRun determines whether this is a dry run. If the value is true, the scenario will not actually execute
	// workflows, but will instead just log what it would have done.
	DryRun bool
	// Lambda is the λ parameter for the exponential distribution function used to determine which host to use for a
	// given workflow. A value close to zero means that all hosts have a similar priority of being selected. The higher
	// the value, the more likely it is that the first host will be selected. This must be > 0.
	Lambda float64
	// HalfLife is τ for the exponential decay function used to determine the delay and error probability for a given
	// host. This means that the delay and error probability will be halved for each subsequent host. This must be > 0.
	// Set it to a very large value to make all hosts have the same delay and error probability. Set it to a very small
	// value to make only the first host have a very large delay and error probability.
	HalfLife float64
	// MaxDelay is the maximum delay to use for a callback. The actual delay will be this value times the exponential
	// decay function of the host index. This must be >= 0.
	MaxDelay time.Duration
	// MaxErrorProbability is the maximum probability that a callback will fail. This is used to simulate a callback
	// that fails to be delivered. The actual probability of failure will be this value times the exponential decay
	// function of the host index. This must be in [0, 1].
	MaxErrorProbability float64
	// AttachWorkflowID determines whether the workflow ID should be attached to the callback URL. This is useful for
	// debugging.
	AttachWorkflowID bool
	// AttachCallbacks determines whether the callback URLs should be attached to the workflow.
	AttachCallbacks bool
}

type completionCallbackScenarioIterationResult struct {
	// WorkflowID of the workflow that was executed.
	WorkflowID string
	// RunID of the workflow that was executed.
	RunID string
	// URL of the callback that was used.
	URL *url.URL
}

type completionCallbackScenarioExecutor struct{}

const (
	// OptionKeyRPS determines CompletionCallbackScenarioOptions.RPS. The default value is 1000.
	OptionKeyRPS = "rps"
	// OptionKeyStartingPort determines CompletionCallbackScenarioOptions.StartingPort.
	OptionKeyStartingPort = "startingPort"
	// OptionKeyNumCallbackHosts determines CompletionCallbackScenarioOptions.NumCallbackHosts.
	OptionKeyNumCallbackHosts = "numCallbackHosts"
	// OptionKeyCallbackHostName determines CompletionCallbackScenarioOptions.CallbackHostName.
	OptionKeyCallbackHostName = "hostName"
	// OptionKeyDryRun determines CompletionCallbackScenarioOptions.DryRun.
	OptionKeyDryRun = "dryRun"
	// OptionKeyLambda determines CompletionCallbackScenarioOptions.Lambda. The default value is 1.0.
	OptionKeyLambda = "lambda"
	// OptionKeyHalfLife determines CompletionCallbackScenarioOptions.HalfLife. The default value is 1.0.
	OptionKeyHalfLife = "halfLife"
	// OptionKeyMaxDelay determines CompletionCallbackScenarioOptions.MaxDelay. The default value is 1s.
	OptionKeyMaxDelay = "maxDelay"
	// OptionKeyMaxErrorProbability determines CompletionCallbackScenarioOptions.MaxErrorProbability.
	// The default value is 0.05.
	OptionKeyMaxErrorProbability = "maxErrorProbability"
	// OptionKeyAttachWorkflowID determines CompletionCallbackScenarioOptions.AttachWorkflowID. The default value is
	// true.
	OptionKeyAttachWorkflowID = "attachWorkflowID"
	// OptionKeyAttachCallbacks determines CompletionCallbackScenarioOptions.AttachCallbacks. The default value is
	// true.
	OptionKeyAttachCallbacks = "attachCallbacks"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "For this scenario, Iterations is not supported and Duration is required. We run a single" +
			" iteration which will spawn a number of workflows, execute them, and verify that all callbacks are" +
			" eventually delivered.",
		Executor: completionCallbackScenarioExecutor{},
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
		return 0, fmt.Errorf("u must be in [0, 1)")
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

// RunCompletionCallbackScenario runs a scenario where each iteration executes a single workflow that has a completion
// callback attached targeting one of a given set of addresses. After each iteration, we query all workflows to verify
// that all callbacks have been delivered.
func RunCompletionCallbackScenario(
	ctx context.Context,
	opts *CompletionCallbackScenarioOptions,
	info loadgen.ScenarioInfo,
) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := validateOptions(opts); err != nil {
		return err
	}
	opts.Logger.Infow("Starting scenario", "options", opts)
	var numSuccesses, numFailures, numStarted, numFinished atomic.Int32
	go func() {
		t := time.NewTicker(time.Second)
		for {
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
				opts.Logger.Infow("Scenario status",
					"numStarted", numStarted.Load(), "numFinished", numFinished.Load(),
					"numSuccesses", numSuccesses.Load(), "numFailures", numFailures.Load(),
				)
			}
		}
	}()
	rateLimiter := rate.NewLimiter(rate.Limit(opts.RPS), opts.RPS)
	results := make([]*completionCallbackScenarioIterationResult, 0, info.Configuration.Iterations)
	l := &loadgen.GenericExecutor{
		Execute: func(ctx context.Context, run *loadgen.Run) error {
			res, err := runWorkflow(ctx, opts, run.DefaultStartWorkflowOptions(), rateLimiter, &numStarted)
			if err != nil {
				return fmt.Errorf("run workflow: %w", err)
			}
			numFinished.Add(1)
			opts.Logger.Debugw("Workflow finished", "url", res.URL.String())
			results = append(results, res)
			return nil
		},
	}
	if err := l.Run(ctx, info); err != nil {
		return fmt.Errorf("completion callback scenario run generic executor: %w", err)
	}
	opts.Logger.Infow("All workflows finished", "numWorkflows", len(results))
	if !opts.AttachCallbacks {
		opts.Logger.Infow("Skipping callback verification because callbacks are not attached")
		return nil
	}
	for _, res := range results {
		opts.Logger.Debugw("Verifying callback succeeded", "url", res.URL.String())
		err := verifyCallbackSucceeded(ctx, opts, rateLimiter, res.WorkflowID, res.RunID, res.URL)
		if err != nil {
			numFailures.Add(1)
			opts.Logger.Errorw("Callback verification failed", "url", res.URL.String(), "error", err)
		} else {
			numSuccesses.Add(1)
			opts.Logger.Debugw("Callback succeeded", "url", res.URL.String())
		}
	}
	if numFailures.Load() > 0 {
		return fmt.Errorf("%d callbacks failed", numFailures.Load())
	}
	return nil
}

func (completionCallbackScenarioExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	opts := &CompletionCallbackScenarioOptions{}
	parseOptions(info.ScenarioOptions, opts)
	opts.Clock = clock.New()
	opts.Logger = info.Logger
	opts.SdkClient = info.Client
	return RunCompletionCallbackScenario(ctx, opts, info)
}

func timed(f func() error) error {
	_, err := timed2(func() (struct{}, error) {
		return struct{}{}, f()
	})
	return err
}

func timed2[T any](f func() (T, error)) (T, error) {
	now := time.Now()
	t, err := f()
	if err != nil {
		if strings.Contains(err.Error(), "deadline") {
			return t, fmt.Errorf("deadline exceeded after %v", time.Since(now))
		}
		return t, err
	}
	return t, nil
}

func runWorkflow(
	ctx context.Context,
	scenarioOptions *CompletionCallbackScenarioOptions,
	startWorkflowOptions client.StartWorkflowOptions,
	limiter *rate.Limiter,
	numStarted *atomic.Int32,
) (*completionCallbackScenarioIterationResult, error) {
	workflowID := uuid.New()
	u, err := generateURLFromOptions(scenarioOptions, workflowID)
	if err != nil {
		return nil, fmt.Errorf("generate callback URL: %w", err)
	}

	if scenarioOptions.DryRun {
		return nil, nil
	}

	if scenarioOptions.AttachCallbacks {
		completionCallbacks := []*commonpb.Callback{{
			Variant: &commonpb.Callback_Nexus_{
				Nexus: &commonpb.Callback_Nexus{
					Url: u.String(),
				},
			},
		}}
		startWorkflowOptions.CompletionCallbacks = completionCallbacks
	}
	startWorkflowOptions.ID = workflowID
	input := &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			kitchensink.NoOpSingleActivityActionSet(),
		},
	}
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("wait for rate limiter to start workflow: %w", err)
	}
	workflowRun, err := timed2(func() (client.WorkflowRun, error) {
		return scenarioOptions.SdkClient.ExecuteWorkflow(ctx, startWorkflowOptions, common.WorkflowNameKitchenSink, input)
	})
	if err != nil {
		return nil, fmt.Errorf("start workflow with completion callback: %w", err)
	}
	numStarted.Add(1)
	if err := limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("wait for rate limiter to get workflow completion: %w", err)
	}
	err = workflowRun.Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("wait for workflow with completion callback: %w", err)
	}

	return &completionCallbackScenarioIterationResult{
		WorkflowID: workflowRun.GetID(),
		RunID:      workflowRun.GetRunID(),
		URL:        u,
	}, nil
}

func verifyCallbackSucceeded(
	ctx context.Context,
	options *CompletionCallbackScenarioOptions,
	limiter *rate.Limiter,
	workflowID string,
	runID string,
	u *url.URL,
) error {
	retryDelay := time.Millisecond * 10
	for {
		if err := limiter.Wait(ctx); err != nil {
			return fmt.Errorf("wait for rate limiter to verify callback succeeded: %w", err)
		}
		execution, err := options.SdkClient.DescribeWorkflowExecution(ctx, workflowID, runID)
		if err != nil {
			return fmt.Errorf("verify callback succeeded describe workflow: %w", err)
		}
		callbacks := execution.Callbacks
		if len(callbacks) != 1 {
			callbacksString := ""
			for i, callback := range callbacks {
				callbacksString += fmt.Sprintf("%d: %t: %+v\n", i, callback == nil, callback)
			}
			return fmt.Errorf("expected 1 callback, got %d: %s", len(callbacks), callbacksString)
		}
		callback := callbacks[0]
		if callback.State == enums.CALLBACK_STATE_SUCCEEDED {
			if callback.Callback.GetNexus().Url != u.String() {
				return fmt.Errorf("expected callback URL %q, got %q", u.String(), callback.Callback.GetNexus().Url)
			}
			return nil
		}
		if callback.State == enums.CALLBACK_STATE_BACKING_OFF {
			options.Logger.Debugw("Callback backing off", "failure", callback.LastAttemptFailure)
		}
		if callback.State == enums.CALLBACK_STATE_FAILED {
			return fmt.Errorf("callback failed: %+v", callback.LastAttemptFailure)
		}
		timer := options.Clock.Timer(retryDelay)
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
	if options.RPS <= 0 {
		errs = append(errs, fmt.Errorf("%q is required and must be > 0", OptionKeyRPS))
	}
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

// parseOptions parses the options for this scenario from the given map.
func parseOptions(m map[string]string, options *CompletionCallbackScenarioOptions) *CompletionCallbackScenarioOptions {
	options.RPS = loadgen.ScenarioOptionInt(m, OptionKeyRPS, 1000)
	options.StartingPort = loadgen.ScenarioOptionInt(m, OptionKeyStartingPort, 0)
	options.NumCallbackHosts = loadgen.ScenarioOptionInt(m, OptionKeyNumCallbackHosts, 0)
	options.CallbackHostName = m[OptionKeyCallbackHostName]
	options.DryRun = loadgen.ScenarioOptionBool(m, OptionKeyDryRun, false)
	options.Lambda = loadgen.ScenarioOptionFloat64(m, OptionKeyLambda, 1.0)
	options.HalfLife = loadgen.ScenarioOptionFloat64(m, OptionKeyHalfLife, 1.0)
	options.MaxDelay = loadgen.ScenarioOptionDuration(m, OptionKeyMaxDelay, time.Second*1)
	options.MaxErrorProbability = loadgen.ScenarioOptionFloat64(m, OptionKeyMaxErrorProbability, 0.05)
	options.AttachWorkflowID = loadgen.ScenarioOptionBool(m, OptionKeyAttachWorkflowID, true)
	options.AttachCallbacks = loadgen.ScenarioOptionBool(m, OptionKeyAttachCallbacks, true)
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
