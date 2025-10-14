package scenarios

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/converter"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	ScheduleCountFlag              = "schedule-count"
	ScheduleActionsPerScheduleFlag = "schedule-actions-per-schedule"
	ScheduleCronFlag               = "schedule-cron"
	ScheduleCompletionTimeoutFlag  = "schedule-completion-timeout"
)

// Note: VisibilityVerificationTimeoutFlag is defined in throughput_stress.go

type scheduleStressConfig struct {
	ScheduleCount                 int
	ActionsPerSchedule            int
	CronSchedule                  string
	ScenarioRunID                 string
	ScheduleCompletionTimeout     time.Duration
	VisibilityVerificationTimeout time.Duration
}

type scheduleStressExecutor struct {
	mu               sync.Mutex
	config           *scheduleStressConfig
	schedulesCreated []string
}

var _ loadgen.Configurable = (*scheduleStressExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Schedule stress scenario. Creates schedules and verifies they execute workflows at the expected rate.\n"+
				"Options:\n"+
				"  %s (default: 10) - Number of schedules to create\n"+
				"  %s (default: 5) - Number of workflow executions per schedule\n"+
				"  %s (default: '* * * * * * *') - Cron schedule for workflow execution\n"+
				"  %s (default: 5m) - Timeout waiting for schedules to complete\n"+
				"  %s (default: 3m) - Timeout for visibility verification",
			ScheduleCountFlag, ScheduleActionsPerScheduleFlag, ScheduleCronFlag,
			ScheduleCompletionTimeoutFlag, VisibilityVerificationTimeoutFlag),
		ExecutorFn: func() loadgen.Executor { return newScheduleStressExecutor() },
	})
}

func newScheduleStressExecutor() *scheduleStressExecutor {
	return &scheduleStressExecutor{}
}

func (e *scheduleStressExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &scheduleStressConfig{
		ScheduleCount:                 info.ScenarioOptionInt(ScheduleCountFlag, 10),
		ActionsPerSchedule:            info.ScenarioOptionInt(ScheduleActionsPerScheduleFlag, 5),
		CronSchedule:                  info.ScenarioOptions[ScheduleCronFlag],
		ScenarioRunID:                 info.RunID,
		ScheduleCompletionTimeout:     info.ScenarioOptionDuration(ScheduleCompletionTimeoutFlag, 5*time.Minute),
		VisibilityVerificationTimeout: info.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 3*time.Minute),
	}

	if config.ScheduleCount <= 0 {
		return fmt.Errorf("%s must be positive, got %d", ScheduleCountFlag, config.ScheduleCount)
	}

	if config.ActionsPerSchedule <= 0 {
		return fmt.Errorf("%s must be positive, got %d", ScheduleActionsPerScheduleFlag, config.ActionsPerSchedule)
	}

	if config.CronSchedule == "" {
		// Default to every second
		config.CronSchedule = "* * * * * * *"
	}

	if config.ScheduleCompletionTimeout <= 0 {
		return fmt.Errorf("%s must be positive, got %v", ScheduleCompletionTimeoutFlag, config.ScheduleCompletionTimeout)
	}

	if config.VisibilityVerificationTimeout <= 0 {
		return fmt.Errorf("%s must be positive, got %v", VisibilityVerificationTimeoutFlag, config.VisibilityVerificationTimeout)
	}

	e.config = config
	return nil
}

