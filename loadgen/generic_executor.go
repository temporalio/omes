package loadgen

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type GenericExecutor struct {
	// Function to execute a single iteration of this scenario
	Execute func(context.Context, *Run) error

	// WorkflowCompletionChecker verifies the workflow completion count after a scenario completed.
	// It is optional; and needs to be initialized by the scenario executor.
	WorkflowCompletionChecker *WorkflowCompletionChecker

	// State management
	mu    sync.Mutex
	state *ExecutorState
}

type genericRun struct {
	executor *GenericExecutor
	info     ScenarioInfo
	config   RunConfiguration
	logger   *zap.SugaredLogger
	// Timer capturing E2E execution of each scenario run iteration.
	executeTimer client.MetricsTimer
}

func (g *GenericExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	g.mu.Lock()
	if g.state == nil {
		g.state = &ExecutorState{}
	}
	if g.state.StartedAt.IsZero() {
		g.state.StartedAt = time.Now()
	}
	g.mu.Unlock()

	r, err := g.newRun(info)
	if err != nil {
		return err
	}
	return r.Run(ctx)
}

func (g *GenericExecutor) RecordCompletion() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != nil {
		g.state.CompletedIterations += 1
		g.state.LastCompletedAt = time.Now()
	}
}

func (g *GenericExecutor) RecordError(err error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state != nil && err != nil {
		g.state.IterationErrors = append(g.state.IterationErrors, err.Error())
	}
}

func (g *GenericExecutor) VerifyRun(ctx context.Context, info ScenarioInfo) []error {
	g.mu.Lock()
	state := *g.state
	checker := g.WorkflowCompletionChecker
	g.mu.Unlock()

	if checker == nil {
		return nil
	}
	if err := checker.Verify(ctx, info, state); err != nil {
		return []error{err}
	}
	return nil
}

// GetState returns a copy of the current state
func (g *GenericExecutor) GetState() ExecutorState {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == nil {
		return ExecutorState{}
	}
	return *g.state
}

func (g *GenericExecutor) Snapshot() any {
	return g.GetState()
}

func (g *GenericExecutor) LoadState(loader func(any) error) error {
	var state ExecutorState
	if err := loader(&state); err != nil {
		return err
	}

	g.mu.Lock()
	g.state = &state
	g.mu.Unlock()

	return nil
}

func (g *GenericExecutor) newRun(info ScenarioInfo) (*genericRun, error) {
	info.Configuration.ApplyDefaults()
	if err := info.Configuration.Validate(); err != nil {
		return nil, fmt.Errorf("invalid scenario: %w", err)
	}
	return &genericRun{
		executor: g,
		info:     info,
		config:   info.Configuration,
		logger:   info.Logger,
		executeTimer: info.MetricsHandler.WithTags(
			map[string]string{"scenario": info.ScenarioName}).Timer("omes_execute_histogram"),
	}, nil
}

