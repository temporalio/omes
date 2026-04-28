package harness

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
	sdkclient "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type WorkerContext struct {
	Logger             *zap.SugaredLogger
	TaskQueue          string
	ErrOnUnimplemented bool
	WorkerOptions      sdkworker.Options
}

type WorkerFactory func(sdkclient.Client, WorkerContext) sdkworker.Worker

type workerCLIOptions struct {
	flags                     *pflag.FlagSet
	loggingOptions            clioptions.LoggingOptions
	clientOptions             clioptions.ClientOptions
	metricsOptions            clioptions.MetricsOptions
	workerOptions             clioptions.WorkerOptions
	taskQueue                 string
	taskQueueSuffixIndexStart int
	taskQueueSuffixIndexEnd   int
}

func runWorkerCLI(workerFactory WorkerFactory, clientFactory ClientFactory, argv []string) error {
	if workerFactory == nil {
		return fmt.Errorf("worker factory is required")
	}
	if clientFactory == nil {
		return fmt.Errorf("client factory is required")
	}
	options := newWorkerCLIOptions()
	if err := options.parse(argv); err != nil {
		return err
	}
	if options.taskQueueSuffixIndexStart > options.taskQueueSuffixIndexEnd {
		return fmt.Errorf("task queue suffix start after end")
	}
	workerOptions, err := buildWorkerOptions(options.flags, options.workerOptions)
	if err != nil {
		return err
	}
	logger, err := configureLogger(options.loggingOptions.LogLevel, options.loggingOptions.LogEncoding)
	if err != nil {
		return err
	}
	defer logger.Sync()

	config, err := buildClientConfig(clientConfigOptions{
		Logger:                  logger,
		ServerAddress:           options.clientOptions.Address,
		Namespace:               options.clientOptions.Namespace,
		AuthHeader:              options.clientOptions.AuthHeader,
		EnableTLS:               options.clientOptions.EnableTLS,
		TLSCertPath:             options.clientOptions.ClientCertPath,
		TLSKeyPath:              options.clientOptions.ClientKeyPath,
		TLSServerName:           options.clientOptions.TLSServerName,
		DisableHostVerification: options.clientOptions.DisableHostVerification,
		PromListenAddress:       options.metricsOptions.PrometheusListenAddress,
		PromHandlerPath:         options.metricsOptions.PrometheusHandlerPath,
	})
	if err != nil {
		return err
	}
	if config.Metrics != nil {
		defer config.Metrics.Shutdown(context.Background(), logger, "", "", "")
	}
	client, err := clientFactory(config)
	if err != nil {
		return err
	}
	if client != nil {
		defer client.Close()
	}

	taskQueues := buildTaskQueues(logger, options.taskQueue, options.taskQueueSuffixIndexStart, options.taskQueueSuffixIndexEnd)
	workers := make([]sdkworker.Worker, 0, len(taskQueues))
	for _, taskQueue := range taskQueues {
		context := WorkerContext{
			Logger:             logger,
			TaskQueue:          taskQueue,
			ErrOnUnimplemented: options.workerOptions.ErrOnUnimplemented,
			WorkerOptions:      workerOptions,
		}
		workers = append(workers, workerFactory(client, context))
	}

	runContext, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return runWorkers(workers, runContext.Done())
}

func buildTaskQueues(logger *zap.SugaredLogger, taskQueue string, suffixStart, suffixEnd int) []string {
	if suffixEnd == 0 {
		logger.Infof("Go worker will run on task queue %s", taskQueue)
		return []string{taskQueue}
	}
	taskQueues := make([]string, 0, suffixEnd-suffixStart+1)
	for i := suffixStart; i <= suffixEnd; i++ {
		taskQueues = append(taskQueues, fmt.Sprintf("%s-%d", taskQueue, i))
	}
	logger.Infof("Go worker will run on %d task queue(s)", len(taskQueues))
	return taskQueues
}

func parseVersioningBehavior(value string) (workflow.VersioningBehavior, error) {
	switch value {
	case "", "auto-upgrade":
		return workflow.VersioningBehaviorAutoUpgrade, nil
	case "pinned":
		return workflow.VersioningBehaviorPinned, nil
	default:
		return 0, fmt.Errorf("invalid --default-versioning-behavior %q (expected pinned or auto-upgrade)", value)
	}
}

