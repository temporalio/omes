package scenarios

// versioning_pinned_workflows implements a scenario for testing worker versioning with pinned workflows.
//
// This scenario uses the Worker Deployment APIs for worker versioning (non-deprecated).
// See: https://docs.temporal.io/develop/go/versioning
//
// Implementation approach:
// - Manages Go SDK workers directly within the scenario (not via OMES worker infrastructure)
// - Uses DeploymentOptions to configure workers with deployment names and build IDs
// - Starts multiple workers concurrently with different build IDs to support version bumping
// - Old workers remain running to handle pinned workflows while new workers handle new traffic
//
// The scenario:
// 1. Starts N workflows pinned to an initial version (default: 1)
// 2. Signals all workflows on each iteration
// 3. Bumps the version every N iterations by starting new workers and setting them as current
// 4. Verifies that workflow build IDs always move forward, never backward

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

// retryUntilCtx retries the given function until it reports done or the context is done.
// Backoff starts at 1s and is capped at 10s.
func retryUntilCtx(ctx context.Context, fn func(context.Context) (bool, error)) error {
	backoff := 1 * time.Second
	for {
		done, err := fn(ctx)
		if done {
			return err
		}
		select {
		case <-ctx.Done():
			if err != nil {
				return err
			}
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 10*time.Second {
			backoff *= 2
			if backoff > 10*time.Second {
				backoff = 10 * time.Second
			}
		}
	}
}

const (
	// NumWorkflowsFlag controls how many workflows to start on iteration 0
	NumWorkflowsFlag = "num-workflows"
	// VersionBumpIntervalFlag controls how many iterations between version bumps
	VersionBumpIntervalFlag = "version-bump-interval"
	// InitialVersionFlag is the initial version number to pin workflows to (default: 1)
	InitialVersionFlag = "initial-version"
)

type versioningPinnedState struct {
	WorkflowIDs     []string `json:"workflowIds"`
	CurrentVersion  string   `json:"currentVersion"`
	VersionSequence []string `json:"versionSequence"`
}

type versioningPinnedConfig struct {
	NumWorkflows        int
	VersionBumpInterval int
	InitialVersion      string
}

type versioningPinnedExecutor struct {
	lock           sync.Mutex
	state          *versioningPinnedState
	config         *versioningPinnedConfig
	workers        []worker.Worker // All active workers (one per version)
	deploymentName string
}

var _ loadgen.Configurable = (*versioningPinnedExecutor)(nil)

// noopActivity is a simple activity for testing
func noopActivity(_ context.Context) error {
	return nil
}

// simpleKitchenSinkWorkflow is a simplified kitchensink workflow for this scenario
// It executes a single activity and then waits indefinitely (until cancelled/terminated)
func simpleKitchenSinkWorkflow(ctx workflow.Context, params *kitchensink.WorkflowInput) (*commonpb.Payload, error) {
	// Execute a simple activity to generate history with build ID
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	activityCtx := workflow.WithActivityOptions(ctx, ao)
	if err := workflow.ExecuteActivity(activityCtx, "noop").Get(activityCtx, nil); err != nil {
		return nil, err
	}

	// Wait for signals indefinitely
	signalChan := workflow.GetSignalChannel(ctx, "do_signal")
	selector := workflow.NewSelector(ctx)

	// Keep workflow alive by continuously waiting for signals
	for {
		selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
			// Receive signal with the correct type (kitchensink.DoSignal)
			var signal kitchensink.DoSignal
			c.Receive(ctx, &signal)

			// Execute another activity when signaled
			ao := workflow.ActivityOptions{
				StartToCloseTimeout: 10 * time.Second,
			}
			activityCtx := workflow.WithActivityOptions(ctx, ao)
			_ = workflow.ExecuteActivity(activityCtx, "noop").Get(activityCtx, nil)
		})

		selector.Select(ctx)

		// Create a new selector for the next iteration
		selector = workflow.NewSelector(ctx)
	}
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Worker versioning scenario with pinned workflows. Starts n workflows pinned to version 1, "+
				"signals them each iteration, and bumps versions every n iterations. "+
				"Use --option with '%s' (default: 10), '%s' (default: 5), '%s' (default: 1)",
			NumWorkflowsFlag, VersionBumpIntervalFlag, InitialVersionFlag),
		ExecutorFn: func() loadgen.Executor { return newVersioningPinnedExecutor() },
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*versioningPinnedExecutor)
			return e.Verify(ctx, info)
		},
	})
}

