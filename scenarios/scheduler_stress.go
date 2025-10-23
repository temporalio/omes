package scenarios

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf("Stress test Temporal's scheduler functionality by creating, reading, updating, and deleting multiple schedules concurrently. "+
			"Available parameters: '%s' (default: %d), '%s' (default: %d), '%s' (default: %d), '%s' (default: %v), '%s' (default: %d), '%s' (default: %v), '%s' (default: %v), "+
			"'%s' (default: '%s'), '%s' (default: '%s', options: skip,buffer_one,buffer_all,cancel_other,terminate_other,all), "+
			"'%s' (default: '%s', options: %s,%s)",
			ScheduleCreationPerIterationFlag, DefaultScheduleCreationPerIteration,
			ScheduleReadsPerCreationFlag, DefaultScheduleReadsPerCreation,
			ScheduleUpdatesPerCreationFlag, DefaultScheduleUpdatesPerCreation,
			SchedulerDurationPerIterationFlag, DefaultSchedulerDurationPerIteration,
			PayloadSizeFlag, DefaultPayloadSize,
			WaitTimeBeforeCleanupFlag, DefaultWaitTimeBeforeCleanup,
			OperationIntervalFlag, DefaultOperationInterval,
			CronExpressionFlag, DefaultCronExpression,
			OverlapPolicyFlag, DefaultOverlapPolicy,
			ScheduledWorkflowTypeFlag, DefaultScheduledWorkflowType, NoopScheduledWorkflowType, SleepScheduleWorkflowType),
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, run *loadgen.Run) error {
					executor := SchedulerExecutor{}
					return executor.Execute(ctx, run)
				},
			}
		},
	})
}

// Types for orchestration workflow
type SchedulerOrchestrationInput struct {
	RunID                 string
	Iteration             int
	WaitTimeBeforeCleanup time.Duration
}

type SchedulerOrchestrationResult struct {
	SchedulesDeleted int
	TotalDuration    time.Duration
}

type ScheduleIDsSignalData struct {
	ScheduleIDs []string
	StartTime   time.Time
}

type schedulerExecutorConfig struct {
	ScheduleCreationPerIteration  int
	ScheduleReadsPerCreation      int
	ScheduleUpdatesPerCreation    int
	SchedulerDurationPerIteration time.Duration
	PayloadSize                   int
	WaitTimeBeforeCleanup         time.Duration
	OperationInterval             time.Duration
	CronExpression                string
	OverlapPolicy                 []enums.ScheduleOverlapPolicy
	ScheduledWorkflowType         string
}

var _ loadgen.Configurable = (*SchedulerExecutor)(nil)

type SchedulerExecutor struct {
	config *schedulerExecutorConfig
}

const (
	SleepScheduleWorkflowType = "SleepScheduledWorkflow"
	NoopScheduledWorkflowType = "NoopScheduledWorkflow"
)

const (
	// Parameter name constants
	ScheduleCreationPerIterationFlag  = "schedule-creation-per-iteration"
	ScheduleReadsPerCreationFlag      = "schedule-reads-per-creation"
	ScheduleUpdatesPerCreationFlag    = "schedule-updates-per-creation"
	SchedulerDurationPerIterationFlag = "scheduler-duration-per-iteration"
	PayloadSizeFlag                   = "payload-size"
	WaitTimeBeforeCleanupFlag         = "wait-time-before-cleanup"
	OperationIntervalFlag             = "operation-interval"
	CronExpressionFlag                = "cron-expression"
	OverlapPolicyFlag                 = "overlap-policy"
	ScheduledWorkflowTypeFlag         = "scheduled-workflow-type"
)

const (
	DefaultScheduleCreationPerIteration = 5
	DefaultScheduleReadsPerCreation     = 1
	DefaultScheduleUpdatesPerCreation   = 1
	DefaultPayloadSize                  = 1024
	DefaultCronExpression               = "* * * * * *"
	DefaultScheduledWorkflowType        = NoopScheduledWorkflowType
	DefaultOverlapPolicy                = "all"
)

// Default duration constants
var (
	DefaultSchedulerDurationPerIteration = time.Minute
	DefaultWaitTimeBeforeCleanup         = 5 * time.Second
	DefaultOperationInterval             = 50 * time.Millisecond
)

