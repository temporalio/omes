// Package encryption is an omes project that installs an encrypting data
// converter and drives a workflow which touches every payload-bearing path the
// Go SDK exposes, so the resulting histories can be inspected path by path.
//
// See README.md for which paths end up encrypted and which do not.
package encryption

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/nexus-rpc/sdk-go/nexus"
	"github.com/temporalio/omes/workers/go/harness"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/activity"
	sdkclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	sdkworker "go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

const (
	workflowExecutionTimeout = 5 * time.Minute

	// Bounds the cleanup of a failed iteration, which runs on its own context so it
	// still happens when the iteration failed because the run's context was done.
	iterationCleanupTimeout = 30 * time.Second
)

// App is the entrypoint the Go worker registry dispatches to.
var App = harness.App{
	Worker:        buildWorker,
	ClientFactory: buildClient,
	Project: &harness.ProjectHandlers{
		Init:    initProject,
		Execute: executeIteration,
	},
}

// state carries the Nexus endpoint from Init to Execute.
var state struct {
	nexusEndpoint string
}

// projectConfig is the JSON handed to the app through
// --option project-config-file=<path>, which is how the project harness passes
// app-specific configuration.
type projectConfig struct {
	// NexusEndpoint names a Nexus endpoint that already exists in the namespace.
	// The app does not create one.
	//
	// The endpoint is a loopback — the caller is the coverage workflow and the
	// handler is the same worker — so its target is this namespace and the run's
	// task queue, omes-<run-id>. That target is fixed when the endpoint is created,
	// so pin --run-id and the same endpoint keeps working across runs.
	NexusEndpoint string `json:"nexusEndpoint"`
}

func buildWorker(client sdkclient.Client, workerContext harness.WorkerContext) sdkworker.Worker {
	worker := sdkworker.New(client, workerContext.TaskQueue, workerContext.WorkerOptions)

	worker.RegisterWorkflowWithOptions(CoverageWorkflow, workflow.RegisterOptions{Name: coverageWorkflowName})
	worker.RegisterWorkflowWithOptions(EchoChildWorkflow, workflow.RegisterOptions{Name: echoChildWorkflowName})
	worker.RegisterWorkflowWithOptions(FailingWorkflow, workflow.RegisterOptions{Name: failingWorkflowName})
	worker.RegisterWorkflowWithOptions(TargetWorkflow, workflow.RegisterOptions{Name: targetWorkflowName})
	worker.RegisterWorkflowWithOptions(RetryWorkflow, workflow.RegisterOptions{Name: retryWorkflowName})

	worker.RegisterActivityWithOptions(EchoActivity, activity.RegisterOptions{Name: echoActivityName})
	worker.RegisterActivityWithOptions(FailActivity, activity.RegisterOptions{Name: failActivityName})
	worker.RegisterActivityWithOptions(CancelActivity, activity.RegisterOptions{Name: cancelActivityName})
	worker.RegisterActivityWithOptions(HeartbeatTimeoutActivity, activity.RegisterOptions{Name: heartbeatTimeoutActivityName})

	service := nexus.NewService(nexusServiceName)
	if err := service.Register(echoOperation); err != nil {
		panic(err)
	}
	worker.RegisterNexusService(service)

	return worker
}

// buildClient is the app's ClientFactory.
func buildClient(config harness.ClientConfig) (sdkclient.Client, error) {
	dataConverter, err := newDataConverter()
	if err != nil {
		return nil, err
	}

	options := harness.BuildSDKClientOptions(config)
	options.DataConverter = dataConverter
	options.FailureConverter = temporal.NewDefaultFailureConverter(temporal.DefaultFailureConverterOptions{
		DataConverter:          dataConverter,
		EncodeCommonAttributes: true,
	})
	// Headers are the one payload field the data converter never sees, so a
	// propagator is the only way to produce them at all. See propagator.go.
	options.ContextPropagators = []workflow.ContextPropagator{plaintextPropagator{}}

	client, err := sdkclient.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return client, nil
}

