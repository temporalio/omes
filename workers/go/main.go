package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/temporalio/omes/app"
	"github.com/temporalio/omes/logging"
	"github.com/temporalio/omes/workers/go/activities"
	"github.com/temporalio/omes/workers/go/workflows"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

type App struct {
	logger            *zap.SugaredLogger
	serverAddress     string
	taskQueue         string
	namespace         string
	tlsCertPath       string
	tlsKeyPath        string
	logLevel          string
	logEncoding       string
	prometheusOptions app.PrometheusOptions
}

func (a *App) Run(cmd *cobra.Command, args []string) {
	a.logger = logging.MustSetupLogging(a.logLevel, a.logEncoding)

	clientOpts := client.Options{
		HostPort:  a.serverAddress,
		Namespace: a.namespace,
	}
	metrics := app.MustInitMetrics(&a.prometheusOptions, a.logger)
	if err := app.LoadCertsIntoOptions(&clientOpts, a.tlsCertPath, a.tlsKeyPath); err != nil {
		a.logger.Fatalf("Failed to load TLS certs: %v", err)
	}
	client := app.MustConnect(clientOpts, metrics, a.logger)
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

	cmd.Flags().StringVar(&app.logLevel, "log-level", "info", "(debug info warn error dpanic panic fatal)")
	cmd.Flags().StringVar(&app.logEncoding, "log-encoding", "console", "(console json)")
	cmd.Flags().StringVarP(&app.serverAddress, "server-address", "a", "localhost:7233", "address of Temporal server")
	cmd.Flags().StringVarP(&app.namespace, "namespace", "n", "default", "namespace to connect to")
	cmd.Flags().StringVar(&app.tlsCertPath, "tls-cert-path", "", "Path to TLS certificate")
	cmd.Flags().StringVar(&app.tlsKeyPath, "tls-key-path", "", "Path to private key")
	cmd.Flags().StringVarP(&app.taskQueue, "task-queue", "q", "omes", "task queue to use")
	cmd.Flags().StringVar(&app.prometheusOptions.ListenAddress, "prom-listen-address", "", "prometheus listen address")
	cmd.Flags().StringVar(&app.prometheusOptions.HandlerPath, "prom-handler-path", "", "prometheus handler path")

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
