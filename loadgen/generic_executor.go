package loadgen

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type GenericExecutor struct {
	// Function to execute a single iteration of this scenario
	Execute func(context.Context, *Run) error
	// Default configuration if any.
	DefaultConfiguration RunConfiguration
}

func (g *GenericExecutor) GetDefaultConfiguration() RunConfiguration {
	return g.DefaultConfiguration
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
	r, err := g.newRun(info)
	if err != nil {
		return err
	}
	return r.Run(ctx)
}

func (g *GenericExecutor) newRun(info ScenarioInfo) (*genericRun, error) {
	run := &genericRun{
		executor: g,
		info:     info,
		config:   info.Configuration,
		logger:   info.Logger,
		executeTimer: info.MetricsHandler.WithTags(
			map[string]string{"scenario": info.ScenarioName}).Timer("omes_execute_histogram"),
	}

	// Setup config
	if run.config.Duration < 0 {
		return nil, fmt.Errorf("invalid scenario: Duration cannot be negative")
	}
	if run.config.Duration == 0 && run.config.Iterations == 0 {
		run.config.Duration, run.config.Iterations = g.DefaultConfiguration.Duration, g.DefaultConfiguration.Iterations
	}
	if run.config.MaxConcurrent == 0 {
		run.config.MaxConcurrent = g.DefaultConfiguration.MaxConcurrent
	}
	if run.config.Timeout == 0 {
		run.config.Timeout = g.DefaultConfiguration.Timeout
	}
	if run.config.MaxIterationsPerSecond == 0 {
		run.config.MaxIterationsPerSecond = g.DefaultConfiguration.MaxIterationsPerSecond
	}
	run.config.ApplyDefaults()
	if run.config.Iterations > 0 {
		if run.config.Duration > 0 {
			return nil, fmt.Errorf("invalid scenario: iterations and duration are mutually exclusive")
		}
		if run.config.StartFromIteration > run.config.Iterations {
			return nil, fmt.Errorf("invalid scenario: StartFromIteration %d is greater than Iterations %d",
				run.config.StartFromIteration, run.config.Iterations)
		}
	}

	return run, nil
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
	doneCh := make(chan error)
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
			startTime := time.Now()
			err := g.executor.Execute(ctx, run)
			if err != nil {
				err = fmt.Errorf("iteration %v failed: %w", run.Iteration, err)
				g.logger.Error(err)
			}
			select {
			case <-ctx.Done():
			case doneCh <- err:
				// Record/log here, not if it was cut short by context complete
				g.executeTimer.Record(time.Since(startTime))
			}
		}()
	}

	// Wait for all to be done or an error to occur. We will wait past the overall duration for
	// executions to complete. It is expected that whatever is running omes may choose to enforce
	// a hard timeout if waiting for started executions to complete exceeds a certain threshold.
	g.logger.Info("Run cooldown: stopped starting new iterations; waiting for running ones to complete")
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
