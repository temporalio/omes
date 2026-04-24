package worker

import (
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers/go/ebbandflow"
	"github.com/temporalio/omes/workers/go/kitchensink"
	"github.com/temporalio/omes/workers/go/schedulerstress"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type App struct {
	logger                    *zap.SugaredLogger
	taskQueue                 string
	taskQueueIndexSuffixStart int
	taskQueueIndexSuffixEnd   int

	loggingOptions clioptions.LoggingOptions
	clientOptions  clioptions.ClientOptions
	metricsOptions clioptions.MetricsOptions
	workerOptions  clioptions.WorkerOptions
}

func (a *App) Run(cmd *cobra.Command, args []string) {
	if a.taskQueueIndexSuffixStart > a.taskQueueIndexSuffixEnd {
		a.logger.Fatal("Task queue suffix start after end")
	}
	a.logger = a.loggingOptions.MustCreateLogger()

	metrics := a.metricsOptions.MustCreateMetrics(cmd.Context(), a.logger)
	client := a.clientOptions.MustDial(metrics, a.logger)

	// If there is an end, we run multiple
	var taskQueues []string
	if a.taskQueueIndexSuffixEnd == 0 {
		taskQueues = []string{a.taskQueue}
	} else {
		for i := a.taskQueueIndexSuffixStart; i <= a.taskQueueIndexSuffixEnd; i++ {
			taskQueues = append(taskQueues, fmt.Sprintf("%v-%v", a.taskQueue, i))
		}
	}

	if err := runWorkers(client, taskQueues, a.workerOptions); err != nil {
		a.logger.Fatalf("Fatal worker error: %v", err)
	}
	if err := metrics.Shutdown(cmd.Context(), a.logger, "", "", ""); err != nil {
		a.logger.Fatalf("Failed to shutdown metrics: %v", err)
	}
}

func makePollerBehavior(simple, auto int) worker.PollerBehavior {
	if auto > 0 {
		return worker.NewPollerBehaviorAutoscaling(worker.PollerBehaviorAutoscalingOptions{
			// TODO: remove InitialNumberOfPollers after https://github.com/temporalio/sdk-go/pull/2105
			InitialNumberOfPollers: auto,
			MaximumNumberOfPollers: auto,
		})
	}
	return worker.NewPollerBehaviorSimpleMaximum(worker.PollerBehaviorSimpleMaximumOptions{
		MaximumNumberOfPollers: simple,
	})
}

func parseVersioningBehavior(s string) (workflow.VersioningBehavior, error) {
	switch s {
	case "", "auto-upgrade":
		return workflow.VersioningBehaviorAutoUpgrade, nil
	case "pinned":
		return workflow.VersioningBehaviorPinned, nil
	default:
		return 0, fmt.Errorf("invalid --default-versioning-behavior %q (expected pinned or auto-upgrade)", s)
	}
}

func runWorkers(client client.Client, taskQueues []string, options clioptions.WorkerOptions) error {
	workerOpts := worker.Options{
		MaxConcurrentActivityExecutionSize:     options.MaxConcurrentActivities,
		MaxConcurrentWorkflowTaskExecutionSize: options.MaxConcurrentWorkflowTasks,
		ActivityTaskPollerBehavior: makePollerBehavior(
			options.MaxConcurrentActivityPollers,
			options.ActivityPollerAutoscaleMax,
		),
		WorkflowTaskPollerBehavior: makePollerBehavior(
			options.MaxConcurrentWorkflowPollers,
			options.WorkflowPollerAutoscaleMax,
		),
		WorkerActivitiesPerSecond: options.WorkerActivitiesPerSecond,
	}
	if options.DeploymentName != "" {
		if options.BuildID != "" {
			return fmt.Errorf("--build-id and --deployment-name are mutually exclusive; use --deployment-build-id with --deployment-name")
		}
		if options.DeploymentBuildID == "" {
			return fmt.Errorf("--deployment-build-id is required when --deployment-name is set")
		}
		behavior, err := parseVersioningBehavior(options.DefaultVersioningBehavior)
		if err != nil {
			return err
		}
		workerOpts.DeploymentOptions = worker.DeploymentOptions{
			UseVersioning: true,
			Version: worker.WorkerDeploymentVersion{
				DeploymentName: options.DeploymentName,
				BuildID:        options.DeploymentBuildID,
			},
			DefaultVersioningBehavior: behavior,
		}
	} else if options.DeploymentBuildID != "" {
		return fmt.Errorf("--deployment-build-id requires --deployment-name")
	} else if options.BuildID != "" {
		// DEPRECATED: BuildID and UseBuildIDForVersioning select the legacy
		// Rules-Based Versioning APIs, which Temporal Server will soon stop
		// supporting. New users should use --deployment-name and
		// --deployment-build-id above, which drive Worker Deployment Versioning.
		workerOpts.BuildID = options.BuildID
		workerOpts.UseBuildIDForVersioning = true
	}

	errCh := make(chan error, len(taskQueues))
	ebbFlowActivities := ebbandflow.Activities{}
	clientActivities := kitchensink.ClientActivities{
		Client: client,
	}
	service := nexus.NewService(kitchensink.KitchenSinkServiceName)
	for _, op := range []nexus.RegisterableOperation{kitchensink.EchoSyncOperation, kitchensink.EchoAsyncOperation} {
		if err := service.Register(op); err != nil {
			panic(fmt.Sprintf("failed to register operation: %v", err))
		}
	}

	for _, taskQueue := range taskQueues {
		taskQueue := taskQueue
		go func() {
			w := worker.New(client, taskQueue, workerOpts)
			w.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
			w.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
			w.RegisterActivityWithOptions(kitchensink.Delay, activity.RegisterOptions{Name: "delay"})
			w.RegisterActivityWithOptions(kitchensink.Payload, activity.RegisterOptions{Name: "payload"})
			w.RegisterActivityWithOptions(kitchensink.RetryableError, activity.RegisterOptions{Name: "retryable_error"})
			w.RegisterActivityWithOptions(kitchensink.Timeout, activity.RegisterOptions{Name: "timeout"})
			w.RegisterActivityWithOptions(kitchensink.Heartbeat, activity.RegisterOptions{Name: "heartbeat"})
			w.RegisterActivityWithOptions(clientActivities.ExecuteClientActivity, activity.RegisterOptions{Name: "client"})
			w.RegisterWorkflow(kitchensink.NexusHandlerWorkflow)
			w.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowTrackWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowTrack"})
			w.RegisterActivity(&ebbFlowActivities)
			w.RegisterWorkflowWithOptions(schedulerstress.NoopScheduledWorkflow, workflow.RegisterOptions{Name: "NoopScheduledWorkflow"})
			w.RegisterWorkflowWithOptions(schedulerstress.SleepScheduledWorkflow, workflow.RegisterOptions{Name: "SleepScheduledWorkflow"})
			w.RegisterNexusService(service)
			errCh <- w.Run(worker.InterruptCh())
		}()
	}
	for range taskQueues {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func Main() {
	var app App

	cmd := &cobra.Command{
		Use:   "worker",
		Short: "A generic worker for running omes scenarios",
		Run:   app.Run,
	}

	cmd.Flags().AddFlagSet(app.loggingOptions.FlagSet())
	cmd.Flags().AddFlagSet(app.clientOptions.FlagSet())
	cmd.Flags().AddFlagSet(app.metricsOptions.FlagSet(""))
	cmd.Flags().AddFlagSet(app.workerOptions.FlagSet())
	cmd.Flags().StringVarP(&app.taskQueue, "task-queue", "q", "omes", "Task queue to use")
	cmd.Flags().IntVar(&app.taskQueueIndexSuffixStart,
		"task-queue-suffix-index-start", 0, "Inclusive start for task queue suffix range")
	cmd.Flags().IntVar(&app.taskQueueIndexSuffixEnd,
		"task-queue-suffix-index-end", 0, "Inclusive end for task queue suffix range")

	defer func() {
		if app.logger != nil {
			app.logger.Sync()
		}
	}()

	if err := cmd.Execute(); err != nil {
		if app.logger != nil {
			app.logger.Fatal(err)
		} else {
			clioptions.BackupLogger.Fatal(err)
		}
	}
}
