// Package encryption is an omes project that installs an encrypting data
// converter and drives a workflow.
//
// See README.md for which paths end up encrypted, which do not, and how the
// fan-out knobs turn one readable iteration into a load generator.
package encryption

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
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

// state carries the parsed configuration from Init to Execute.
var state struct {
	config projectConfig
	filler string
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

	// ConcurrentUpdates is how many updates the driver sends to the target
	// workflow at once, beyond the one carried by update-with-start.
	ConcurrentUpdates int `json:"concurrentUpdates"`

	// MemoEntries is how many fields each memo carries — the start memo and the
	// target's memo here, and the upserted and child memos in the workflow.
	MemoEntries int `json:"memoEntries"`
	// ActivityCount fans out the echo activity and the failing activity.
	ActivityCount int `json:"activityCount"`
	// MarkerCount fans out side effects and failing local activities.
	MarkerCount int `json:"markerCount"`
	// ChildCount fans out the echo child workflow.
	ChildCount int `json:"childCount"`
	// SignalCount is how many signals reach the target workflow, which waits for
	// exactly this many before completing.
	SignalCount int `json:"signalCount"`
	// NexusCount fans out the Nexus operation, when an endpoint is configured.
	NexusCount int `json:"nexusCount"`
	// FailureDepth is how many levels deep a failure's cause chain goes. Each
	// level is its own Failure proto with its own details and, because the
	// failure converter sets EncodeCommonAttributes, its own encoded_attributes.
	// Depth therefore adds proto nodes rather than only enlarging one payload.
	FailureDepth int `json:"failureDepth"`
	// PayloadBytes pads every payload body.
	PayloadBytes int `json:"payloadBytes"`
}

// withDefaults clamps the counts so that an unconfigured run is the one-of-each
// iteration and a zero never silently turns a path off.
func (c projectConfig) withDefaults() projectConfig {
	for _, count := range []*int{
		&c.ConcurrentUpdates, &c.MemoEntries, &c.ActivityCount, &c.MarkerCount,
		&c.ChildCount, &c.SignalCount, &c.NexusCount, &c.FailureDepth,
	} {
		if *count < 1 {
			*count = 1
		}
	}
	if c.PayloadBytes < 0 {
		c.PayloadBytes = 0
	}
	return c
}

// coverageInput builds the coverage workflow's input.
func coverageInput(config projectConfig, filler, message, targetID string, iteration int64) CoverageInput {
	return CoverageInput{
		Message:          message,
		Iteration:        iteration,
		TargetWorkflowID: targetID,
		NexusEndpoint:    config.NexusEndpoint,
		Filler:           filler,
		MemoEntries:      config.MemoEntries,
		ActivityCount:    config.ActivityCount,
		MarkerCount:      config.MarkerCount,
		ChildCount:       config.ChildCount,
		SignalCount:      config.SignalCount,
		NexusCount:       config.NexusCount,
		FailureDepth:     config.FailureDepth,
	}
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

	var config projectConfig
	if len(initContext.ConfigJSON) == 0 {
		initContext.Logger.Warnf("No project config, so this run generates no Nexus load: %s", nexusHint)
	} else if err := json.Unmarshal(initContext.ConfigJSON, &config); err != nil {
		return fmt.Errorf("failed to parse project config: %w", err)
	}

	if config.NexusEndpoint == "" {
		if len(initContext.ConfigJSON) > 0 {
			initContext.Logger.Warnf("Project config names no endpoint, so this run generates no Nexus load: %s", nexusHint)
		}
	} else {
		initContext.Logger.Infof("Using Nexus endpoint %q", config.NexusEndpoint)
	}

	state.config = config.withDefaults()
	state.filler = makeFiller(state.config.PayloadBytes)

	// A payload census, so a latency measurement taken against this run can be
	// read as a per-payload cost rather than an opaque aggregate.
	initContext.Logger.Infof(
		"Per-iteration fan-out: memoEntries=%d activityCount=%d markerCount=%d childCount=%d "+
			"signalCount=%d nexusCount=%d failureDepth=%d payloadBytes=%d concurrentUpdates=%d",
		state.config.MemoEntries, state.config.ActivityCount, state.config.MarkerCount,
		state.config.ChildCount, state.config.SignalCount, state.config.NexusCount,
		state.config.FailureDepth, state.config.PayloadBytes, state.config.ConcurrentUpdates)

	return nil
}