func newVersioningPinnedExecutor() *versioningPinnedExecutor {
	return &versioningPinnedExecutor{
		state: &versioningPinnedState{
			WorkflowIDs:     []string{},
			CurrentVersion:  "",
			VersionSequence: []string{},
		},
	}
}

// Configure initializes the executor configuration from scenario options.
func (e *versioningPinnedExecutor) Configure(info loadgen.ScenarioInfo) error {
	initialVersionNum := info.ScenarioOptionInt(InitialVersionFlag, 1)

	config := &versioningPinnedConfig{
		NumWorkflows:        info.ScenarioOptionInt(NumWorkflowsFlag, 10),
		VersionBumpInterval: info.ScenarioOptionInt(VersionBumpIntervalFlag, 5),
		InitialVersion:      fmt.Sprintf("%d", initialVersionNum),
	}

	if config.NumWorkflows <= 0 {
		return fmt.Errorf("%s must be positive, got %d", NumWorkflowsFlag, config.NumWorkflows)
	}

	if config.VersionBumpInterval <= 0 {
		return fmt.Errorf("%s must be positive, got %d", VersionBumpIntervalFlag, config.VersionBumpInterval)
	}

	if initialVersionNum <= 0 {
		return fmt.Errorf("%s must be positive, got %d", InitialVersionFlag, initialVersionNum)
	}

	e.config = config
	return nil
}

// startWorker creates and starts a new worker with the specified build ID and deployment options.
func (e *versioningPinnedExecutor) startWorker(ctx context.Context, info loadgen.ScenarioInfo, buildID string) (worker.Worker, error) {
	taskQueue := info.RunID + ".local"

	// Create worker with deployment options
	w := worker.New(info.Client, taskQueue, worker.Options{
		BuildID:                 buildID,
		UseBuildIDForVersioning: true,
		DeploymentOptions: worker.DeploymentOptions{
			UseVersioning: true,
			Version: worker.WorkerDeploymentVersion{
				DeploymentName: e.deploymentName,
				BuildID:        buildID,
			},
			// Use Pinned behavior by default - workflows stay on the version they started with
			DefaultVersioningBehavior: workflow.VersioningBehaviorPinned,
		},
	})

	// Register workflow and activities
	w.RegisterWorkflowWithOptions(simpleKitchenSinkWorkflow, workflow.RegisterOptions{Name: "kitchenSink"})
	w.RegisterActivityWithOptions(noopActivity, activity.RegisterOptions{Name: "noop"})

	// Start the worker with retry until context done
	if err := retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
		if err := w.Start(); err != nil {
			return false, err
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to start worker with build ID %s: %w", buildID, err)
	}

	info.Logger.Infof("Started worker with build ID %s on task queue %s", buildID, taskQueue)
	return w, nil
}

// stopAllWorkers stops all running workers.
func (e *versioningPinnedExecutor) stopAllWorkers() {
	e.lock.Lock()
	workers := e.workers
	e.workers = nil
	e.lock.Unlock()

	for _, w := range workers {
		if w != nil {
			w.Stop()
		}
	}
}

// Run executes the versioning scenario.
func (e *versioningPinnedExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to configure scenario: %w", err)
	}

	e.lock.Lock()
	e.state.CurrentVersion = e.config.InitialVersion
	e.state.VersionSequence = []string{e.config.InitialVersion}
	e.deploymentName = fmt.Sprintf("omes-deployment-%s", info.RunID)
	e.lock.Unlock()

	// Ensure all workers are stopped when we exit
	defer e.stopAllWorkers()

	// Calculate total iterations
	totalIterations := info.Configuration.Iterations
	if totalIterations == 0 && info.Configuration.Duration > 0 {
		// Estimate iterations based on duration (assuming ~1 iteration per second)
		totalIterations = int(info.Configuration.Duration.Seconds())
	}

	for iteration := 0; iteration < totalIterations; iteration++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if iteration == 0 {
			// Iteration 0: Start worker with initial version and start workflows
			w, err := e.startWorker(ctx, info, e.state.CurrentVersion)
			if err != nil {
				return fmt.Errorf("failed to start initial worker: %w", err)
			}
			e.lock.Lock()
			e.workers = append(e.workers, w)
			e.lock.Unlock()

			// Wait for worker to be ready
			time.Sleep(1 * time.Second)

			// Start n kitchensink workflows
			if err := e.startWorkflows(ctx, info, iteration); err != nil {
				return fmt.Errorf("failed to start workflows on iteration 0: %w", err)
			}
			info.Logger.Infof("Started %d workflows pinned to version %s", e.config.NumWorkflows, e.state.CurrentVersion)
		} else {
			// Check if we need to bump the version
			if iteration > 0 && iteration%e.config.VersionBumpInterval == 0 {
				if err := e.bumpVersion(ctx, info); err != nil {
					return fmt.Errorf("failed to bump version on iteration %d: %w", iteration, err)
				}
			}

			// Send signals to all workflows
			if err := e.signalAllWorkflows(ctx, info, iteration); err != nil {
				// Log signal failures but don't fail the scenario (as per requirements)
				info.Logger.Warnf("Some signals failed on iteration %d: %v", iteration, err)
			}
		}

		// Add a small delay between iterations to avoid overwhelming the system
		time.Sleep(100 * time.Millisecond)
	}

	// After all iterations, terminate workflows to complete the scenario
	info.Logger.Info("Terminating workflows after scenario completion")
	e.lock.Lock()
	workflowIDs := make([]string, len(e.state.WorkflowIDs))
	copy(workflowIDs, e.state.WorkflowIDs)
	e.lock.Unlock()

	for _, workflowID := range workflowIDs {
		err := info.Client.TerminateWorkflow(ctx, workflowID, "", "scenario completed")
		if err != nil {
			info.Logger.Warnf("Failed to terminate workflow %s: %v", workflowID, err)
		}
	}

	return nil
}

