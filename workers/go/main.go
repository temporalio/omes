package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/components/client"
	"github.com/temporalio/omes/components/logging"
	"github.com/temporalio/omes/components/metrics"
	"github.com/temporalio/omes/workers/go/activities"
	"github.com/temporalio/omes/workers/go/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type App struct {
	logger         *zap.SugaredLogger
	taskQueue      string
	loggingOptions logging.Options
	clientOptions  client.Options
	metricsOptions metrics.Options
}

func (a *App) Run(cmd *cobra.Command, args []string) {
	a.logger = logging.MustSetup(&a.loggingOptions)
	metrics := metrics.MustSetup(&a.metricsOptions, a.logger)
	client := client.MustConnect(&a.clientOptions, metrics, a.logger)

	workerOpts := worker.Options{}
	w := worker.New(client, a.taskQueue, workerOpts)
	w.RegisterWorkflowWithOptions(workflows.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	w.RegisterActivityWithOptions(activities.NoopActivity, activity.RegisterOptions{Name: "noop"})

	stopChan := make(chan interface{})
	go func() {
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
		sig := <-sigs
		a.logger.Infof("Got signal %s, stopping...", sig)
		close(stopChan)
	}()
	if err := w.Run(stopChan); err != nil {
		a.logger.Fatalf("Fatal worker error: %v", err)
	}
	if err := metrics.Shutdown(cmd.Context()); err != nil {
		a.logger.Fatalf("Failed to shutdown metrics: %v", err)
	}
}

func main() {
	var app App

	var cmd = &cobra.Command{
		Use:   "worker",
		Short: "A generic worker for running omes scenarios",
		Run:   app.Run,
	}

	logging.AddCLIFlags(cmd.Flags(), &app.loggingOptions)
	client.AddCLIFlags(cmd.Flags(), &app.clientOptions)
	metrics.AddCLIFlags(cmd.Flags(), &app.metricsOptions)
	cmd.Flags().StringVarP(&app.taskQueue, "task-queue", "q", "omes", "task queue to use")

	defer func() {
		if app.logger != nil {
			app.logger.Sync()
		}
	}()

	if err := cmd.Execute(); err != nil {
		if app.logger != nil {
			app.logger.Fatal(err)
		} else {
			logging.BackupLogger.Fatal(err)
		}
	}
}