// executeIteration runs one full pass over the payload paths. Every path is
// covered at every setting; the shape only decides how much each request
// carries.
func executeIteration(client sdkclient.Client, executeContext harness.ProjectExecuteContext) (err error) {
	base := fmt.Sprintf("%s-%d", executeContext.Run.ExecutionID, executeContext.Iteration)
	config := state.config
	filler := state.filler

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
	// iteration carries.
	message := "encryption-coverage-" + base

	// The header the propagator injects comes off the context.
	ctx := withHeaderValue(context.Background(), "tenant-"+base)

	// The target is started with update-with-start, which the client issues as a
	// single ExecuteMultiOperation carrying a StartWorkflowExecutionRequest and an
	// UpdateWorkflowExecutionRequest nested inside operation wrappers.
	targetID := base + "-target"
	started = append(started, targetID)
	startTarget := client.NewWithStartWorkflowOperation(sdkclient.StartWorkflowOptions{
		ID:                       targetID,
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		// Required for update-with-start. USE_EXISTING rather than FAIL so that an
		// iteration omes retries attaches to the target its first attempt started.
		WorkflowIDConflictPolicy: enumspb.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		Memo:                     iterationMemo("encryptionTargetMemo", "target", config.MemoEntries, message, filler),
	}, targetWorkflowName, TargetInput{ExpectedSignals: config.SignalCount})

	accepted, err := client.UpdateWithStartWorkflow(ctx, sdkclient.UpdateWithStartWorkflowOptions{
		StartWorkflowOperation: startTarget,
		UpdateOptions: sdkclient.UpdateWorkflowOptions{
			UpdateName:   coverageUpdateName,
			WaitForStage: sdkclient.WorkflowUpdateStageCompleted,
			Args: []any{UpdateInput{
				Step: "update-with-start", Message: message, Filler: filler,
			}},
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

	var updateWG sync.WaitGroup
	for i := 0; i < config.ConcurrentUpdates; i++ {
		updateWG.Add(1)
		go func(index int) {
			defer updateWG.Done()
			reject := index == 0
			step := "update-accepted"
			if reject {
				step = "update-rejected"
			}
			handle, updateErr := client.UpdateWorkflow(ctx, sdkclient.UpdateWorkflowOptions{
				WorkflowID:   targetID,
				UpdateName:   coverageUpdateName,
				WaitForStage: sdkclient.WorkflowUpdateStageCompleted,
				Args: []any{UpdateInput{
					Step:         step,
					Message:      message,
					Reject:       reject,
					Filler:       filler,
					FailureDepth: config.FailureDepth,
				}},
			})
			if updateErr == nil {
				_ = handle.Get(ctx, nil)
			}
		}(i)
	}
	updateWG.Wait()

	coverage, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:                       base,
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		Memo:                     iterationMemo("encryptionStartMemo", "start", config.MemoEntries, message, filler),
		TypedSearchAttributes: temporal.NewSearchAttributes(
			keywordSearchAttribute.ValueSet(message),
		),
	}, coverageWorkflowName, coverageInput(config, filler, message, targetID, executeContext.Iteration))
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

	// A workflow whose first attempt fails, so the second attempt's started event
	// carries the encoded failure as ContinuedFailure. Waiting on the run follows the
	// retry chain to the attempt that succeeds.
	retry, err := client.ExecuteWorkflow(ctx, sdkclient.StartWorkflowOptions{
		ID:                       base + "-retry",
		TaskQueue:                executeContext.TaskQueue,
		WorkflowExecutionTimeout: workflowExecutionTimeout,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
			InitialInterval: retryInitialInterval,
		},
	}, retryWorkflowName, RetryInput{
		Message: message, Filler: filler, FailureDepth: config.FailureDepth,
	})
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

// iterationMemo builds a memo with the configured number of fields.
func iterationMemo(prefix, step string, entries int, message, filler string) map[string]any {
	memo := make(map[string]any, entries)
	for i := 0; i < entries; i++ {
		memo[fmt.Sprintf("%s-%d", prefix, i)] = MarkerData{
			Step: step, Message: message, Filler: filler,
		}
	}
	return memo
}