// startWorkflows starts n kitchensink workflows pinned to the current version.
func (e *versioningPinnedExecutor) startWorkflows(ctx context.Context, info loadgen.ScenarioInfo, iteration int) error {
	e.lock.Lock()
	currentVersion := e.state.CurrentVersion
	deploymentName := e.deploymentName
	e.lock.Unlock()

	taskQueue := info.RunID + ".local"

	// Set the current version as the deployment's current version
	// The worker has already registered the deployment, now we set it as current
	if err := e.setupVersioning(ctx, info.Client, info.Namespace, deploymentName, currentVersion); err != nil {
		return fmt.Errorf("failed to setup versioning: %w", err)
	}

	var wg sync.WaitGroup
	errChan := make(chan error, e.config.NumWorkflows)

	for i := 0; i < e.config.NumWorkflows; i++ {
		wg.Add(1)
		go func(workflowNum int) {
			defer wg.Done()

			workflowID := fmt.Sprintf("%s-versioned-%d", info.RunID, workflowNum)

			// Create a long-running workflow that waits for signals
			testInput := &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{
						{
							Actions: []*kitchensink.Action{
								// Set a workflow state to track initialization
								kitchensink.NewSetWorkflowStateAction(fmt.Sprintf("workflow-%d-started", workflowNum), "true"),
								// Execute a noop activity to generate some history with build ID
								kitchensink.GenericActivity("noop", kitchensink.DefaultRemoteActivity),
								// Wait for completion signal (this state will be set by a final signal)
								kitchensink.NewAwaitWorkflowStateAction(fmt.Sprintf("workflow-%d-complete", workflowNum), "true"),
							},
							Concurrent: false,
						},
					},
				},
			}

			options := client.StartWorkflowOptions{
				ID:                       workflowID,
				TaskQueue:                taskQueue,
				WorkflowExecutionTimeout: 24 * time.Hour,
				SearchAttributes: map[string]any{
					loadgen.OmesExecutionIDSearchAttribute: info.ExecutionID,
				},
			}

			var startErr error
			if err := retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
				_, startErr = info.Client.ExecuteWorkflow(
					ctx,
					options,
					"kitchenSink",
					testInput.WorkflowInput,
				)
				if startErr == nil {
					return true, nil
				}
				// Treat AlreadyStarted as success for idempotency
				if _, ok := startErr.(*serviceerror.WorkflowExecutionAlreadyStarted); ok {
					return true, nil
				}
				return false, startErr
			}); err != nil {
				errChan <- fmt.Errorf("failed to start workflow %s: %w", workflowID, startErr)
				return
			}

			e.lock.Lock()
			e.state.WorkflowIDs = append(e.state.WorkflowIDs, workflowID)
			e.lock.Unlock()
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Collect any errors
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to start %d workflows: %v", len(errs), errs[0])
	}

	return nil
}

// setupVersioning configures the worker versioning for the deployment using Worker Deployment APIs.
func (e *versioningPinnedExecutor) setupVersioning(ctx context.Context, c client.Client, namespace, deploymentName, buildID string) error {
	if err := retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
		_, err := c.WorkflowService().SetWorkerDeploymentCurrentVersion(ctx, &workflowservice.SetWorkerDeploymentCurrentVersionRequest{
			Namespace:      namespace,
			DeploymentName: deploymentName,
			BuildId:        buildID,
		})
		return err == nil, err
	}); err != nil {
		return fmt.Errorf("failed to set version %s as current for deployment %s: %w", buildID, deploymentName, err)
	}
	return nil
}

