package worker

import (
	"fmt"

	"github.com/temporalio/omes/workers/go/kitchensink"
	"github.com/temporalio/omes/workers/go/throughputstress"
	"go.temporal.io/sdk/activity"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/cmdoptions"
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

	if err := runWorkers(client, taskQueues); err != nil {
		a.logger.Fatalf("Fatal worker error: %v", err)
	}
	if err := metrics.Shutdown(cmd.Context()); err != nil {
		a.logger.Fatalf("Failed to shutdown metrics: %v", err)
	}
}

func runWorkers(client client.Client, taskQueues []string) error {
	errCh := make(chan error, len(taskQueues))
	tpsActivities := throughputstress.Activities{
		Client: client,
	}
	for _, taskQueue := range taskQueues {
		taskQueue := taskQueue
		go func() {
			w := worker.New(client, taskQueue, worker.Options{})
			w.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
			w.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
			w.RegisterWorkflowWithOptions(throughputstress.ThroughputStressWorkflow, workflow.RegisterOptions{Name: "throughputStress"})
			w.RegisterWorkflow(throughputstress.ThroughputStressChild)
			w.RegisterWorkflow(throughputstress.ThroughputStressExecutorWorkflow)
			w.RegisterActivity(&tpsActivities)
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
