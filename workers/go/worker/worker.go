package worker

import (
	"github.com/spf13/cobra"
	"github.com/temporalio/omes/cmd/cmdoptions"
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
	loggingOptions cmdoptions.LoggingOptions
	clientOptions  cmdoptions.ClientOptions
	metricsOptions cmdoptions.MetricsOptions
}

func (a *App) Run(cmd *cobra.Command, args []string) {
	a.logger = a.loggingOptions.MustCreateLogger()
	metrics := a.metricsOptions.MustCreateMetrics(a.logger)
	client := a.clientOptions.MustDial(metrics, a.logger)

	workerOpts := worker.Options{}
	w := worker.New(client, a.taskQueue, workerOpts)
	w.RegisterWorkflowWithOptions(workflows.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	w.RegisterActivityWithOptions(activities.NoopActivity, activity.RegisterOptions{Name: "noop"})

	if err := w.Run(worker.InterruptCh()); err != nil {
		a.logger.Fatalf("Fatal worker error: %v", err)
	}
	if err := metrics.Shutdown(cmd.Context()); err != nil {
		a.logger.Fatalf("Failed to shutdown metrics: %v", err)
	}
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
			cmdoptions.BackupLogger.Fatal(err)
		}
	}
}
