package scenarios

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	deploymentpb "go.temporal.io/api/deployment/v1"
	enumspb "go.temporal.io/api/enums/v1"
	workflowservicepb "go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	sdkworker "go.temporal.io/sdk/worker"
	"golang.org/x/sync/errgroup"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

const (
	wvcosDeploymentNameFlag           = "deployment-name"
	wvcosV1BuildIDFlag                = "v1-build-id"
	wvcosV2BuildIDFlag                = "v2-build-id"
	wvcosWorkflowCountFlag            = "workflow-count"
	wvcosStartConcurrencyFlag         = "start-concurrency"
	wvcosReleaseConcurrencyFlag       = "release-concurrency"
	wvcosStatusTimeoutFlag            = "status-timeout"
	wvcosSetCurrentTimeoutFlag        = "set-current-timeout"
	wvcosReleaseTimeoutFlag           = "release-timeout"
	wvcosWorkflowExecutionTimeoutFlag = "workflow-execution-timeout"
)

type workerVersioningCurrentOverrideStormExecutor struct {
	config workerVersioningCurrentOverrideStormConfig
}

type workerVersioningCurrentOverrideStormConfig struct {
	DeploymentName           string
	V1BuildID                string
	V2BuildID                string
	WorkflowCount            int
	StartConcurrency         int
	ReleaseConcurrency       int
	StatusTimeout            time.Duration
	SetCurrentTimeout        time.Duration
	ReleaseTimeout           time.Duration
	WorkflowExecutionTimeout time.Duration
}

type workerVersioningStormWorkflow struct {
	ID    string
	RunID string
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Starts many KitchenSink workflows pinned to the current worker deployment version, " +
			"then promotes another version and verifies routing propagation plus draining state. " +
			"Options: " + wvcosDeploymentNameFlag + " (required), " +
			wvcosV1BuildIDFlag + " (default v1), " +
			wvcosV2BuildIDFlag + " (default v2), " +
			wvcosWorkflowCountFlag + " (default 100), " +
			wvcosStartConcurrencyFlag + " (default 50).",
		ExecutorFn: func() loadgen.Executor {
			return &workerVersioningCurrentOverrideStormExecutor{}
		},
	})
}

func (e *workerVersioningCurrentOverrideStormExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := workerVersioningCurrentOverrideStormConfig{
		DeploymentName:           strings.TrimSpace(info.ScenarioOptionString(wvcosDeploymentNameFlag, "")),
		V1BuildID:                strings.TrimSpace(info.ScenarioOptionString(wvcosV1BuildIDFlag, "v1")),
		V2BuildID:                strings.TrimSpace(info.ScenarioOptionString(wvcosV2BuildIDFlag, "v2")),
		WorkflowCount:            info.ScenarioOptionInt(wvcosWorkflowCountFlag, 100),
		StartConcurrency:         info.ScenarioOptionInt(wvcosStartConcurrencyFlag, 50),
		StatusTimeout:            info.ScenarioOptionDuration(wvcosStatusTimeoutFlag, 2*time.Minute),
		SetCurrentTimeout:        info.ScenarioOptionDuration(wvcosSetCurrentTimeoutFlag, 5*time.Minute),
		ReleaseTimeout:           info.ScenarioOptionDuration(wvcosReleaseTimeoutFlag, 10*time.Minute),
		WorkflowExecutionTimeout: info.ScenarioOptionDuration(wvcosWorkflowExecutionTimeoutFlag, 30*time.Minute),
	}
	config.ReleaseConcurrency = info.ScenarioOptionInt(wvcosReleaseConcurrencyFlag, config.StartConcurrency)

	if config.DeploymentName == "" {
		return fmt.Errorf("%s is required", wvcosDeploymentNameFlag)
	}
	if config.V1BuildID == "" {
		return fmt.Errorf("%s is required", wvcosV1BuildIDFlag)
	}
	if config.V2BuildID == "" {
		return fmt.Errorf("%s is required", wvcosV2BuildIDFlag)
	}
	if config.V1BuildID == config.V2BuildID {
		return fmt.Errorf("%s and %s must differ", wvcosV1BuildIDFlag, wvcosV2BuildIDFlag)
	}
	if config.WorkflowCount <= 0 {
		return fmt.Errorf("%s must be positive", wvcosWorkflowCountFlag)
	}
	if config.StartConcurrency <= 0 {
		return fmt.Errorf("%s must be positive", wvcosStartConcurrencyFlag)
	}
	if config.ReleaseConcurrency <= 0 {
		return fmt.Errorf("%s must be positive", wvcosReleaseConcurrencyFlag)
	}
	if config.StatusTimeout <= 0 {
		return fmt.Errorf("%s must be positive", wvcosStatusTimeoutFlag)
	}
	if config.SetCurrentTimeout <= 0 {
		return fmt.Errorf("%s must be positive", wvcosSetCurrentTimeoutFlag)
	}
	if config.ReleaseTimeout <= 0 {
		return fmt.Errorf("%s must be positive", wvcosReleaseTimeoutFlag)
	}
	if config.WorkflowExecutionTimeout <= 0 {
		return fmt.Errorf("%s must be positive", wvcosWorkflowExecutionTimeoutFlag)
	}

	e.config = config
	return nil
}

func (e *workerVersioningCurrentOverrideStormExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) (err error) {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}
	if info.Client == nil {
		return fmt.Errorf("scenario requires a Temporal client")
	}

	if info.Configuration.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, info.Configuration.Timeout)
		defer cancel()
	}

	if !info.Configuration.DoNotRegisterSearchAttributes {
		if err := info.RegisterDefaultSearchAttributes(ctx); err != nil {
			return err
		}
	}

	config := e.config
	var workflows []workerVersioningStormWorkflow
	cleanupPending := false
	defer func() {
		if !cleanupPending {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), config.ReleaseTimeout)
		defer cancel()
		cleanupErr := e.releaseWorkflows(cleanupCtx, info, config, workflows)
		if cleanupErr == nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("failed to cleanup pinned workflows: %w", cleanupErr)
			return
		}
		info.Logger.Warnf("failed to cleanup pinned workflows after scenario error: %v", cleanupErr)
	}()

	info.Logger.Infof("Waiting for worker deployment versions %s and %s to register",
		canonicalDeploymentVersion(config.DeploymentName, config.V1BuildID),
		canonicalDeploymentVersion(config.DeploymentName, config.V2BuildID))
	if _, err := e.waitForVersionRegistered(ctx, info, config, config.V1BuildID); err != nil {
		return err
	}
	if _, err := e.waitForVersionRegistered(ctx, info, config, config.V2BuildID); err != nil {
		return err
	}

	info.Logger.Infof("Setting %s as current", canonicalDeploymentVersion(config.DeploymentName, config.V1BuildID))
	if err := e.setCurrentVersion(ctx, info, config, config.V1BuildID); err != nil {
		return fmt.Errorf("failed setting v1 current: %w", err)
	}

	info.Logger.Infof("Starting %d workflows pinned to %s",
		config.WorkflowCount,
		canonicalDeploymentVersion(config.DeploymentName, config.V1BuildID))
	workflows, err = e.startPinnedWorkflows(ctx, info, config)
	if len(workflows) > 0 {
		cleanupPending = true
	}
	if err != nil {
		return err
	}

	info.Logger.Infof("Setting %s as current", canonicalDeploymentVersion(config.DeploymentName, config.V2BuildID))
	if err := e.setCurrentVersion(ctx, info, config, config.V2BuildID); err != nil {
		return fmt.Errorf("failed setting v2 current after pinned override storm: %w", err)
	}
	if _, err := e.waitForVersionStatus(ctx, info, config, config.V1BuildID, enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_DRAINING); err != nil {
		return fmt.Errorf("failed waiting for v1 to become draining: %w", err)
	}

	info.Logger.Infof("Releasing %d pinned workflows", len(workflows))
	if err := e.releaseWorkflows(ctx, info, config, workflows); err != nil {
		return fmt.Errorf("failed releasing pinned workflows: %w", err)
	}
	cleanupPending = false

	info.Logger.Infof("Worker versioning current override storm completed successfully")
	return nil
}