// newDataConverter builds the encrypting data converter. The base is the SDK
// default converter.
func newDataConverter() (converter.DataConverter, error) {
	codec, err := newEncryptionCodec(testOnlyKey)
	if err != nil {
		return nil, err
	}
	return converter.NewCodecDataConverter(converter.GetDefaultDataConverter(), codec), nil
}

// initProject runs once before the first iteration. Nexus is optional: the app
// creates no endpoint, and a run that names none simply generates no Nexus load.
// A config file that cannot be parsed is still an error, since that is a typo
// rather than a choice.
func initProject(_ sdkclient.Client, initContext harness.ProjectInitContext) error {
	nexusHint := fmt.Sprintf(
		"to include Nexus load, pass --option project-config-file=<path> naming a JSON file that "+
			"sets nexusEndpoint to an endpoint targeting this run's namespace and task queue %q",
		initContext.TaskQueue)

	if len(initContext.ConfigJSON) == 0 {
		initContext.Logger.Warnf("No project config, so this run generates no Nexus load: %s", nexusHint)
		return nil
	}
	var config projectConfig
	if err := json.Unmarshal(initContext.ConfigJSON, &config); err != nil {
		return fmt.Errorf("failed to parse project config: %w", err)
	}
	if config.NexusEndpoint == "" {
		initContext.Logger.Warnf("Project config names no endpoint, so this run generates no Nexus load: %s", nexusHint)
		return nil
	}

	state.nexusEndpoint = config.NexusEndpoint
	initContext.Logger.Infof("Using Nexus endpoint %q", state.nexusEndpoint)
	return nil
}

