package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/temporalio/omes/scenario"
	"github.com/temporalio/omes/shared"
	"go.opentelemetry.io/otel/attribute"
	metrics "go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/metric/instrument"
	"go.opentelemetry.io/otel/metric/instrument/syncint64"
	"go.temporal.io/api/batch/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const DEFAULT_CONCURRENCY = 10

var Meter = metrics.Meter("omes")

// Options for creating a Runner
type Options struct {
	ClientOptions client.Options
	// ID used for prefixing workflow IDs and determining the task queue
	RunID    string
	Scenario *scenario.Scenario
}

type RunnerMetrics struct {
	executeHistogram  syncint64.Histogram
	scenarioAttribute attribute.KeyValue
}

type Runner struct {
	iterationCounter atomic.Uint64
	done             sync.WaitGroup
	options          Options
	errors           chan error
	logger           *zap.SugaredLogger
	metrics          *RunnerMetrics
}

// NewRunner instantiates a Runner
func NewRunner(options Options, logger *zap.SugaredLogger) (*Runner, error) {
	iterations := options.Scenario.Iterations
	duration := options.Scenario.Duration
	if iterations == 0 && duration == 0 {
		return nil, errors.New("invalid scenario: either iterations or duration is required")
	}
	if iterations > 0 && duration > 0 {
		return nil, errors.New("invalid scenario: iterations and duration are mutually exclusive")
	}

	executeHistogram, err := Meter.SyncInt64().Histogram(
		"execute.histogram",
		instrument.WithUnit("microseconds"),
		instrument.WithDescription("Excute method histogram"),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create a histogram: %w", err)
	}
	scenarioAttribute := attribute.KeyValue{Key: "scenario", Value: attribute.StringValue(options.Scenario.Name)}

	return &Runner{
		options: options,
		errors:  make(chan error),
		logger:  logger,
		metrics: &RunnerMetrics{
			executeHistogram:  executeHistogram,
			scenarioAttribute: scenarioAttribute,
		},
	}, nil
}

func calcConcurrency(iterations int, scenario *scenario.Scenario) int {
	concurrency := scenario.Concurrency
	if concurrency == 0 {
		concurrency = DEFAULT_CONCURRENCY
	}
	if iterations > 0 && concurrency > iterations {
		// Don't spin up more coroutines than the number of total iterations
		concurrency = iterations
	}
	return concurrency
}

// Run a scenario.
// Spins up coroutines according to the scenario configuration.
// Each coroutine runs the scenario Execute method in a loop until the scenario duration or max iterations is reached.
func (r *Runner) Run(ctx context.Context) error {
	c, err := shared.Connect(r.options.ClientOptions, r.logger)
	if err != nil {
		return err
	}
	defer c.Close()

	iterations := r.options.Scenario.Iterations
	duration := r.options.Scenario.Duration
	concurrency := calcConcurrency(iterations, r.options.Scenario)

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
		go r.runOne(ctx, logger, c)
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
func (r *Runner) runOne(ctx context.Context, logger *zap.SugaredLogger, c client.Client) {
	iterations := r.options.Scenario.Iterations
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
		run := scenario.Run{
			Client:          c,
			Scenario:        r.options.Scenario,
			IterationInTest: iteration,
			Logger:          logger.With("iteration", iteration),
			ID:              r.options.RunID,
		}

		startTime := time.Now()
		if err := r.options.Scenario.Execute(ctx, &run); err != nil {
			duration := time.Since(startTime)
			r.metrics.executeHistogram.Record(ctx, duration.Microseconds(), r.metrics.scenarioAttribute)
			err = fmt.Errorf("iteration %d failed: %w", iteration, err)
			logger.Error(err)
			r.errors <- err
			// Even though context will be cancelled by the runner, we break here to avoid needlessly running another iteration
			break
		}
	}
}

type CleanupOptions struct {
	PollInterval time.Duration
}

