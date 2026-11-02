package worker

import (
	"fmt"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/omes/clioptions"
	"github.com/temporalio/omes/workers/go/harness"
	"github.com/temporalio/omes/workers/go/workerlib/ebbandflow"
	"github.com/temporalio/omes/workers/go/workerlib/kitchensink"
	"github.com/temporalio/omes/workers/go/workerlib/schedulerstress"
	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

var App, finishPrivateCapture = privateCaptureApp(harness.App{
	Worker:        buildWorker,
	ClientFactory: buildClient,
}, privateCaptureCase{
	Namespace:        "loadtest",
	TaskQueuePrefix:  "omes-cicd-go-vcr",
	WorkflowIDPrefix: "w-cicd-go-vcr-",
})

// Main is the default Go worker entry point.
func Main() {
	err := harness.Run(App)
	finishPrivateCapture()
	if err != nil {
		clioptions.BackupLogger.Fatal(err)
	}
}

func buildWorker(client sdkclient.Client, context harness.WorkerContext) sdkworker.Worker {
	ebbFlowActivities := ebbandflow.Activities{}
	clientActivities := kitchensink.ClientActivities{Client: client}
	service := nexus.NewService(kitchensink.KitchenSinkServiceName)
	if err := service.Register(kitchensink.KitchenSinkNexusOperation); err != nil {
		panic(err)
	}
	w := sdkworker.New(client, context.TaskQueue, context.WorkerOptions)
	w.RegisterWorkflowWithOptions(kitchensink.KitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	w.RegisterActivityWithOptions(kitchensink.Noop, activity.RegisterOptions{Name: "noop"})
	w.RegisterActivityWithOptions(kitchensink.Delay, activity.RegisterOptions{Name: "delay"})
	w.RegisterActivityWithOptions(kitchensink.Payload, activity.RegisterOptions{Name: "payload"})
	w.RegisterActivityWithOptions(kitchensink.RetryableError, activity.RegisterOptions{Name: "retryable_error"})
	w.RegisterActivityWithOptions(kitchensink.Timeout, activity.RegisterOptions{Name: "timeout"})
	w.RegisterActivityWithOptions(kitchensink.Heartbeat, activity.RegisterOptions{Name: "heartbeat"})
	w.RegisterActivityWithOptions(clientActivities.ExecuteClientActivity, activity.RegisterOptions{Name: "client"})
	w.RegisterWorkflowWithOptions(ebbandflow.EbbAndFlowTrackWorkflow, workflow.RegisterOptions{Name: "ebbAndFlowTrack"})
	w.RegisterActivity(&ebbFlowActivities)
	w.RegisterWorkflowWithOptions(schedulerstress.NoopScheduledWorkflow, workflow.RegisterOptions{Name: "NoopScheduledWorkflow"})
	w.RegisterWorkflowWithOptions(schedulerstress.SleepScheduledWorkflow, workflow.RegisterOptions{Name: "SleepScheduledWorkflow"})
	w.RegisterNexusService(service)
	return w
}

func buildClient(config harness.ClientConfig) (sdkclient.Client, error) {
	return buildClientWithCapture(config, nil)
}

func buildClientWithCapture(config harness.ClientConfig, capture *privateCapture) (sdkclient.Client, error) {
	options := harness.BuildSDKClientOptions(config)
	options.DataConverter = clioptions.OmesDataConverter()
	if capture != nil {
		options = capture.clientOptions(options, config.Logger)
	}
	client, err := sdkclient.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return client, nil
}