// bumpVersion increases the version, starts a new worker with the new build ID, and sets it as current.
func (e *versioningPinnedExecutor) bumpVersion(ctx context.Context, info loadgen.ScenarioInfo) error {
	e.lock.Lock()
	// Parse current version (e.g., "1" -> 1) and increment
	var versionNum int
	_, err := fmt.Sscanf(e.state.CurrentVersion, "%d", &versionNum)
	if err != nil {
		e.lock.Unlock()
		return fmt.Errorf("failed to parse version %s: %w", e.state.CurrentVersion, err)
	}

	versionNum++
	newVersion := fmt.Sprintf("%d", versionNum)
	e.lock.Unlock()

	// Start a new worker with the new build ID
	// This keeps the old worker running to handle pinned workflows
	w, err := e.startWorker(ctx, info, newVersion)
	if err != nil {
		return fmt.Errorf("failed to start worker for version %s: %w", newVersion, err)
	}

	e.lock.Lock()
	e.workers = append(e.workers, w)
	deploymentName := e.deploymentName
	e.lock.Unlock()

	// Wait for worker to be ready
	time.Sleep(1 * time.Second)

	// Retry indefinitely until ctx is done when setting the new current version
	// Set the new version as the current deployment version (retry until ctx done)
	if err := retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
		_, err = info.Client.WorkflowService().SetWorkerDeploymentCurrentVersion(ctx, &workflowservice.SetWorkerDeploymentCurrentVersionRequest{
			Namespace:      info.Namespace,
			DeploymentName: deploymentName,
			BuildId:        newVersion,
		})
		return err == nil, err
	}); err != nil {
		return fmt.Errorf("failed to set version %s as current: %w", newVersion, err)
	}

	e.lock.Lock()
	info.Logger.Infof("Bumped version from %s to %s", e.state.CurrentVersion, newVersion)
	e.state.CurrentVersion = newVersion
	e.state.VersionSequence = append(e.state.VersionSequence, newVersion)
	e.lock.Unlock()

	return nil
}

// signalAllWorkflows sends a signal to all tracked workflows.
func (e *versioningPinnedExecutor) signalAllWorkflows(ctx context.Context, info loadgen.ScenarioInfo, iteration int) error {
	e.lock.Lock()
	workflowIDs := make([]string, len(e.state.WorkflowIDs))
	copy(workflowIDs, e.state.WorkflowIDs)
	e.lock.Unlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(workflowIDs))

	for _, workflowID := range workflowIDs {
		wg.Add(1)
		go func(wfID string) {
			defer wg.Done()

			// Send a signal that executes a simple action
			signalAction := &kitchensink.DoSignal{
				Variant: &kitchensink.DoSignal_DoSignalActions_{
					DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActions{
							DoActions: kitchensink.SingleActionSet(
								// Execute a noop activity as part of signal processing
								kitchensink.GenericActivity("noop", kitchensink.DefaultLocalActivity),
							),
						},
					},
				},
			}

			err := info.Client.SignalWorkflow(
				ctx,
				wfID,
				"",
				"do_signal",
				signalAction,
			)
			if err != nil {
				// As per requirements, we ignore signal failures
				info.Logger.Warnf("Signal failed for workflow %s: %v", wfID, err)
			}
		}(workflowID)
	}

	wg.Wait()
	close(errChan)

	return nil
}

// Verify checks that each workflow's build ID always moved forward and never backward.
func (e *versioningPinnedExecutor) Verify(ctx context.Context, info loadgen.ScenarioInfo) []error {
	e.lock.Lock()
	workflowIDs := make([]string, len(e.state.WorkflowIDs))
	copy(workflowIDs, e.state.WorkflowIDs)
	e.lock.Unlock()

	var errors []error
	var errorsMutex sync.Mutex

	// Check each workflow's history
	var wg sync.WaitGroup
	for _, workflowID := range workflowIDs {
		wg.Add(1)
		go func(wfID string) {
			defer wg.Done()

			violations := e.checkWorkflowHistory(ctx, info, wfID)
			if len(violations) > 0 {
				errorsMutex.Lock()
				errors = append(errors, violations...)
				errorsMutex.Unlock()
			}
		}(workflowID)
	}

	wg.Wait()

	if len(errors) == 0 {
		info.Logger.Infof("Verification passed: All %d workflows maintained forward-only version progression", len(workflowIDs))
	} else {
		info.Logger.Errorf("Verification failed: Found %d version progression violations", len(errors))
	}

	return errors
}

