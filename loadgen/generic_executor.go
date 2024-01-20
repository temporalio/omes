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
	if run.config.Duration == 0 && run.config.Iterations == 0 {
		run.config.Duration, run.config.Iterations = g.DefaultConfiguration.Duration, g.DefaultConfiguration.Iterations
	}
	if run.config.MaxConcurrent == 0 {
		run.config.MaxConcurrent = g.DefaultConfiguration.MaxConcurrent
	}
	if run.config.Limiter == nil {
		run.config.Limiter = g.DefaultConfiguration.Limiter
	}
	run.config.ApplyDefaults()
	if run.config.Iterations > 0 && run.config.Duration > 0 {
		return nil, fmt.Errorf("invalid scenario: iterations and duration are mutually exclusive")
	}

	return run, nil
}

// Run a scenario.
// Spins up coroutines according to the scenario configuration.
// Each coroutine runs the scenario Execute method in a loop until the scenario duration or max
// iterations is reached.
func (g *genericRun) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	if g.config.Duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, g.config.Duration)
	}
	defer cancel()

	startTime := time.Now()
	var runErr error
	doneCh := make(chan error)
	var currentlyRunning int
	waitOne := func() {
		select {
		case err := <-doneCh:
			currentlyRunning--
			if err != nil {
				runErr = err
			}
		case <-ctx.Done():
		}
	}

	// Run all until we've gotten an error or reached iteration limit
	for i := 0; runErr == nil && ctx.Err() == nil &&
		(g.config.Iterations == 0 || i < g.config.Iterations); i++ {
		// If there are already MaxConcurrent running, wait for one
		if currentlyRunning >= g.config.MaxConcurrent {
			waitOne()
			// Exit loop if error
			if runErr != nil || ctx.Err() != nil {
				break
			}
		}
		// Run concurrently
		g.logger.Debugf("Running iteration %v", i)
		currentlyRunning++
		run := g.info.NewRun(i + 1)
		go func() {
			var runStartTime time.Time
			err := func() error {
				if g.config.Limiter != nil {
					if innerErr := g.config.Limiter.Wait(ctx); innerErr != nil {
						return innerErr
					}
				}
				runStartTime = time.Now()
				return g.executor.Execute(ctx, run)
			}()
			// Only log/wrap/send to channel if context is not done
			if ctx.Err() == nil {
				if err != nil {
					err = fmt.Errorf("iteration %v failed: %w", run.Iteration, err)
					g.logger.Error(err)
				}
				select {
				case <-ctx.Done():
				case doneCh <- err:
					// Record/log here, not if it was cut short by context complete
					g.executeTimer.Record(time.Since(runStartTime))
				}
			}
		}()
	}
	// Wait for all to be done or an error to occur
	for runErr == nil && ctx.Err() == nil && currentlyRunning > 0 {
		waitOne()
	}
	if runErr != nil {
		return fmt.Errorf("run finished with error after %v: %w", time.Since(startTime), runErr)
	}
	g.logger.Infof("Run complete in %v", time.Since(startTime))
	return nil
}
