package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"
	"time"

	"github.com/temporalio/omes/components"
	clientComponent "github.com/temporalio/omes/components/client"
	"github.com/temporalio/omes/components/logging"
	"github.com/temporalio/omes/components/metrics"
	"github.com/temporalio/omes/scenario"
	"go.temporal.io/api/batch/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const DEFAULT_CONCURRENCY = 10

// Options for creating a Runner.
type Options struct {
	ClientOptions clientComponent.Options
	// ID used for prefixing workflow IDs and determining the task queue.
	RunID        string
	ScenarioName string
	Scenario     *scenario.Scenario
}

type Runner struct {
	options Options
	metrics *metrics.Metrics
	logger  *zap.SugaredLogger
}

// NewRunner instantiates a Runner
func NewRunner(options Options, metrics *metrics.Metrics, logger *zap.SugaredLogger) *Runner {
	return &Runner{
		options: options,
		logger:  logger,
		metrics: metrics,
	}
}

// Run a scenario.
// The actual run logic is delegated to the scenario Executor.
func (r *Runner) Run(ctx context.Context, client client.Client) error {
	return r.options.Scenario.Executor.Run(ctx, &scenario.RunOptions{
		ScenarioName: r.options.ScenarioName,
		RunID:        r.options.RunID,
		Logger:       r.logger,
		Metrics:      r.metrics,
		Client:       client,
	})
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

func normalizeLangName(lang string) (string, error) {
	switch lang {
	case "go", "java", "ts", "py":
		// Allow the full typescript or python word, but we need to match the file
		// extension for the rest of run
	case "typescript":
		lang = "ts"
	case "python":
		lang = "py"
	default:
		return "", fmt.Errorf("invalid language %q, must be one of: go or java or ts or py", lang)
	}
	return lang, nil
}

// Cleanup cleans up all workflows associated with the task.
// Requires ElasticSearch.
// TODO(bergundy): This fails on Cloud, not sure why.
// TODO(bergundy): Add support for cleaning the entire namespace or by search attribute if we decide to add multi-queue scenarios.
func (r *Runner) Cleanup(ctx context.Context, client client.Client, options CleanupOptions) error {
	taskQueue := scenario.TaskQueueForRun(r.options.ScenarioName, r.options.RunID)
	jobId := taskQueue
	// Clean based on task queue to avoid relying on search attributes and reducing the requirements of this framework.
	query := fmt.Sprintf("TaskQueue = %q", taskQueue)

	_, err := client.WorkflowService().StartBatchOperation(ctx, &workflowservice.StartBatchOperationRequest{
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
		response, err := client.WorkflowService().DescribeBatchOperation(ctx, &workflowservice.DescribeBatchOperationRequest{
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
			filepath.Join("workers", "go", "main.go"),
		}
		r.logger.Infof("Building go worker with %v", args)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		return cmd.Run()
	case "py":
		os.Mkdir(options.Output, 0755)
		pwd, _ := os.Getwd()
		pyProjectTOML := `
[tool.poetry]
name = "omes-python-load-test-worker"
version = "0.1.0"
description = "Temporal Omes load testing framework worker"
authors = ["Temporal Technologies Inc <sdk@temporal.io>"]

[tool.poetry.dependencies]
python = "^3.10"
omes = { path = "` + pwd + `" }

[build-system]
requires = ["poetry-core>=1.0.0"]
build-backend = "poetry.core.masonry.api"`
		if err := os.WriteFile(filepath.Join(options.Output, "pyproject.toml"), []byte(pyProjectTOML), 0644); err != nil {
			return fmt.Errorf("failed writing pyproject.toml: %w", err)
		}
		args := []string{
			"poetry",
			"install",
			"--no-root",
			"--no-dev",
			"-v",
		}
		r.logger.Infof("Building python worker with %v %v", args, options.Output)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = options.Output
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("language not supported: '%s'", options.Language)
	}
}

type WorkerOptions struct {
	MetricsOptions metrics.Options
	LoggingOptions logging.Options
	ClientOptions  clientComponent.Options
	// Worker SDK language
	Language string
	// Time to wait before killing the worker process after sending SIGTERM in case it doesn't gracefully shut down.
	// Default is 30 seconds.
	GracefulShutdownDuration time.Duration
	RetainBuildDir           bool
	// TODO: worker tuning options
}

// RunWorker prepares (e.g. builds) and run a worker for a given language.
// The worker process will be killed with SIGTERM when the given context is cancelled.
// If the worker process does not exit after options.GracefulShutdownDuration, it will get a SIGKILL.
func (r *Runner) RunWorker(ctx context.Context, options WorkerOptions) error {
	var args []string
	tmpDir, err := os.MkdirTemp(os.TempDir(), "omes-build-")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	r.logger.Infof("Created worker build dir %s", tmpDir)
	if !options.RetainBuildDir {
		defer os.RemoveAll(tmpDir)
	}
	gracefulShutdownDuration := options.GracefulShutdownDuration
	if gracefulShutdownDuration == 0 {
		gracefulShutdownDuration = 30 * time.Second
	}
	language, err := normalizeLangName(options.Language)
	if err != nil {
		return fmt.Errorf("could not parse this language: %w", err)
	}

	switch language {
	case "go":
		outputPath := filepath.Join(tmpDir, "worker")
		if err := r.prepareWorker(ctx, PrepareWorkerOptions{Language: language, Output: outputPath}); err != nil {
			return err
		}
		args = []string{
			outputPath,
			"--task-queue", scenario.TaskQueueForRun(r.options.ScenarioName, r.options.RunID),
		}
		args = append(args, components.OptionsToFlags(&options.ClientOptions)...)
		args = append(args, components.OptionsToFlags(&options.MetricsOptions)...)
		args = append(args, components.OptionsToFlags(&options.LoggingOptions)...)
	case "py":
		outputPath := filepath.Join(tmpDir, "worker")
		if err := r.prepareWorker(ctx, PrepareWorkerOptions{Language: language, Output: outputPath}); err != nil {
			return err
		}
		args = []string{
			"poetry",
			"run",
			"python",
			"-m",
			"workers.python.main",
			"--task-queue", scenario.TaskQueueForRun(r.options.ScenarioName, r.options.RunID),
		}
		args = append(args, components.OptionsToFlags(&options.ClientOptions)...)
		args = append(args, components.OptionsToFlags(&options.MetricsOptions)...)
		args = append(args, components.OptionsToFlags(&options.LoggingOptions)...)
	default:
		return fmt.Errorf("language not supported: '%s'", options.Language)
	}

	runErrorChan := make(chan error, 1)
	// Inentionally not using CommandContext since we want to kill the worker gracefully (using SIGTERM).
	cmd := exec.Command(args[0], args[1:]...)
	outputPath := filepath.Join(tmpDir, "worker")
	cmd.Dir = outputPath
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

// RunAllInOne runs an all-in-one scenario (RunWorker + Run).
func (r *Runner) RunAllInOne(ctx context.Context, client client.Client, options AllInOneOptions) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	workerErrChan := make(chan error, 1)
	runnerErrChan := make(chan error, 1)

	go func() {
		workerErrChan <- r.RunWorker(ctx, options.WorkerOptions)
	}()
	go func() {
		runnerErrChan <- r.Run(ctx, client)
	}()

	var workerErr error
	var runErr error

	select {
	case workerErr = <-workerErrChan:
		return fmt.Errorf("worker exited prematurely: %w", workerErr)
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
