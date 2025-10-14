package worker

import (
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/workers/go/ebbandflow"
	"github.com/temporalio/omes/workers/go/kitchensink"
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
	metrics := a.metricsOptions.MustCreateMetrics(a.logger)
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
	if err := metrics.Shutdown(cmd.Context()); err != nil {
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

func runWorkers(client client.Client, taskQueues []string, options clioptions.WorkerOptions) error {
	errCh := make(chan error, len(taskQueues))
	ebbFlowActivities := ebbandflow.Activities{}
	clientActivities := kitchensink.ClientActivities{
		Client: client,
	}
	service := nexus.NewService(kitchensink.KitchenSinkServiceName)
	err := service.Register(kitchensink.EchoSyncOperation, kitchensink.EchoAsyncOperation, kitchensink.WaitForCancelOperation)
	if err != nil {
		panic(fmt.Sprintf("failed to register operations: %v", err))
	}

	for _, taskQueue := range taskQueues {
		taskQueue := taskQueue
		go func() {
			w := worker.New(client, taskQueue, worker.Options{
				BuildID:                                options.BuildID,
				UseBuildIDForVersioning:                options.BuildID != "",
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
			})
			w.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
			w.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
			w.RegisterActivityWithOptions(kitchensink.Delay, activity.RegisterOptions{Name: "delay"})
			w.RegisterActivityWithOptions(kitchensink.Payload, activity.RegisterOptions{Name: "payload"})
			w.RegisterActivityWithOptions(kitchensink.RetryableError, activity.RegisterOptions{Name: "retryable_error"})
			w.RegisterActivityWithOptions(kitchensink.Timeout, activity.RegisterOptions{Name: "timeout"})
			w.RegisterActivityWithOptions(kitchensink.Heartbeat, activity.RegisterOptions{Name: "heartbeat"})
			w.RegisterActivityWithOptions(clientActivities.ExecuteClientActivity, activity.RegisterOptions{Name: "client"})
			w.RegisterWorkflow(kitchensink.EchoWorkflow)
			w.RegisterWorkflow(kitchensink.WaitForCancelWorkflow)
			w.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowTrackWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowTrack"})
			w.RegisterActivity(&ebbFlowActivities)
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

	var cmd = &cobra.Command{
		Use:   "worker",
		Short: "A generic worker for running omes scenarios",
		Run:   app.Run,
	}

	cmd.Flags().AddFlagSet(app.loggingOptions.FlagSet())
	cmd.Flags().AddFlagSet(app.clientOptions.FlagSet())
	cmd.Flags().AddFlagSet(app.metricsOptions.FlagSet(""))
	cmd.Flags().AddFlagSet(app.workerOptions.FlagSet(""))
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