// Run executes the schedule stress scenario.
//
// It creates N schedules (configured by schedule-count), each configured to trigger M workflow
// executions (configured by schedule-actions-per-schedule) on a cron schedule.
// After all schedules complete, it verifies the expected number of workflows were executed via visibility.
func (e *scheduleStressExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}

	// Each iteration creates a schedule that will trigger the configured number of workflow executions
	info.Logger.Info(fmt.Sprintf("Creating %d schedules with %d actions each",
		e.config.ScheduleCount, e.config.ActionsPerSchedule))

	// Cleanup all schedules on exit
	defer func() {
		e.mu.Lock()
		schedules := make([]string, len(e.schedulesCreated))
		copy(schedules, e.schedulesCreated)
		e.mu.Unlock()

		if len(schedules) == 0 {
			return
		}

		info.Logger.Info(fmt.Sprintf("Cleaning up %d schedules", len(schedules)))
		for _, scheduleID := range schedules {
			handle := info.Client.ScheduleClient().GetHandle(context.Background(), scheduleID)
			if err := handle.Delete(context.Background()); err != nil {
				info.Logger.Error(fmt.Sprintf("Failed to delete schedule %s: %v", scheduleID, err))
			} else {
				info.Logger.Info(fmt.Sprintf("Deleted schedule %s", scheduleID))
			}
		}
	}()

	ksExec := &loadgen.KitchenSinkExecutor{
		TestInput: &kitchensink.TestInput{
			WorkflowInput: &kitchensink.WorkflowInput{
				InitialActions: []*kitchensink.ActionSet{},
			},
		},
		UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
			// Each iteration creates a schedule
			scheduleID := fmt.Sprintf("schedule-%s-%d", run.RunID, run.Iteration)
			e.mu.Lock()
			e.schedulesCreated = append(e.schedulesCreated, scheduleID)
			e.mu.Unlock()

			// Create workflow input for scheduled workflows with a return action that returns a non-nil payload
			// This is necessary because the kitchen sink workflow enters an infinite signal loop
			// if initial actions return (nil, nil)
			emptyPayload := &common.Payload{
				Data: []byte("{}"),
			}
			scheduledWorkflowInput := &kitchensink.WorkflowInput{
				InitialActions: []*kitchensink.ActionSet{
					{
						Actions: []*kitchensink.Action{
							{
								Variant: &kitchensink.Action_ReturnResult{
									ReturnResult: &kitchensink.ReturnResultAction{
										ReturnThis: emptyPayload,
									},
								},
							},
						},
					},
				},
			}

			// Encode the workflow input as a payload using the default data converter
			dc := converter.GetDefaultDataConverter()
			scheduledWorkflowPayload, err := dc.ToPayload(scheduledWorkflowInput)
			if err != nil {
				return fmt.Errorf("failed to encode scheduled workflow input: %w", err)
			}

			// The workflow will execute a CreateScheduleActivity to create the schedule
			options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
				{
					Actions: []*kitchensink.Action{
						{
							Variant: &kitchensink.Action_CreateSchedule{
								CreateSchedule: &kitchensink.CreateScheduleAction{
									ScheduleId: scheduleID,
									Spec: &kitchensink.ScheduleSpec{
										CronExpressions: []string{e.config.CronSchedule},
										Jitter:          durationpb.New(150 * time.Millisecond),
									},
									Action: &kitchensink.ScheduleAction{
										WorkflowId:               fmt.Sprintf("scheduled-workflow-%s-%d", run.RunID, run.Iteration),
										WorkflowType:             "kitchenSink",
										TaskQueue:                run.TaskQueue(),
										Arguments:                []*common.Payload{scheduledWorkflowPayload},
										WorkflowExecutionTimeout: durationpb.New(30 * time.Second),
										WorkflowTaskTimeout:      durationpb.New(10 * time.Second),
									},
									Policies: &kitchensink.SchedulePolicies{
										RemainingActions:   int64(e.config.ActionsPerSchedule),
										TriggerImmediately: false,
										CatchupWindow:      durationpb.New(60 * time.Second),
									},
								},
							},
						},
						{
							Variant: &kitchensink.Action_ReturnResult{
								ReturnResult: &kitchensink.ReturnResultAction{
									ReturnThis: emptyPayload,
								},
							},
						},
					},
					Concurrent: false,
				},
			}

			return nil
		},
	}

	// Run the executor to create all the schedules
	info.Logger.Info("Starting kitchen sink executor to create schedules")
	if err := ksExec.Run(ctx, info); err != nil {
		return fmt.Errorf("failed to create schedules: %w", err)
	}

	e.mu.Lock()
	schedules := make([]string, len(e.schedulesCreated))
	copy(schedules, e.schedulesCreated)
	e.mu.Unlock()

	info.Logger.Info(fmt.Sprintf("Kitchen sink executor finished. Created %d schedules, waiting for them to complete", len(schedules)))

	// Wait for all schedules to complete concurrently using errgroup
	g, gctx := errgroup.WithContext(ctx)

	for _, scheduleID := range schedules {
		scheduleID := scheduleID // capture loop variable
		g.Go(func() error {
			info.Logger.Infof("Waiting for schedule %s to complete", scheduleID)
			if err := e.waitForScheduleCompletion(gctx, info, scheduleID); err != nil {
				return fmt.Errorf("failed waiting for schedule %s to complete: %w", scheduleID, err)
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	info.Logger.Info("All schedules completed")

	// Verify that the expected number of workflows were executed
	expectedWorkflows := e.config.ScheduleCount + (e.config.ScheduleCount * e.config.ActionsPerSchedule)
	info.Logger.Info(fmt.Sprintf("Verifying %d workflows were executed", expectedWorkflows))

	// Post-scenario: verify reported workflow completion count from Visibility
	if err := loadgen.MinVisibilityCountEventually(
		ctx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query:     fmt.Sprintf("WorkflowType='kitchenSink' AND TaskQueue='%s'", loadgen.TaskQueueForRun(info.RunID)),
		},
		expectedWorkflows,
		e.config.VisibilityVerificationTimeout,
	); err != nil {
		return fmt.Errorf("visibility verification failed: %w", err)
	}

	info.Logger.Info("Schedule stress scenario completed successfully")
	return nil
}

// waitForScheduleCompletion polls a schedule until it has completed all actions
func (e *scheduleStressExecutor) waitForScheduleCompletion(ctx context.Context, info loadgen.ScenarioInfo, scheduleID string) error {
	ctx, cancel := context.WithTimeout(ctx, e.config.ScheduleCompletionTimeout)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for schedule %s to complete: %w", scheduleID, ctx.Err())
		case <-ticker.C:
			handle := info.Client.ScheduleClient().GetHandle(ctx, scheduleID)
			desc, err := handle.Describe(ctx)
			if err != nil {
				return fmt.Errorf("failed to describe schedule %s: %w", scheduleID, err)
			}

			// Schedule is complete when RemainingActions is 0 and no workflows are running
			if desc.Schedule.State.RemainingActions == 0 && len(desc.Info.RunningWorkflows) == 0 {
				return nil
			}
		}
	}
}