// checkWorkflowHistory checks a workflow's versioning info for build ID violations.
func (e *versioningPinnedExecutor) checkWorkflowHistory(ctx context.Context, info loadgen.ScenarioInfo, workflowID string) []error {
	var errors []error

	// Get workflow execution description to access versioning info (with retry)
	var describeResp *workflowservice.DescribeWorkflowExecutionResponse
	var derr error
	_ = retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
		describeResp, derr = info.Client.WorkflowService().DescribeWorkflowExecution(ctx, &workflowservice.DescribeWorkflowExecutionRequest{
			Namespace: info.Namespace,
			Execution: &commonpb.WorkflowExecution{
				WorkflowId: workflowID,
			},
		})
		return derr == nil, derr
	})
	if derr != nil {
		errors = append(errors, fmt.Errorf("workflow %s: failed to describe execution: %w", workflowID, derr))
		return errors
	}

	versioningInfo := describeResp.WorkflowExecutionInfo.GetVersioningInfo()
	if versioningInfo == nil {
		errors = append(errors, fmt.Errorf("workflow %s: no versioning info found", workflowID))
		return errors
	}

	// Get workflow history to track build ID sequence
	historyIter := info.Client.GetWorkflowHistory(
		ctx,
		workflowID,
		"",
		false,
		enums.HISTORY_EVENT_FILTER_TYPE_ALL_EVENT,
	)

	var buildIDSequence []string
	buildIDVersionMap := make(map[string]int) // Map build IDs to version numbers

	// Parse version numbers from version sequence
	e.lock.Lock()
	for _, version := range e.state.VersionSequence {
		var versionNum int
		fmt.Sscanf(version, "%d", &versionNum)
		buildIDVersionMap[version] = versionNum
	}
	e.lock.Unlock()

	// Iterate through history events to track build ID progression
	// Use Started events (not deprecated) instead of Completed events
	for historyIter.HasNext() {
		var event *historypb.HistoryEvent
		if err := retryUntilCtx(ctx, func(ctx context.Context) (bool, error) {
			var err error
			event, err = historyIter.Next()
			if err != nil {
				return false, err
			}
			return true, nil
		}); err != nil {
			errors = append(errors, fmt.Errorf("workflow %s: failed to read history: %w", workflowID, err))
			return errors
		}

		// Check for build ID in Started events (GetWorkerVersion on Started events is not deprecated)
		var buildID string
		switch event.EventType {
		case enums.EVENT_TYPE_WORKFLOW_TASK_STARTED:
			if event.GetWorkflowTaskStartedEventAttributes() != nil &&
				event.GetWorkflowTaskStartedEventAttributes().GetWorkerVersion() != nil {
				buildID = event.GetWorkflowTaskStartedEventAttributes().GetWorkerVersion().GetBuildId()
			}
		case enums.EVENT_TYPE_ACTIVITY_TASK_STARTED:
			if event.GetActivityTaskStartedEventAttributes() != nil &&
				event.GetActivityTaskStartedEventAttributes().GetWorkerVersion() != nil {
				buildID = event.GetActivityTaskStartedEventAttributes().GetWorkerVersion().GetBuildId()
			}
		}

		// If we found a build ID, track it (avoid duplicates)
		if buildID != "" {
			if len(buildIDSequence) == 0 || buildIDSequence[len(buildIDSequence)-1] != buildID {
				buildIDSequence = append(buildIDSequence, buildID)
			}
		}
	}

	// Check that build IDs never moved backward
	for i := 1; i < len(buildIDSequence); i++ {
		prevBuildID := buildIDSequence[i-1]
		currBuildID := buildIDSequence[i]

		prevVersion, prevExists := buildIDVersionMap[prevBuildID]
		currVersion, currExists := buildIDVersionMap[currBuildID]

		if !prevExists {
			errors = append(errors, fmt.Errorf(
				"workflow %s: unknown build ID '%s' at position %d in sequence",
				workflowID, prevBuildID, i-1))
			continue
		}

		if !currExists {
			errors = append(errors, fmt.Errorf(
				"workflow %s: unknown build ID '%s' at position %d in sequence",
				workflowID, currBuildID, i))
			continue
		}

		if currVersion < prevVersion {
			errors = append(errors, fmt.Errorf(
				"workflow %s: build ID moved backward from %s (%d) to %s (%d) at history position %d",
				workflowID, prevBuildID, prevVersion, currBuildID, currVersion, i))
		}
	}

	return errors
}
