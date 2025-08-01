package worker

import (
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/cmdoptions"
	"github.com/temporalio/omes/workers/go/ebbandflow"
	"github.com/temporalio/omes/workers/go/kitchensink"
	"github.com/temporalio/omes/workers/go/throughputstress"
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

	loggingOptions cmdoptions.LoggingOptions
	clientOptions  cmdoptions.ClientOptions
	metricsOptions cmdoptions.MetricsOptions
	workerOptions  cmdoptions.WorkerOptions
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

func runWorkers(client client.Client, taskQueues []string, options cmdoptions.WorkerOptions) error {
	errCh := make(chan error, len(taskQueues))
	tpsActivities := throughputstress.Activities{
		Client: client,
	}
	ebbFlowActivities := ebbandflow.Activities{}
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
				MaxConcurrentActivityTaskPollers:       options.MaxConcurrentActivityPollers,
				MaxConcurrentWorkflowTaskPollers:       options.MaxConcurrentWorkflowPollers,
			})
			w.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
			w.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
			w.RegisterActivityWithOptions(kitchensink.Delay, activity.RegisterOptions{Name: "delay"})
			w.RegisterActivityWithOptions(kitchensink.Payload, activity.RegisterOptions{Name: "payload"})
			w.RegisterWorkflowWithOptions(throughputstress.ThroughputStressWorkflow, workflow.RegisterOptions{Name: "throughputStress"})
			w.RegisterWorkflow(throughputstress.ThroughputStressChild)
			w.RegisterWorkflow(kitchensink.EchoWorkflow)
			w.RegisterWorkflow(kitchensink.WaitForCancelWorkflow)
			w.RegisterActivity(&tpsActivities)
			w.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowTrackWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowTrack"})
			w.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowReportWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowReport"})
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

	app.loggingOptions.AddCLIFlags(cmd.Flags())
	app.clientOptions.AddCLIFlags(cmd.Flags())
	app.metricsOptions.AddCLIFlags(cmd.Flags(), "")
	app.workerOptions.AddCLIFlags(cmd.Flags(), "")
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
			cmdoptions.BackupLogger.Fatal(err)
		}
	}
}
