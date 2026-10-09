package scenarios

import (
	"context"
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"

	"github.com/temporalio/omes/loadgen"
)

type workerVersioningInactiveOverrideStormExecutor struct {
	config workerVersioningCurrentOverrideStormConfig
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Starts many KitchenSink workflows pinned to an inactive worker deployment version, " +
			"verifies the version becomes draining, then releases the workflows. " +
			"Options: " + wvcosDeploymentNameFlag + " (required), " +
			wvcosV1BuildIDFlag + " (default v1), " +
			wvcosWorkflowCountFlag + " (default 100), " +
			wvcosStartConcurrencyFlag + " (default 50).",
		ExecutorFn: func() loadgen.Executor {
			return &workerVersioningInactiveOverrideStormExecutor{}
		},
	})
}

func (e *workerVersioningInactiveOverrideStormExecutor) Configure(info loadgen.ScenarioInfo) error {
	config, err := configureWorkerVersioningStorm(info, false)
	if err != nil {
		return err
	}

	e.config = config
	return nil
}

func (e *workerVersioningInactiveOverrideStormExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) (err error) {
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
	helper := &workerVersioningCurrentOverrideStormExecutor{}
	var workflows []workerVersioningStormWorkflow
	cleanupPending := false
	defer func() {
		if !cleanupPending {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), config.ReleaseTimeout)
		defer cancel()
		cleanupErr := helper.releaseWorkflows(cleanupCtx, info, config, workflows)
		if cleanupErr == nil {
			return
		}
		if err == nil {
			err = fmt.Errorf("failed to cleanup pinned workflows: %w", cleanupErr)
			return
		}
		info.Logger.Warnf("failed to cleanup pinned workflows after scenario error: %v", cleanupErr)
	}()

	version := canonicalDeploymentVersion(config.DeploymentName, config.V1BuildID)
	info.Logger.Infof("Waiting for inactive worker deployment version %s to register", version)
	if _, err := helper.waitForVersionRegistered(ctx, info, config, config.V1BuildID); err != nil {
		return err
	}
	if _, err := helper.waitForVersionStatus(ctx, info, config, config.V1BuildID, enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_INACTIVE); err != nil {
		return fmt.Errorf("failed waiting for %s to be inactive before starting pinned workflows: %w", version, err)
	}

	info.Logger.Infof("Starting %d workflows pinned to inactive %s", config.WorkflowCount, version)
	workflows, err = helper.startPinnedWorkflows(ctx, info, config)
	if len(workflows) > 0 {
		cleanupPending = true
	}
	if err != nil {
		return fmt.Errorf("failed starting pinned workflows on inactive version: %w", err)
	}

	info.Logger.Infof("Waiting for %s to become draining after pinned override storm", version)
	if _, err := helper.waitForVersionStatus(ctx, info, config, config.V1BuildID, enumspb.WORKER_DEPLOYMENT_VERSION_STATUS_DRAINING); err != nil {
		return fmt.Errorf("failed waiting for inactive version to become draining after pinned override storm: %w", err)
	}

	info.Logger.Infof("Releasing %d pinned workflows", len(workflows))
	if err := helper.releaseWorkflows(ctx, info, config, workflows); err != nil {
		return fmt.Errorf("failed releasing pinned workflows: %w", err)
	}
	cleanupPending = false

	info.Logger.Infof("Worker versioning inactive override storm completed successfully")
	return nil
}