// Run a scenario.
// Spins up coroutines according to the scenario configuration.
// Each coroutine runs the scenario Execute method in a loop until the scenario duration or max
// iterations is reached.
func (g *genericRun) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	if !g.info.Configuration.DoNotRegisterSearchAttributes {
		err := g.info.RegisterDefaultSearchAttributes(ctx)
		if err != nil {
			return err
		}
	}

	// If set, timeout overrides the default context.
	if g.config.Timeout > 0 {
		g.logger.Debugf("Will timeout after %v", g.config.Timeout)
		ctx, cancel = context.WithTimeout(ctx, g.config.Timeout)
		defer cancel()
	}

	durationCtx := ctx
	if g.config.Duration > 0 {
		durationCtx, cancel = context.WithTimeout(ctx, g.config.Duration)
		defer cancel()
	}

	startTime := time.Now()
	var runErr error
	doneCh := make(chan error, g.config.MaxConcurrent)
	var currentlyRunning int
	waitOne := func(contextToWaitOn context.Context) {
		select {
		case err := <-doneCh:
			currentlyRunning--
			if err != nil {
				if g.config.ContinueOnError {
					g.logger.Warnf("Iteration failed but continuing due to --continue-on-error: %v", err)
					g.executor.RecordError(err)
				} else {
					runErr = err
				}
			}
		case <-contextToWaitOn.Done():
		}
	}

	if g.config.Iterations > 0 {
		g.logger.Debugf("Will run %v iterations", g.config.Iterations)
	} else {
		g.logger.Debugf("Will start iterations for %v", g.config.Duration)
	}

	var rateLimiter <-chan time.Time
	if g.config.MaxIterationsPerSecond > 0 {
		g.logger.Debugf("Will run at rate of %v iteration(s) per second", g.config.MaxIterationsPerSecond)
		rateLimiter = time.Tick(time.Duration(float64(time.Second) / g.config.MaxIterationsPerSecond))
	}

	// Run all until we've gotten an error or reached iteration limit or timeout
	for i := g.info.Configuration.StartFromIteration; runErr == nil && durationCtx.Err() == nil &&
		(g.config.Iterations == 0 || i < g.config.Iterations); i++ {

		// If there is a rate limit, enforce it
		if rateLimiter != nil {
			<-rateLimiter
		}

		// If there are already MaxConcurrent running, wait for one
		if currentlyRunning >= g.config.MaxConcurrent {
			waitOne(durationCtx)
			// Exit loop on first error, or if scenario duration has elapsed
			if runErr != nil || durationCtx.Err() != nil {
				break
			}
		}

		// Run concurrently
		g.logger.Debugf("Running iteration %v", i)
		currentlyRunning++
		run := g.info.NewRun(i + 1)
		go func() {
			var err error
			var shouldRecordCompletion bool
			iterStart := time.Now()

			defer func() {
				g.executeTimer.Record(time.Since(iterStart))

				select {
				case <-ctx.Done():
				case doneCh <- err:
					if err == nil && shouldRecordCompletion {
						g.executor.RecordCompletion()
						if g.config.OnCompletion != nil {
							g.config.OnCompletion(ctx, run)
						}
					}
				}
			}()

		retryLoop:
			for {
				err = g.executor.Execute(ctx, run)
				if err != nil && g.config.HandleExecuteError != nil {
					err = g.config.HandleExecuteError(ctx, run, err)
				}

				// Check if workflow was already started
				if err != nil {
					var alreadyStartedErr *serviceerror.WorkflowExecutionAlreadyStarted
					if errors.As(err, &alreadyStartedErr) {
						if g.config.IgnoreAlreadyStarted {
							g.logger.Debugf("Workflow already started, skipping iteration %v", run.Iteration)
							err = nil
							shouldRecordCompletion = false
							break
						}
					}
				}

				if err == nil {
					shouldRecordCompletion = true
					break
				}

				backoff, retry := run.ShouldRetry(err)
				if retry {
					err = fmt.Errorf("iteration %v encountered error: %w", run.Iteration, err)
					g.logger.Error(err)
				} else {
					err = fmt.Errorf("iteration %v failed: %w", run.Iteration, err)
					g.logger.Error(err)
					break retryLoop
				}

				select {
				case <-time.After(backoff):
					// wait for backoff, then try again
				case <-ctx.Done():
					break retryLoop
				}
			}
		}()
	}

	// Wait for all to be done or an error to occur. We will wait past the overall duration for
	// executions to complete. It is expected that whatever is running omes may choose to enforce
	// a hard timeout if waiting for started executions to complete exceeds a certain threshold.
	g.logger.Infof("Run cooldown: stopped starting new iterations and waiting for %d iterations to complete",
		currentlyRunning)
	for runErr == nil && currentlyRunning > 0 {
		waitOne(ctx)
		if ctx.Err() != nil {
			return fmt.Errorf("timed out while waiting for runs to complete: %w", ctx.Err())
		}
	}
	if runErr != nil {
		return fmt.Errorf("run finished with error after %v: %w", time.Since(startTime), runErr)
	}
	g.logger.Infof("Run completed in %v", time.Since(startTime))
	return nil
}