func (e *workerVersioningCurrentOverrideStormExecutor) startPinnedWorkflows(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
) ([]workerVersioningStormWorkflow, error) {
	group, groupCtx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, config.StartConcurrency)
	var started atomic.Int64
	var lock sync.Mutex
	workflows := make([]workerVersioningStormWorkflow, 0, config.WorkflowCount)

	for i := 0; i < config.WorkflowCount; i++ {
		iteration := i + 1
		group.Go(func() error {
			if err := acquire(groupCtx, sem); err != nil {
				return err
			}
			defer release(sem)

			run := info.NewRun(iteration)
			options := run.DefaultStartWorkflowOptions()
			options.WorkflowExecutionTimeout = config.WorkflowExecutionTimeout
			options.WorkflowRunTimeout = config.WorkflowExecutionTimeout
			options.VersioningOverride = &client.PinnedVersioningOverride{
				Version: sdkworker.WorkerDeploymentVersion{
					DeploymentName: config.DeploymentName,
					BuildID:        config.V1BuildID,
				},
			}

			workflowRun, err := info.Client.ExecuteWorkflow(
				groupCtx,
				options,
				"kitchenSink",
				&kitchensink.WorkflowInput{},
			)
			if err != nil {
				return fmt.Errorf("failed starting pinned workflow %s: %w", options.ID, err)
			}

			lock.Lock()
			workflows = append(workflows, workerVersioningStormWorkflow{
				ID:    workflowRun.GetID(),
				RunID: workflowRun.GetRunID(),
			})
			lock.Unlock()

			count := started.Add(1)
			if count%1000 == 0 || int(count) == config.WorkflowCount {
				info.Logger.Infof("Started %d/%d pinned workflows", count, config.WorkflowCount)
			}
			return nil
		})
	}

	err := group.Wait()
	return workflows, err
}

func (e *workerVersioningCurrentOverrideStormExecutor) releaseWorkflows(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	workflows []workerVersioningStormWorkflow,
) error {
	group, groupCtx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, config.ReleaseConcurrency)
	for _, workflow := range workflows {
		workflow := workflow
		group.Go(func() error {
			if err := acquire(groupCtx, sem); err != nil {
				return err
			}
			defer release(sem)

			signal := &kitchensink.DoSignal_DoSignalActions{
				Variant: &kitchensink.DoSignal_DoSignalActions_DoActions{
					DoActions: kitchensink.SingleActionSet(kitchensink.NewEmptyReturnResultAction()),
				},
			}
			if err := info.Client.SignalWorkflow(groupCtx, workflow.ID, workflow.RunID, "do_actions_signal", signal); err != nil {
				return fmt.Errorf("failed signaling workflow %s/%s: %w", workflow.ID, workflow.RunID, err)
			}
			if err := info.Client.GetWorkflow(groupCtx, workflow.ID, workflow.RunID).Get(groupCtx, nil); err != nil {
				return fmt.Errorf("failed waiting for workflow %s/%s to complete: %w", workflow.ID, workflow.RunID, err)
			}
			return nil
		})
	}
	return group.Wait()
}

func (e *workerVersioningCurrentOverrideStormExecutor) setCurrentVersion(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	buildID string,
) error {
	setCtx, cancel := context.WithTimeout(ctx, config.SetCurrentTimeout)
	defer cancel()
	_, err := info.Client.WorkerDeploymentClient().
		GetHandle(config.DeploymentName).
		SetCurrentVersion(setCtx, client.WorkerDeploymentSetCurrentVersionOptions{
			BuildID: buildID,
		})
	if err != nil {
		return err
	}
	return e.waitForCurrentRouting(setCtx, info, config, buildID)
}

func (e *workerVersioningCurrentOverrideStormExecutor) waitForVersionRegistered(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	buildID string,
) (*deploymentpb.WorkerDeploymentVersionInfo, error) {
	return e.waitForVersion(ctx, info, config, buildID, "registered", func(versionInfo *deploymentpb.WorkerDeploymentVersionInfo) (bool, string) {
		return deploymentVersionMatches(versionInfo.GetDeploymentVersion(), config.DeploymentName, buildID),
			fmt.Sprintf("status=%s version=%s",
				versionInfo.GetStatus().String(),
				deploymentVersionString(versionInfo.GetDeploymentVersion()))
	})
}

func (e *workerVersioningCurrentOverrideStormExecutor) waitForVersionStatus(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	buildID string,
	status enumspb.WorkerDeploymentVersionStatus,
) (*deploymentpb.WorkerDeploymentVersionInfo, error) {
	return e.waitForVersion(ctx, info, config, buildID, status.String(), func(versionInfo *deploymentpb.WorkerDeploymentVersionInfo) (bool, string) {
		return versionInfo.GetStatus() == status,
			fmt.Sprintf("status=%s version=%s",
				versionInfo.GetStatus().String(),
				deploymentVersionString(versionInfo.GetDeploymentVersion()))
	})
}

