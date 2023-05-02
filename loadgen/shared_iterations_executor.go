package loadgen

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const defaultSharedIterationsConcurrency = 10

type SharedIterationsExecutor struct {
	// Number of instances of the Execute method to run concurrently.
	Concurrency int
	// Number of iterations to run of this scenario (mutually exclusive with Duration).
	Iterations int
	// Duration limit of this scenario (mutually exclusive with Iterations).
	Duration time.Duration
	// Function to execute a single iteration of this scenario.
	Execute func(ctx context.Context, run *Run) error
	// If true, will also include errors even after context complete.
	IncludeErrorsAfterContextComplete bool
}

func calcConcurrency(iterations int, concurrency int) int {
	if concurrency == 0 {
		concurrency = defaultSharedIterationsConcurrency
	}
	if iterations > 0 && concurrency > iterations {
		// Don't spin up more coroutines than the number of total iterations
		concurrency = iterations
	}
	return concurrency
}

type run struct {
	executor         *SharedIterationsExecutor
	options          *RunOptions
	logger           *zap.SugaredLogger
	errors           chan error
	done             sync.WaitGroup
	iterationCounter atomic.Uint64
	// Timer capturing E2E execution of each scenario run iteration.
	executeTimer client.MetricsTimer
}

// Run a scenario.
func (e *SharedIterationsExecutor) Run(ctx context.Context, options *RunOptions) error {
	r, err := e.newRun(options)
	if err != nil {
		return err
	}
	return r.Run(ctx)
}

func (e *SharedIterationsExecutor) newRun(options *RunOptions) (*run, error) {
	iterations := e.Iterations
	duration := e.Duration
	if iterations == 0 && duration == 0 {
		return nil, errors.New("invalid scenario: either iterations or duration is required")
	}
	if iterations > 0 && duration > 0 {
		return nil, errors.New("invalid scenario: iterations and duration are mutually exclusive")
	}

	executeTimer := options.MetricsHandler.WithTags(map[string]string{"scenario": options.ScenarioName}).Timer("omes_execute_histogram")

	return &run{
		executor:     e,
		options:      options,
		logger:       options.Logger,
		errors:       make(chan error),
		executeTimer: executeTimer,
	}, nil
}

// Run a scenario.
// Spins up coroutines according to the scenario configuration.
// Each coroutine runs the scenario Execute method in a loop until the scenario duration or max iterations is reached.
func (r *run) Run(ctx context.Context) error {
	iterations := r.executor.Iterations
	duration := r.executor.Duration
	concurrency := calcConcurrency(iterations, r.executor.Concurrency)

	ctx, cancel := context.WithCancel(ctx)
	if duration > 0 {
		ctx, cancel = context.WithTimeout(ctx, duration)
	}
	defer cancel()

	r.done.Add(concurrency)

	startTime := time.Now()
	waitChan := make(chan struct{})
	go func() {
		r.done.Wait()
		close(waitChan)
	}()

	for i := 0; i < concurrency; i++ {
		logger := r.logger.With("coroID", i)
		go r.runOne(ctx, logger)
	}

	var accumulatedErrors []string

	for {
		select {
		case err := <-r.errors:
			cancel()
			accumulatedErrors = append(accumulatedErrors, err.Error())
		case <-waitChan:
			if len(accumulatedErrors) > 0 {
				return fmt.Errorf("run finished with errors after %s, errors:\n%s", time.Since(startTime), strings.Join(accumulatedErrors, "\n"))
			}
			r.logger.Infof("Run complete in %v", time.Since(startTime))
			return nil
		}
	}
}

// runOne - where "one" is a single routine out of N concurrent defined for the scenario.
// This method will loop until context is cancelled or the number of iterations for the scenario have exhuasted.
func (r *run) runOne(ctx context.Context, logger *zap.SugaredLogger) {
	iterations := r.executor.Iterations
	defer r.done.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		iteration := int(r.iterationCounter.Add(1))
		// If the scenario is limited in number of iterations, do not exceed that number
		if iterations > 0 && iteration > iterations {
			break
		}
		logger.Debugf("Running iteration %d", iteration)
		run := Run{
			Client:          r.options.Client,
			ScenarioName:    r.options.ScenarioName,
			IterationInTest: iteration,
			Logger:          logger.With("iteration", iteration),
			ID:              r.options.RunID,
		}

		startTime := time.Now()
		// TODO: ctx deadline might be too short if scenario is run with the Duration option.
		// Set different duration here.
		if err := r.executor.Execute(ctx, &run); err != nil {
			// Only record the error if the context is not complete or we're including those
			if ctx.Err() == nil || r.executor.IncludeErrorsAfterContextComplete {
				duration := time.Since(startTime)
				r.executeTimer.Record(duration)
				err = fmt.Errorf("iteration %d failed: %w", iteration, err)
				logger.Error(err)
				r.errors <- err
			}
			// Even though context will be cancelled by the runner, we break here to avoid needlessly running another iteration
			break
		}
	}
}