// Configure implements the loadgen.Configurable interface
func (s *SchedulerExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &schedulerExecutorConfig{
		ScheduleCreationPerIteration:  info.ScenarioOptionInt(ScheduleCreationPerIterationFlag, DefaultScheduleCreationPerIteration),
		ScheduleReadsPerCreation:      info.ScenarioOptionInt(ScheduleReadsPerCreationFlag, DefaultScheduleReadsPerCreation),
		ScheduleUpdatesPerCreation:    info.ScenarioOptionInt(ScheduleUpdatesPerCreationFlag, DefaultScheduleUpdatesPerCreation),
		SchedulerDurationPerIteration: info.ScenarioOptionDuration(SchedulerDurationPerIterationFlag, DefaultSchedulerDurationPerIteration),
		PayloadSize:                   info.ScenarioOptionInt(PayloadSizeFlag, DefaultPayloadSize),
		WaitTimeBeforeCleanup:         info.ScenarioOptionDuration(WaitTimeBeforeCleanupFlag, DefaultWaitTimeBeforeCleanup),
		OperationInterval:             info.ScenarioOptionDuration(OperationIntervalFlag, DefaultOperationInterval),
		CronExpression:                info.ScenarioOptionString(CronExpressionFlag, DefaultCronExpression),
		OverlapPolicy:                 parseOverlapPolicy(info.ScenarioOptionString(OverlapPolicyFlag, DefaultOverlapPolicy)),
		ScheduledWorkflowType:         info.ScenarioOptionString(ScheduledWorkflowTypeFlag, DefaultScheduledWorkflowType),
	}
	if config.ScheduleCreationPerIteration <= 0 {
		return fmt.Errorf("schedule-creation-per-iteration must be positive, got %d", config.ScheduleCreationPerIteration)
	}
	if config.ScheduleReadsPerCreation < 0 {
		return fmt.Errorf("schedule-reads-per-creation cannot be negative, got %d", config.ScheduleReadsPerCreation)
	}
	if config.ScheduleUpdatesPerCreation < 0 {
		return fmt.Errorf("schedule-updates-per-creation cannot be negative, got %d", config.ScheduleUpdatesPerCreation)
	}
	if config.PayloadSize <= 0 {
		return fmt.Errorf("payload-size must be positive, got %d", config.PayloadSize)
	}
	if config.SchedulerDurationPerIteration <= 0 {
		return fmt.Errorf("scheduler-duration-per-iteration must be positive, got %v", config.SchedulerDurationPerIteration)
	}
	if config.WaitTimeBeforeCleanup < 0 {
		return fmt.Errorf("wait-time-before-cleanup cannot be negative, got %v", config.WaitTimeBeforeCleanup)
	}
	if config.OperationInterval < 0 {
		return fmt.Errorf("operation-interval cannot be negative, got %v", config.OperationInterval)
	}
	s.config = config
	return nil
}