func buildWorkerOptions(flags *pflag.FlagSet, args clioptions.WorkerOptions) (sdkworker.Options, error) {
	options := sdkworker.Options{}
	if args.DeploymentName != "" {
		if args.BuildID != "" {
			return sdkworker.Options{}, fmt.Errorf("--build-id and --deployment-name are mutually exclusive; use --deployment-build-id with --deployment-name")
		}
		if args.DeploymentBuildID == "" {
			return sdkworker.Options{}, fmt.Errorf("--deployment-build-id is required when --deployment-name is set")
		}
		behavior, err := parseVersioningBehavior(args.DefaultVersioningBehavior)
		if err != nil {
			return sdkworker.Options{}, err
		}
		options.DeploymentOptions = sdkworker.DeploymentOptions{
			UseVersioning: true,
			Version: sdkworker.WorkerDeploymentVersion{
				DeploymentName: args.DeploymentName,
				BuildID:        args.DeploymentBuildID,
			},
			DefaultVersioningBehavior: behavior,
		}
	} else if args.DeploymentBuildID != "" {
		return sdkworker.Options{}, fmt.Errorf("--deployment-build-id requires --deployment-name")
	} else if args.BuildID != "" {
		options.BuildID = args.BuildID
		options.UseBuildIDForVersioning = true
	}
	if flags.Changed("activity-poller-autoscale-max") {
		options.ActivityTaskPollerBehavior = sdkworker.NewPollerBehaviorAutoscaling(sdkworker.PollerBehaviorAutoscalingOptions{
			InitialNumberOfPollers: args.ActivityPollerAutoscaleMax,
			MaximumNumberOfPollers: args.ActivityPollerAutoscaleMax,
		})
	} else if flags.Changed("max-concurrent-activity-pollers") {
		options.ActivityTaskPollerBehavior = sdkworker.NewPollerBehaviorSimpleMaximum(sdkworker.PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: args.MaxConcurrentActivityPollers,
		})
	}
	if flags.Changed("workflow-poller-autoscale-max") {
		options.WorkflowTaskPollerBehavior = sdkworker.NewPollerBehaviorAutoscaling(sdkworker.PollerBehaviorAutoscalingOptions{
			InitialNumberOfPollers: args.WorkflowPollerAutoscaleMax,
			MaximumNumberOfPollers: args.WorkflowPollerAutoscaleMax,
		})
	} else if flags.Changed("max-concurrent-workflow-pollers") {
		options.WorkflowTaskPollerBehavior = sdkworker.NewPollerBehaviorSimpleMaximum(sdkworker.PollerBehaviorSimpleMaximumOptions{
			MaximumNumberOfPollers: args.MaxConcurrentWorkflowPollers,
		})
	}
	if flags.Changed("max-concurrent-activities") {
		options.MaxConcurrentActivityExecutionSize = args.MaxConcurrentActivities
	}
	if flags.Changed("max-concurrent-workflow-tasks") {
		options.MaxConcurrentWorkflowTaskExecutionSize = args.MaxConcurrentWorkflowTasks
	}
	if flags.Changed("activities-per-second") {
		options.WorkerActivitiesPerSecond = args.WorkerActivitiesPerSecond
	}
	return options, nil
}

func runWorkers(workers []sdkworker.Worker, stopCh <-chan struct{}) error {
	if len(workers) == 0 {
		return nil
	}

	workerStopCh := make(chan interface{})
	var stopOnce sync.Once
	stopWorkers := func() {
		stopOnce.Do(func() {
			close(workerStopCh)
		})
	}

	results := make(chan error, len(workers))
	for _, worker := range workers {
		go func(worker sdkworker.Worker) {
			results <- worker.Run(workerStopCh)
		}(worker)
	}

	var firstErr error
	remaining := len(workers)
	for remaining > 0 {
		select {
		case err := <-results:
			remaining--
			if err != nil && firstErr == nil {
				firstErr = err
				stopWorkers()
			}
		case <-stopCh:
			stopWorkers()
			stopCh = nil
		}
	}
	return firstErr
}

func newWorkerCLIOptions() *workerCLIOptions {
	options := &workerCLIOptions{taskQueue: "omes"}
	flags := pflag.NewFlagSet("worker", pflag.ContinueOnError)
	flags.StringVarP(&options.taskQueue, "task-queue", "q", options.taskQueue, "Task queue to use")
	flags.IntVar(&options.taskQueueSuffixIndexStart, "task-queue-suffix-index-start", 0, "Inclusive start for task queue suffix range")
	flags.IntVar(&options.taskQueueSuffixIndexEnd, "task-queue-suffix-index-end", 0, "Inclusive end for task queue suffix range")
	flags.AddFlagSet(options.loggingOptions.FlagSet())
	flags.AddFlagSet(options.clientOptions.FlagSet())
	flags.AddFlagSet(options.metricsOptions.FlagSet(""))
	flags.AddFlagSet(options.workerOptions.FlagSetWithPrefix(""))
	options.flags = flags
	return options
}

func (o *workerCLIOptions) parse(argv []string) error {
	return o.flags.Parse(argv)
}
