package loadgen

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type GenericExecutor struct {
	// Function to execute a single iteration of this scenario
	Execute func(context.Context, *Run) error
}

type genericRun struct {
	executor *GenericExecutor
	info     ScenarioInfo
	config   RunConfiguration
	logger   *zap.SugaredLogger
	// Timer capturing E2E execution of each scenario run iteration.
	executeTimer client.MetricsTimer
	// Iteration outcome tallies, used for the end-of-run summary. failed counts
	// only terminal failures tolerated via ContinueOnIterationFailure.
	completed atomic.Int64
	failed    atomic.Int64
}

func (g *GenericExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	r, err := g.newRun(info)
	if err != nil {
		return err
	}
	return r.Run(ctx)
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
				runErr = err
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
			iterStart := time.Now()

			defer func() {
				elapsed := time.Since(iterStart)
				g.executeTimer.Record(elapsed)

				// Swallow the error for the spawn loop only, so a tolerated failure
				// doesn't stop new iterations from starting. It is not ignored: it is
				// counted (g.failed) and surfaced via OnIterationFailure, and the run
				// still fails at the end if the tally is nonzero.
				iterErr := err
				if iterErr != nil && g.config.ContinueOnIterationFailure {
					err = nil
				}

				// An iteration cut short because the run is stopping is left out of
				// both tallies.
				//
				// Both conditions are needed: the run's context is also done once a
				// run simply finishes, and an iteration can fail with its own
				// cancellation while the run is healthy.
				stopping := iterErr != nil && ctx.Err() != nil && errors.Is(iterErr, context.Canceled)

				select {
				case <-ctx.Done():
				case doneCh <- err:
					switch {
					case stopping:
						g.logger.Debugf("Iteration %v abandoned: run is stopping", run.Iteration)
					case iterErr == nil:
						run.Duration = elapsed
						g.completed.Add(1)
						if g.config.OnCompletion != nil {
							g.config.OnCompletion(ctx, run)
						}
					default:
						g.failed.Add(1)
						if g.config.OnIterationFailure != nil {
							g.config.OnIterationFailure(ctx, run, iterErr)
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
				if err == nil {
					break
				}

				backoff, retry := run.ShouldRetry(err)
				if retry {
					// Transient: this attempt is superseded by the retry and
					// never propagated, so log it here as the only record.
					err = fmt.Errorf("iteration %v encountered error: %w", run.Iteration, err)
					g.logger.Error(err)
				} else {
					// Terminal: propagated via doneCh and reported by Run's
					// caller, so don't also log it here.
					err = fmt.Errorf("iteration %v failed: %w", run.Iteration, err)
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
	// ContinueOnIterationFailure changed only when the run stops (it ran to
	// completion instead of aborting on the first failure); the verdict is
	// unchanged — any terminal iteration failure fails the run. Per-iteration
	// outcomes were tallied into the snapshot, so the caller can read the
	// success/failure counts and apply its own policy on top.
	if failed := g.failed.Load(); failed > 0 {
		completed := g.completed.Load()
		g.logger.Infof("Run completed in %v: %d iterations succeeded, %d failed",
			time.Since(startTime), completed, failed)
		return fmt.Errorf("run completed with %d of %d iterations failed", failed, completed+failed)
	}
	g.logger.Infof("Run completed in %v", time.Since(startTime))
	return nil
}