func (s *SchedulerExecutor) Execute(ctx context.Context, run *loadgen.Run) error {
	if err := s.Configure(*run.ScenarioInfo); err != nil {
		return err
	}

	logger := run.Logger
	client := run.Client
	workflowOptions := run.DefaultStartWorkflowOptions()
	workflowOptions.ID = fmt.Sprintf("scheduler-orchestration-%s-%d", run.RunID, run.Iteration)
	workflowRun, err := run.Client.ExecuteWorkflow(ctx, workflowOptions, "SchedulerOrchestrationWorkflow",
		SchedulerOrchestrationInput{
			RunID:                 run.RunID,
			Iteration:             run.Iteration,
			WaitTimeBeforeCleanup: s.config.WaitTimeBeforeCleanup,
		})
	if err != nil {
		logger.Error("Failed to start orchestration workflow", "error", err)
		return err
	}

	g, gctx := errgroup.WithContext(ctx)
	for i := range s.config.ScheduleCreationPerIteration {
		sc := ScheduleState{
			ScheduleID: fmt.Sprintf("sched-%s-%d-%d", run.RunID, run.Iteration, i),
		}
		g.Go(func() error {
			ticker := time.NewTicker(s.config.OperationInterval)
			defer ticker.Stop()

			scheduleStartTime := time.Now()
			if err := s.createSchedule(gctx, client, sc.ScheduleID, logger); err != nil {
				logger.Error("Failed to create schedule", "scheduleID", sc.ScheduleID, "error", err)
				return err
			}
			<-ticker.C // Wait before next operation

			signalData := ScheduleIDsSignalData{
				ScheduleIDs: []string{sc.ScheduleID},
				StartTime:   scheduleStartTime,
			}

			err := run.Client.SignalWorkflow(gctx, workflowOptions.ID, "", "schedule-ids", signalData)
			if err != nil {
				logger.Error("Failed to signal schedule IDs", "error", err)
				return err
			}
			<-ticker.C // Wait before read operations

			for range s.config.ScheduleReadsPerCreation {
				if err := s.describeSchedule(gctx, client, sc.ScheduleID, logger); err != nil {
					logger.Error("Failed to describe schedule", "scheduleID", sc.ScheduleID, "error", err)
					return err
				}
				<-ticker.C // Wait between read operations
			}

			for range s.config.ScheduleUpdatesPerCreation {
				if err := s.updateSchedule(gctx, client, sc.ScheduleID, logger); err != nil {
					logger.Error("Failed to update schedule", "scheduleID", sc.ScheduleID, "error", err)
					return err
				}
				<-ticker.C // Wait between update operations
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	// Signal the workflow to start deletion
	err = run.Client.SignalWorkflow(ctx, workflowOptions.ID, "", "delete-schedules", nil)
	if err != nil {
		logger.Error("Failed to signal deletion", "error", err)
		return err
	}

	// Wait for workflow completion
	var result SchedulerOrchestrationResult
	err = workflowRun.Get(ctx, &result)
	logger.Info("Scheduler execution completed via orchestration workflow",
		"schedulesDeleted", result.SchedulesDeleted,
		"totalDuration", result.TotalDuration)
	return nil
}

type ScheduleState struct {
	ScheduleID string
}

func (s *SchedulerExecutor) createSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	workflowID := fmt.Sprintf("scheduled-%s", scheduleID)
	taskQueue := fmt.Sprintf("scheduler-test-%s", scheduleID)
	action := &client.ScheduleWorkflowAction{
		ID:        workflowID,
		Workflow:  s.config.ScheduledWorkflowType,
		Args:      []any{make([]byte, s.config.PayloadSize)},
		TaskQueue: taskQueue,
	}

	// Set schedule end time based on scheduler duration per iteration
	endTime := time.Now().Add(s.config.SchedulerDurationPerIteration)

	options := client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{s.config.CronExpression},
			EndAt:           endTime,
		},
		Action:  action,
		Overlap: randSelectOverlapPolicy(s.config.OverlapPolicy, logger),
	}

	logger.Info("Creating schedule with end time", "scheduleID", scheduleID, "endTime", endTime)
	_, err := c.ScheduleClient().Create(ctx, options)
	return err
}

func randSelectOverlapPolicy(policies []enums.ScheduleOverlapPolicy, logger *zap.SugaredLogger) enums.ScheduleOverlapPolicy {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(policies))))
	if err != nil {
		logger.Error("Failed to select overlap policy", "error", err)
		return policies[0]
	}
	return policies[n.Int64()]
}

func (s *SchedulerExecutor) describeSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	startTime := time.Now()
	handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	duration := time.Since(startTime)

	if err != nil {
		logger.Warn("Describe schedule failed", "scheduleID", scheduleID, "duration", duration, "error", err)
		return err
	}

	logger.Debug("Describe schedule succeeded", "scheduleID", scheduleID, "duration", duration)
	return nil
}

func (s *SchedulerExecutor) updateSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	handle := c.ScheduleClient().GetHandle(ctx, scheduleID)

	updateFn := func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
		schedule := input.Description.Schedule

		// Keep same cron but update workflow args
		if action, ok := schedule.Action.(*client.ScheduleWorkflowAction); ok {
			action.Args = []any{make([]byte, s.config.PayloadSize)}
		}

		return &client.ScheduleUpdate{
			Schedule: &schedule,
		}, nil
	}
	return handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: updateFn,
	})
}

func (s *SchedulerExecutor) deleteSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
	return handle.Delete(ctx)
}

// parseOverlapPolicy converts string overlap policy to enum
func parseOverlapPolicy(policyStr string) []enums.ScheduleOverlapPolicy {
	policyStr = strings.ToLower(policyStr)
	l := []enums.ScheduleOverlapPolicy{}
	for p := range strings.SplitSeq(policyStr, ",") {
		p = strings.TrimSpace(p)
		switch p {
		case "skip":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_SKIP)
		case "buffer_one":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE)
		case "buffer_all":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL)
		case "cancel_other":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER)
		case "terminate_other":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER)
		case "all":
			l = append(l, enums.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER)
			return []enums.ScheduleOverlapPolicy{
				enums.SCHEDULE_OVERLAP_POLICY_SKIP,
				enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE,
				enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL,
				enums.SCHEDULE_OVERLAP_POLICY_CANCEL_OTHER,
				enums.SCHEDULE_OVERLAP_POLICY_TERMINATE_OTHER,
			}
		}
	}
	if len(l) == 0 {
		return []enums.ScheduleOverlapPolicy{enums.SCHEDULE_OVERLAP_POLICY_SKIP}
	}
	return l
}