func getIdentity() string {
	username := "anonymous"
	user, err := user.Current()
	if err == nil {
		username = user.Name
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s@%s", username, hostname)
}

// Cleanup cleans up all workflows associated with the task.
// Requires ElasticSearch.
// TODO(bergundy): This fails on Cloud, not sure why
func (r *Runner) Cleanup(ctx context.Context, options CleanupOptions) error {
	c, err := shared.Connect(r.options.ClientOptions, r.logger)
	if err != nil {
		return err
	}
	defer c.Close()
	taskQueue := r.options.Scenario.TaskQueueForRunID(r.options.RunID)
	jobId := taskQueue
	// Clean based on task queue to avoid relying on search attributes and reducing the requirements of this framework.
	// Not escaping the value here and living with the consequences.
	query := fmt.Sprintf("TaskQueue = '%s'", taskQueue)

	_, err = c.WorkflowService().StartBatchOperation(ctx, &workflowservice.StartBatchOperationRequest{
		Namespace:       r.options.ClientOptions.Namespace,
		JobId:           jobId,
		Reason:          "omes cleanup",
		VisibilityQuery: query,
		Operation: &workflowservice.StartBatchOperationRequest_DeletionOperation{
			DeletionOperation: &batch.BatchOperationDeletion{Identity: getIdentity()},
		},
	})
	if err != nil {
		return err
	}
	// Loop and wait for the batch to complete
	for {
		response, err := c.WorkflowService().DescribeBatchOperation(ctx, &workflowservice.DescribeBatchOperationRequest{
			Namespace: r.options.ClientOptions.Namespace,
			JobId:     jobId,
		})
		if err != nil {
			return err
		}
		switch response.State {
		case enums.BATCH_OPERATION_STATE_FAILED:
			return fmt.Errorf("cleanup batch failed: %s", response.Reason)
		case enums.BATCH_OPERATION_STATE_COMPLETED:
			return nil
		case enums.BATCH_OPERATION_STATE_RUNNING:
			select {
			case <-time.After(options.PollInterval):
				// go to next loop iteration
			case <-ctx.Done():
				return ctx.Err()
			}
		case enums.BATCH_OPERATION_STATE_UNSPECIFIED:
			return fmt.Errorf("invalid batch state: %s - reason: %s", response.State, response.Reason)
		default:
			return fmt.Errorf("unexepcted batch state: %s - reason: %s", response.State, response.Reason)
		}
	}
}

type PrepareWorkerOptions struct {
	Language string
	Output   string
}

func (r *Runner) prepareWorker(ctx context.Context, options PrepareWorkerOptions) error {
	switch options.Language {
	case "go":
		args := []string{
			"go",
			"build",
			"-o", options.Output,
			// TODO: use relative path
			filepath.Join("workers", "go", "worker.go"),
		}
		r.logger.Infof("Building go worker with %v", args)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		return cmd.Run()
	default:
		return fmt.Errorf("language not supported: '%s'", options.Language)
	}
}

// TODO: worker tuning options
type WorkerOptions struct {
	Language    string
	TLSCertPath string
	TLSKeyPath  string
	// Time to wait before killing the worker process after sending SIGTERM in case it doesn't gracefully shut down.
	// Default is 30 seconds.
	GracefulShutdownDuration time.Duration
	//
	RetainBuildDir bool
}

// RunWorker prepares (e.g. builds) and run a worker for a given language.
// The worker process will be killed with SIGTERM when the given context is cancelled.
// If the worker process does not exit after options.GracefulShutdownDuration, it will get a SIGKILL
func (r *Runner) RunWorker(ctx context.Context, options WorkerOptions) error {
	var args []string
	tmpDir, err := os.MkdirTemp(os.TempDir(), "omes-build-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	r.logger.Infof("Created worker build dir %s", tmpDir)
	if options.RetainBuildDir {
		defer os.RemoveAll(tmpDir)
	}
	gracefulShutdownDuration := options.GracefulShutdownDuration
	if gracefulShutdownDuration == 0 {
		gracefulShutdownDuration = 30 * time.Second
	}

	switch options.Language {
	case "go":
		outputPath := filepath.Join(tmpDir, "worker")
		if err := r.prepareWorker(ctx, PrepareWorkerOptions{Language: options.Language, Output: outputPath}); err != nil {
			return err
		}
		args = []string{
			outputPath,
			"--server-address", r.options.ClientOptions.HostPort,
			"--namespace", r.options.ClientOptions.Namespace,
			"--task-queue", r.options.Scenario.TaskQueueForRunID(r.options.RunID),
		}
		if options.TLSCertPath != "" && options.TLSKeyPath != "" {
			args = append(args, "--tls-cert-path", options.TLSCertPath, "--tls-key-path", options.TLSKeyPath)
		}
	default:
		return fmt.Errorf("language not supported: '%s'", options.Language)
	}

	runErrorChan := make(chan error, 1)
	// Inentionally not using CommandContext since we want to kill the worker gracefully (using SIGTERM).
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	r.logger.Infof("Starting worker with args: %v", args)
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		runErrorChan <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled before worker shutdown
		r.logger.Infof("Sending SIGTERM to worker, PID: %d", cmd.Process.Pid)
		err := cmd.Process.Signal(syscall.SIGTERM)
		if err != nil {
			r.logger.Fatalf("failed to kill worker: %w", err)
		}
		select {
		case err := <-runErrorChan:
			return err
		case <-time.After(gracefulShutdownDuration):
			if err := cmd.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill worker: %w", err)
			}
			return <-runErrorChan
		}
	case err := <-runErrorChan:
		if err == nil {
			r.logger.Info("Worker gracefully stopped")
		}
		// Worker shutdown before context cancelled
		return err
	}
}

type AllInOneOptions struct {
	WorkerOptions WorkerOptions
}

// AllInOne run an all-in-one scenario (StartWorker + Run)
func (r *Runner) AllInOne(ctx context.Context, options AllInOneOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	workerErrChan := make(chan error, 1)
	runnerErrChan := make(chan error, 1)

	go func() {
		workerErrChan <- r.RunWorker(ctx, options.WorkerOptions)
	}()
	go func() {
		runnerErrChan <- r.Run(ctx)
	}()

	var workerErr error
	var runErr error

	select {
	case workerErr = <-workerErrChan:
		return fmt.Errorf("Worker exited prematurely: %w", workerErr)
	case runErr = <-runnerErrChan:
		if runErr != nil {
			r.logger.Errorf("Run completed with: %e", runErr)
		}
		cancel()
		workerErr = <-workerErrChan
	}

	if workerErr != nil {
		if runErr != nil {
			return fmt.Errorf("both run and worker completed with errors, worker error: %v, run error: %w", workerErr, runErr)
		}
		return fmt.Errorf("worker finished with error: %w", workerErr)
	}
	return runErr
}