func (e *workerVersioningCurrentOverrideStormExecutor) waitForVersion(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	buildID string,
	expected string,
	predicate func(*deploymentpb.WorkerDeploymentVersionInfo) (bool, string),
) (*deploymentpb.WorkerDeploymentVersionInfo, error) {
	waitCtx, cancel := context.WithTimeout(ctx, config.StatusTimeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastSeen string
	var lastErr error
	for {
		versionInfo, err := e.describeWorkerDeploymentVersion(waitCtx, info, config, buildID)
		if err != nil {
			lastErr = err
		} else if versionInfo == nil {
			lastSeen = "missing version info"
		} else if ok, detail := predicate(versionInfo); ok {
			return versionInfo, nil
		} else {
			lastSeen = detail
		}

		select {
		case <-ticker.C:
		case <-waitCtx.Done():
			return nil, fmt.Errorf("timed out waiting for %s to be %s; last seen: %s; last error: %v: %w",
				canonicalDeploymentVersion(config.DeploymentName, buildID),
				expected,
				lastSeen,
				lastErr,
				waitCtx.Err())
		}
	}
}

func (e *workerVersioningCurrentOverrideStormExecutor) describeWorkerDeploymentVersion(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	buildID string,
) (*deploymentpb.WorkerDeploymentVersionInfo, error) {
	resp, err := info.Client.WorkflowService().DescribeWorkerDeploymentVersion(ctx, &workflowservicepb.DescribeWorkerDeploymentVersionRequest{
		Namespace: info.Namespace,
		DeploymentVersion: &deploymentpb.WorkerDeploymentVersion{
			DeploymentName: config.DeploymentName,
			BuildId:        buildID,
		},
	})
	if err != nil {
		return nil, err
	}
	return resp.GetWorkerDeploymentVersionInfo(), nil
}

func (e *workerVersioningCurrentOverrideStormExecutor) waitForCurrentRouting(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	config workerVersioningCurrentOverrideStormConfig,
	expectedBuildID string,
) error {
	waitCtx, cancel := context.WithTimeout(ctx, config.StatusTimeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastSeen string
	var lastErr error
	for {
		resp, err := info.Client.WorkflowService().DescribeWorkerDeployment(waitCtx, &workflowservicepb.DescribeWorkerDeploymentRequest{
			Namespace:      info.Namespace,
			DeploymentName: config.DeploymentName,
		})
		if err != nil {
			lastErr = err
		} else {
			deploymentInfo := resp.GetWorkerDeploymentInfo()
			if deploymentInfo == nil {
				lastSeen = "missing deployment info"
			} else {
				routing := deploymentInfo.GetRoutingConfig()
				current := routing.GetCurrentDeploymentVersion()
				ramping := routing.GetRampingDeploymentVersion()
				currentOK := deploymentVersionMatches(current, config.DeploymentName, expectedBuildID) ||
					routing.GetCurrentVersion() == canonicalDeploymentVersion(config.DeploymentName, expectedBuildID)
				rampingClear := ramping == nil &&
					routing.GetRampingVersion() == "" &&
					routing.GetRampingVersionPercentage() == 0
				propagationComplete := deploymentInfo.GetRoutingConfigUpdateState() == enumspb.ROUTING_CONFIG_UPDATE_STATE_COMPLETED
				lastSeen = fmt.Sprintf("current=%s ramping=%s rampPercentage=%v propagation=%s",
					deploymentVersionString(current),
					deploymentVersionString(ramping),
					routing.GetRampingVersionPercentage(),
					deploymentInfo.GetRoutingConfigUpdateState().String())
				if currentOK && rampingClear && propagationComplete {
					return nil
				}
			}
		}

		select {
		case <-ticker.C:
		case <-waitCtx.Done():
			return fmt.Errorf("timed out waiting for current routing to %s with propagation completed; last seen: %s; last error: %v: %w",
				canonicalDeploymentVersion(config.DeploymentName, expectedBuildID),
				lastSeen,
				lastErr,
				waitCtx.Err())
		}
	}
}

func acquire(ctx context.Context, sem chan struct{}) error {
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release(sem chan struct{}) {
	<-sem
}

func deploymentVersionMatches(version *deploymentpb.WorkerDeploymentVersion, deploymentName string, buildID string) bool {
	return version != nil &&
		version.GetDeploymentName() == deploymentName &&
		version.GetBuildId() == buildID
}

func deploymentVersionString(version *deploymentpb.WorkerDeploymentVersion) string {
	if version == nil {
		return "<nil>"
	}
	return canonicalDeploymentVersion(version.GetDeploymentName(), version.GetBuildId())
}

func canonicalDeploymentVersion(deploymentName string, buildID string) string {
	return deploymentName + "." + buildID
}
