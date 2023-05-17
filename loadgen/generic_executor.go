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
	Execute func(ctx context.Context, run *Run) error
	// Default configuration if any.
	DefaultConfiguration RunConfiguration
}

func (g GenericExecutor) GetDefaultConfiguration() RunConfiguration { return g.DefaultConfiguration }

type genericRun struct {
	executor *GenericExecutor
	options  RunOptions
	config   RunConfiguration
	logger   *zap.SugaredLogger
	// Timer capturing E2E execution of each scenario run iteration.
	executeTimer client.MetricsTimer
}

// Run a scenario.
func (g GenericExecutor) Run(ctx context.Context, options RunOptions) error {
	r, err := g.newRun(options)
	if err != nil {
		return err
	}
	return r.Run(ctx)
}

func (g *GenericExecutor) newRun(options RunOptions) (*genericRun, error) {
	run := &genericRun{
		executor: g,
		options:  options,
		config:   options.Configuration,
		logger:   options.Logger,
		executeTimer: options.MetricsHandler.WithTags(
			map[string]string{"scenario": options.ScenarioName}).Timer("omes_execute_histogram"),
	}

	// Setup config
	if run.config.Duration == 0 && run.config.Iterations == 0 {
		run.config.Duration, run.config.Iterations = g.DefaultConfiguration.Duration, g.DefaultConfiguration.Iterations
	}
	if run.config.MaxConcurrent == 0 {
		run.config.MaxConcurrent = g.DefaultConfiguration.MaxConcurrent
	}
	run.config.ApplyDefaults()
	if run.config.Iterations > 0 && run.config.Duration > 0 {
		return nil, fmt.Errorf("invalid scenario: iterations and duration are mutually exclusive")
	}

	return run, nil
}

// Run a scenario.
// Spins up coroutines according to the scenario configuration.
// Each coroutine runs the scenario Execute method in a loop until the scenario duration or max iterations is reached.
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
	for i := 0; runErr == nil && ctx.Err() == nil && (g.config.Iterations == 0 || i < g.config.Iterations); i++ {
		// If there are already more running than max concurrent, wait for one
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
		run := &Run{
			Client:          g.options.Client,
			ScenarioName:    g.options.ScenarioName,
			IterationInTest: i + 1,
			Logger:          g.logger.With("iteration", i),
			ID:              g.options.RunID,
			RunOptions:      &g.options,
		}
		go func() {
			startTime := time.Now()
			err := g.executor.Execute(ctx, run)
			// Only log/wrap/send to channel if context is not done
			if ctx.Err() == nil {
				if err != nil {
					err = fmt.Errorf("iteration %v failed: %w", run.IterationInTest, err)
					g.logger.Error(err)
				}
				select {
				case <-ctx.Done():
				case doneCh <- err:
					// Record/log here, not if it was cut short by context complete
					g.executeTimer.Record(time.Since(startTime))
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
