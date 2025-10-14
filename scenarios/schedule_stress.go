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
	ScheduleCountFlag                       = "schedule-count"
	ScheduleActionsPerScheduleFlag          = "schedule-actions-per-schedule"
	ScheduleCronFlag                        = "schedule-cron"
	ScheduleStressScenarioIdSearchAttribute = "ScheduleStressScenarioId"
)

type scheduleStressConfig struct {
	ScheduleCount      int
	ActionsPerSchedule int
	CronSchedule       string
	ScenarioRunID      string
}

type scheduleStressExecutor struct {
	config *scheduleStressConfig
}

var _ loadgen.Configurable = (*scheduleStressExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Schedule stress scenario. Use --option with '%s', '%s' to control parameters",
			ScheduleCountFlag, ScheduleActionsPerScheduleFlag),
		ExecutorFn: func() loadgen.Executor { return newScheduleStressExecutor() },
	})
}

func newScheduleStressExecutor() *scheduleStressExecutor {
	return &scheduleStressExecutor{}
}

func (e *scheduleStressExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &scheduleStressConfig{
		ScheduleCount:      info.ScenarioOptionInt(ScheduleCountFlag, 10),
		ActionsPerSchedule: info.ScenarioOptionInt(ScheduleActionsPerScheduleFlag, 5),
		CronSchedule:       info.ScenarioOptions[ScheduleCronFlag],
		ScenarioRunID:      info.RunID,
	}

	if config.ScheduleCount <= 0 {
		return fmt.Errorf("schedule-count must be positive, got %d", config.ScheduleCount)
	}

	if config.ActionsPerSchedule <= 0 {
		return fmt.Errorf("schedule-actions-per-schedule must be positive, got %d", config.ActionsPerSchedule)
	}

	if config.CronSchedule == "" {
		// Default to every second
		config.CronSchedule = "* * * * * * *"
	}

	e.config = config
	return nil
}

func (e *scheduleStressExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}

	// Add search attribute for tracking
	if err := loadgen.InitSearchAttribute(ctx, info, ScheduleStressScenarioIdSearchAttribute); err != nil {
		return err
	}

	// Each iteration creates a schedule that will trigger the configured number of workflow executions
	info.Logger.Info(fmt.Sprintf("Creating %d schedules with %d actions each",
		e.config.ScheduleCount, e.config.ActionsPerSchedule))

	// Track created schedules for cleanup and waiting
	var schedulesCreated []string
	var schedulesMu sync.Mutex

	// Cleanup all schedules on exit
	defer func() {
		schedulesMu.Lock()
		schedules := make([]string, len(schedulesCreated))
		copy(schedules, schedulesCreated)
		schedulesMu.Unlock()

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
			scheduleID := loadgen.ScheduleIDForRun(run.RunID, run.Iteration)
			// Thread-safe append to track created schedules
			schedulesMu.Lock()
			schedulesCreated = append(schedulesCreated, scheduleID)
			schedulesMu.Unlock()

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

	schedulesMu.Lock()
	schedules := make([]string, len(schedulesCreated))
	copy(schedules, schedulesCreated)
	schedulesMu.Unlock()

	info.Logger.Info(fmt.Sprintf("Kitchen sink executor finished. Created %d schedules, waiting for them to complete", len(schedules)))

	// Wait for all schedules to complete concurrently using errgroup
	completionTimeout := 5 * time.Minute
	g, gctx := errgroup.WithContext(ctx)

	for _, scheduleID := range schedules {
		scheduleID := scheduleID // capture loop variable
		g.Go(func() error {
			info.Logger.Infof("Waiting for schedule %s to complete", scheduleID)
			if err := loadgen.WaitForScheduleCompletion(gctx, info.Client, scheduleID, completionTimeout); err != nil {
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

	// Give visibility some time to catch up
	visibilityTimeout := 3 * time.Minute
	if err := loadgen.MinVisibilityCountEventually(
		ctx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query:     fmt.Sprintf("WorkflowType='kitchenSink' AND TaskQueue='%s'", loadgen.TaskQueueForRun(info.RunID)),
		},
		expectedWorkflows,
		visibilityTimeout,
	); err != nil {
		return fmt.Errorf("visibility verification failed: %w", err)
	}

	info.Logger.Info("Schedule stress scenario completed successfully")
	return nil
}