// executeIteration runs one full pass over the payload paths. It is deliberately
// sequential rather than fast: the point is a history that is easy to read, not
// throughput.
func executeIteration(client sdkclient.Client, executeContext harness.ProjectExecuteContext) (err error) {
	base := fmt.Sprintf("%s-%d", executeContext.Run.ExecutionID, executeContext.Iteration)

	// An iteration that returns early leaves behind whatever it had already started
	// but never waited for. Those orphans outlive the run — omes stops the worker
	// when the scenario ends — so they close with a workflow task timeout that reads
	// like the cause of the failure rather than the fallout from it. Terminating
	// them keeps the real failure the only interesting thing in the namespace.
	var started []string
	defer func() {
		if err == nil {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), iterationCleanupTimeout)
		defer cancel()
		for _, id := range started {
			terminateErr := client.TerminateWorkflow(cleanupCtx, id, "", "encryption iteration failed")
			// Already closed is the common case and not worth a line.
			var notFound *serviceerror.NotFound
			if terminateErr == nil || errors.As(terminateErr, &notFound) {
				continue
			}
			executeContext.Logger.Warnf("Failed to terminate %s after a failed iteration: %v", id, terminateErr)
		}
	}()
	// The payload body every workflow, activity, signal, and marker in this
	// iteration carries. It is per-iteration only so that concurrent iterations are
	// distinguishable in a history.
	message := "encryption-coverage-" + base

	// The header the propagator injects comes off the context.
	ctx := withHeaderValue(context.Background(), "tenant-"+base)

	// The target is started with update-with-start, which the client issues as a
	// single ExecuteMultiOperation carrying a StartWorkflowExecutionRequest and an
	// UpdateWorkflowExecutionRequest nested inside operation wrappers. That is the
	// deepest payload-bearing request the client sends, and this one holds four
	// payloads at three different depths: the workflow input is absent but the memo,
	// the header, and the update argument are all in there.
	targetID := base + "-target"
	started = append(started, targetID)
	startTarget := client.NewWithStartWorkflowOperation(sdkclient.StartWorkflowOptions{
		ID:                       targetID,
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		// Required for update-with-start. USE_EXISTING rather than FAIL so that an
		// iteration omes retries attaches to the target its first attempt started.
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		Memo: map[string]any{
			"encryptionTargetMemo": MarkerData{Step: "target", Message: message},
		},
	}, targetWorkflowName)

	accepted, err := client.UpdateWithStartWorkflow(ctx, sdkclient.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startTarget,
		UpdateOptions: sdkclient.UpdateWorkflowOptions{
			UpdateName:   coverageUpdateName,
			WaitForStage: sdkclient.WorkflowUpdateStageCompleted,
			Args:         []any{UpdateInput{Step: "update-with-start", Message: message}},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to update-with-start the target workflow: %w", err)
	}
	var updateResult UpdateResult
	if err := accepted.Get(ctx, &updateResult); err != nil {
		return fmt.Errorf("update-with-start of the target workflow failed: %w", err)
	}
	target, err := startTarget.Get(ctx)
	if err != nil {
		return fmt.Errorf("failed to start target workflow: %w", err)
	}

	// A plain update too, on the other RPC, and one the validator rejects, which
	// produces a failure payload through the failure converter rather than a result.
	// It goes before the coverage workflow because the signal from that workflow
	// completes the target, and a completed workflow takes no updates. The rejection
	// is the point, so the error is expected and dropped; it can surface either from
	// the call or from Get depending on how far the update got.
	rejected, err := client.UpdateWorkflow(ctx, sdkclient.UpdateWorkflowOptions{
		WorkflowID:   targetID,
		UpdateName:   coverageUpdateName,
		WaitForStage: sdkclient.WorkflowUpdateStageCompleted,
		Args:         []any{UpdateInput{Step: "update-rejected", Message: message, Reject: true}},
	})
	if err == nil {
		_ = rejected.Get(ctx, nil)
	}

	// A top-level failing workflow, so the failure payloads are also decoded by the
	// client rather than only by a parent workflow.
	failing, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:                       base + "-failing",
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
	}, failingWorkflowName, FailingInput{Message: message})
	if err != nil {
		return fmt.Errorf("failed to start failing workflow: %w", err)
	}
	started = append(started, base+"-failing")

	coverage, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:                       base,
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		Memo: map[string]any{
			"encryptionStartMemo": MarkerData{Step: "start", Message: message},
		},
		TypedSearchAttributes: temporal.NewSearchAttributes(
			keywordSearchAttribute.ValueSet(message),
		),
	}, coverageWorkflowName, CoverageInput{
		Iteration:        executeContext.Iteration,
		Message:          message,
		TargetWorkflowID: targetID,
		NexusEndpoint:    state.nexusEndpoint,
	})
	if err != nil {
		return fmt.Errorf("failed to start coverage workflow: %w", err)
	}
	started = append(started, base)

	var coverageResult CoverageOutput
	if err := coverage.Get(ctx, &coverageResult); err != nil {
		return fmt.Errorf("coverage workflow failed: %w", err)
	}

	var signalled SignalInput
	if err := target.Get(ctx, &signalled); err != nil {
		return fmt.Errorf("target workflow failed: %w", err)
	}

	// This one is supposed to fail; waiting on it is what drives the client-side
	// decode of the failure, so the error is expected and ignored.
	_ = failing.Get(ctx, nil)

	// A workflow whose first attempt fails, so the second attempt's started event
	// carries the encoded failure as ContinuedFailure. Waiting on the run follows the
	// retry chain to the attempt that succeeds.
	retry, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:                       base + "-retry",
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		RetryPolicy:              &temporal.RetryPolicy{MaximumAttempts: 2},
	}, retryWorkflowName, RetryInput{Message: message})
	if err != nil {
		return fmt.Errorf("failed to start retry workflow: %w", err)
	}
	started = append(started, base+"-retry")
	// Decoded rather than discarded: Get skips FromPayloads entirely when handed a
	// nil pointer, so a result nobody decodes is converter work that never happens.
	var retryResult RetryResult
	if err := retry.Get(ctx, &retryResult); err != nil {
		return fmt.Errorf("retry workflow failed: %w", err)
	}
	return nil
}
