package scenarios

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf("Stress test Temporal's scheduler functionality by creating, reading, updating, and deleting multiple schedules concurrently. "+
			"Available parameters: '%s' (default: %d), '%s' (default: %d), '%s' (default: %d), '%s' (default: %v), '%s' (default: %d), '%s' (default: %v), '%s' (default: %v), "+
			"'%s' (default: '%s'), '%s' (default: '%s', options: skip,buffer_one,buffer_all,cancel_other,terminate_other,all), "+
			"'%s' (default: '%s', options: %s,%s), '%s' (default: %v), '%s' (default: %v), '%s' (default: %v), '%s' (default: %d)",
			ScheduleCreationPerIterationFlag, DefaultScheduleCreationPerIteration,
			ScheduleReadsPerCreationFlag, DefaultScheduleReadsPerCreation,
			ScheduleUpdatesPerCreationFlag, DefaultScheduleUpdatesPerCreation,
			SchedulerDurationPerIterationFlag, DefaultSchedulerDurationPerIteration,
			PayloadSizeFlag, DefaultPayloadSize,
			WaitTimeBeforeCleanupFlag, DefaultWaitTimeBeforeCleanup,
			OperationIntervalFlag, DefaultOperationInterval,
			CronExpressionFlag, DefaultCronExpression,
			OverlapPolicyFlag, DefaultOverlapPolicy,
			ScheduledWorkflowTypeFlag, DefaultScheduledWorkflowType, NoopScheduledWorkflowType, SleepScheduleWorkflowType,
			EnableChasmSchedulerFlag, DefaultEnableChasmScheduler,
			SkipDeletionFlag, DefaultSkipDeletion,
			SkipCreationFlag, DefaultSkipCreation,
			WorkerCountFlag, DefaultWorkerCount),
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

type CleanUpScheduleInput struct {
	ScheduleID  string
	DeleteAfter time.Duration
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
	EnableChasmScheduler          bool
	SkipDeletion                  bool
	SkipCreation                  bool
	WorkerCount                   int
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
	EnableChasmSchedulerFlag          = "enable-chasm-scheduler"
	SkipDeletionFlag                  = "skip-deletion"
	SkipCreationFlag                  = "skip-creation"
	WorkerCountFlag                   = "worker-count"
)

const (
	DefaultScheduleCreationPerIteration = 10
	DefaultScheduleReadsPerCreation     = 3
	DefaultScheduleUpdatesPerCreation   = 3
	DefaultPayloadSize                  = 1024
	DefaultCronExpression               = "* * * * * *"
	DefaultScheduledWorkflowType        = NoopScheduledWorkflowType
	DefaultOverlapPolicy                = "all"
	DefaultEnableChasmScheduler         = true
	DefaultSkipDeletion                 = false
	DefaultSkipCreation                 = false
	DefaultWorkerCount                  = 1000
)

// Default duration constants
var (
	DefaultSchedulerDurationPerIteration = 30 * time.Second
	DefaultWaitTimeBeforeCleanup         = 25 * time.Second
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
		EnableChasmScheduler:          info.ScenarioOptionBool(EnableChasmSchedulerFlag, DefaultEnableChasmScheduler),
		SkipDeletion:                  info.ScenarioOptionBool(SkipDeletionFlag, DefaultSkipDeletion),
		SkipCreation:                  info.ScenarioOptionBool(SkipCreationFlag, DefaultSkipCreation),
		WorkerCount:                   info.ScenarioOptionInt(WorkerCountFlag, DefaultWorkerCount),
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
	if config.WorkerCount <= 0 {
		return fmt.Errorf("worker-count must be positive, got %d", config.WorkerCount)
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

	// Enable chasm-scheduler experiment if configured
	if s.config.EnableChasmScheduler {
		ctx = metadata.AppendToOutgoingContext(ctx, "temporal-experiment", "chasm-scheduler")
	}

	if s.config.SkipCreation {
		return s.executeWithExistingSchedules(ctx, client, logger)
	}

	return s.executeWithNewSchedules(ctx, run, client, logger)
}

type scheduleConfig struct {
	ScheduleID string
	TaskQueue  string
}

func (s *SchedulerExecutor) executeWithNewSchedules(ctx context.Context, run *loadgen.Run, client client.Client, logger *zap.SugaredLogger) error {
	// Create a channel to send schedule configs
	scheduleConfigChan := make(chan scheduleConfig, 1000)

	// Start a goroutine to produce schedule configs
	go func() {
		defer close(scheduleConfigChan)
		for i := range s.config.ScheduleCreationPerIteration {
			sch_id := fmt.Sprintf("sched-%s-%d-%d-%s", run.RunID, run.Iteration, i, uuid.New())
			select {
			case scheduleConfigChan <- scheduleConfig{
				ScheduleID: sch_id,
				TaskQueue:  run.TaskQueue(),
			}:
			case <-ctx.Done():
				logger.Infow("Context canceled while producing schedule configs")
				return
			}
		}
		logger.Infow("Finished producing schedule configs")
	}()

	// Start workers that will create and process schedules as configs arrive
	var wg sync.WaitGroup

	for range s.config.WorkerCount {
		wg.Go(func() {
			// Each worker keeps consuming from the channel until it's closed
			for config := range scheduleConfigChan {
				func() {

					start := time.Now()
					ticker := time.NewTicker(s.config.OperationInterval)

					sc, err := s.createSchedule(ctx, client, config.ScheduleID, config.TaskQueue, logger)
					if err != nil {
						logger.Errorw("Failed to create schedule", "scheduleID", config.ScheduleID, "error", err)
						ticker.Stop()
						return
					}

					if !s.config.SkipDeletion {
						defer func(id string, startTime time.Time) {
							dur := time.Until(startTime.Add(s.config.WaitTimeBeforeCleanup))
							select {
							case <-time.After(dur):
								if err := s.deleteSchedule(ctx, client, id, logger); err != nil {
									logger.Errorw("Failed to delete schedule", "scheduleID", id, "error", err)
								}
							case <-ctx.Done():
								logger.Infow("Context canceled")
							}
						}(sc.ScheduleID, start)
					}

					<-ticker.C
					s.performScheduleOperations(ctx, client, sc.ScheduleID, ticker, logger)
					ticker.Stop()
				}()
			}
		})
	}

	// Wait for all workers to complete (they'll finish when channel closes)
	wg.Wait()
	logger.Infow("all workers closed")

	return nil
}

func (s *SchedulerExecutor) executeWithExistingSchedules(ctx context.Context, client client.Client, logger *zap.SugaredLogger) error {
	// Create a channel to receive schedule IDs
	scheduleIDChan := make(chan string, 100)
	listErrChan := make(chan error, 1)

	// Start a goroutine to list schedules and send them to the channel
	go func() {
		defer close(scheduleIDChan)
		listErrChan <- s.listSchedules(ctx, client, scheduleIDChan, logger)
		logger.Infow("about to close ch")
	}()

	// Start workers that will process schedules as they arrive
	var wg sync.WaitGroup

	for range s.config.WorkerCount {
		wg.Go(func() {
			// Each worker keeps consuming from the channel until it's closed
			for scheduleID := range scheduleIDChan {
				func() {

					start := time.Now()
					ticker := time.NewTicker(s.config.OperationInterval)

					if !s.config.SkipDeletion {
						defer func(id string, startTime time.Time) {
							dur := time.Until(startTime.Add(s.config.WaitTimeBeforeCleanup))
							select {
							case <-time.After(dur):
								if err := s.deleteSchedule(ctx, client, id, logger); err != nil {
									logger.Errorw("Failed to delete schedule", "scheduleID", id, "error", err)
								}
							case <-ctx.Done():
								logger.Infow("Context canceled")
							}
						}(scheduleID, start)
					}

					<-ticker.C
					s.performScheduleOperations(ctx, client, scheduleID, ticker, logger)
					ticker.Stop()
				}()
			}
		})
	}

	// Wait for all workers to complete (they'll finish when channel closes)
	wg.Wait()
	logger.Infow("all workers closed")

	// Check if there was an error during listing
	listErr := <-listErrChan
	if listErr != nil {
		return fmt.Errorf("failed to list schedules: %w", listErr)
	}

	return nil
}

func (s *SchedulerExecutor) performScheduleOperations(ctx context.Context, client client.Client, scheduleID string, ticker *time.Ticker, logger *zap.SugaredLogger) {
	for range s.config.ScheduleReadsPerCreation {
		<-ticker.C // Wait between read operations
		if err := s.describeSchedule(ctx, client, scheduleID, logger); err != nil {
			logger.Errorw("Failed to describe schedule", "scheduleID", scheduleID, "error", err)
		}
		<-ticker.C // Wait between read operations
	}

	for range s.config.ScheduleUpdatesPerCreation {
		if err := s.updateSchedule(ctx, client, scheduleID, logger); err != nil {
			logger.Errorw("Failed to update schedule", "scheduleID", scheduleID, "error", err)
		}
		<-ticker.C // Wait between update operations
	}
}

type ScheduleState struct {
	ScheduleID  string
	DeleteAfter time.Duration
}

func (s *SchedulerExecutor) listSchedules(ctx context.Context, c client.Client, scheduleIDChan chan<- string, logger *zap.SugaredLogger) error {
	iter, err := c.ScheduleClient().List(ctx, client.ScheduleListOptions{})
	if err != nil {
		logger.Errorw("Error creating schedule list iterator", "error", err)
		return fmt.Errorf("error creating schedule list iterator: %w", err)
	}

	count := 0
	for iter.HasNext() {
		entry, err := iter.Next()
		if err != nil {
			logger.Errorw("Error iterating schedules", "error", err, "count", count)
			return fmt.Errorf("error iterating schedules: %w", err)
		}
		select {
		case scheduleIDChan <- entry.ID:
			count++
			if count%100 == 0 {
				logger.Debugw("Listing schedules in progress", "count", count)
			}
		case <-ctx.Done():
			logger.Infow("Context canceled while listing schedules", "count", count)
			return ctx.Err()
		}
	}
	logger.Infow("Finished listing schedules", "totalCount", count)
	return nil
}

func (s *SchedulerExecutor) createSchedule(ctx context.Context, c client.Client, scheduleID string, taskQueue string, logger *zap.SugaredLogger) (ScheduleState, error) {
	sc := ScheduleState{
		ScheduleID:  scheduleID,
		DeleteAfter: s.config.SchedulerDurationPerIteration,
	}
	workflowID := fmt.Sprintf("w-%s", scheduleID)
	action := &client.ScheduleWorkflowAction{
		ID:        workflowID,
		Workflow:  s.config.ScheduledWorkflowType,
		Args:      []any{make([]byte, s.config.PayloadSize)},
		TaskQueue: taskQueue,
	}

	dur := (time.Duration(int64(int64(s.config.ScheduleReadsPerCreation)+int64(s.config.ScheduleUpdatesPerCreation))) *
		s.config.OperationInterval) +
		(2 * time.Second)

	//Add some time to give the executor enough time to delete the schedule
	endTime := time.Now().Add(sc.DeleteAfter).Add(dur)

	options := client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{s.config.CronExpression},
			// defining an end at ensures all schedules will be removed
			// regardless of errors from this executor
			EndAt:   endTime,
			StartAt: time.Now(),
		},
		TriggerImmediately: true,
		Action:             action,
		Overlap:            pickOverlap(s.config.OverlapPolicy, logger),
	}

	err := retryWithBackoff(ctx, "createSchedule", logger, func() error {
		// Create new context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		_, err := c.ScheduleClient().Create(opCtx, options)
		return err
	})
	return sc, err
}

func pickOverlap(policies []enums.ScheduleOverlapPolicy, logger *zap.SugaredLogger) enums.ScheduleOverlapPolicy {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(policies))))
	if err != nil {
		logger.Errorw("Failed to select overlap policy", "error", err)
		return policies[0]
	}
	return policies[n.Int64()]
}

// retryConfig defines retry behavior for schedule operations
const (
	maxRetries      = 5
	baseBackoff     = 100 * time.Millisecond
	maxBackoff      = 10 * time.Second
	backoffMultiple = 2.0
)

// retryWithBackoff executes an operation with exponential backoff and jitter on context deadline errors
func retryWithBackoff(ctx context.Context, operation string, logger *zap.SugaredLogger, fn func() error) error {
	var lastErr error
	backoff := baseBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Check if parent context is cancelled before attempting
		if ctx.Err() != nil {
			logger.Debugw("Parent context cancelled, stopping retries", "operation", operation, "attempt", attempt)
			return ctx.Err()
		}

		// Execute the operation
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		// Check if this is a deadline exceeded error (from SDK or context)
		var deadlineErr *serviceerror.DeadlineExceeded
		isDeadlineError := errors.As(lastErr, &deadlineErr) || errors.Is(lastErr, context.DeadlineExceeded)

		// Log error details for debugging
		if deadlineErr != nil {
			logger.Debugw("Operation failed",
				"operation", operation,
				"attempt", attempt+1,
				"err", lastErr.Error(),
				"message", deadlineErr.Message,
				"status", fmt.Sprintf("%v", deadlineErr.Status()),
				"code", deadlineErr.Status().Code())
		} else {
			logger.Debugw("Operation failed",
				"operation", operation,
				"attempt", attempt+1,
				"errorType", fmt.Sprintf("%T", lastErr),
				"errorMsg", lastErr.Error())
		}

		if !isDeadlineError {
			logger.Debugw("Not a deadline error, not retrying")
			return lastErr
		}

		// Don't retry if we've exhausted attempts
		if attempt == maxRetries-1 {
			logger.Warnw("Max retries reached", "operation", operation, "attempts", maxRetries, "error", lastErr)
			return fmt.Errorf("max retries (%d) exceeded for %s: %w", maxRetries, operation, lastErr)
		}

		// Calculate backoff with jitter
		jitterMax := big.NewInt(int64(backoff / 2))
		jitterVal, err := rand.Int(rand.Reader, jitterMax)
		if err != nil {
			jitterVal = big.NewInt(0)
		}
		sleepDuration := backoff + time.Duration(jitterVal.Int64())

		logger.Debugw("Retrying after backoff",
			"operation", operation,
			"attempt", attempt+1,
			"backoff", sleepDuration,
			"error", lastErr)

		// Sleep with backoff, but respect parent context cancellation
		select {
		case <-time.After(sleepDuration):
			// Continue to next attempt
		case <-ctx.Done():
			logger.Debugw("Parent context cancelled during backoff", "operation", operation)
			return ctx.Err()
		}

		// Increase backoff exponentially, capped at maxBackoff
		backoff = time.Duration(float64(backoff) * backoffMultiple)
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	return lastErr
}

func (s *SchedulerExecutor) describeSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	return retryWithBackoff(ctx, "describeSchedule", logger, func() error {
		// Create new context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		handle := c.ScheduleClient().GetHandle(opCtx, scheduleID)
		_, err := handle.Describe(opCtx)
		if err != nil {
			var notFoundErr *serviceerror.NotFound
			if errors.As(err, &notFoundErr) {
				// Return nil if schedule is not found (already deleted or never existed)
				logger.Debugw("Schedule not found during describe operation", "scheduleID", scheduleID)
				return nil
			}
		}
		return err
	})
}

func (s *SchedulerExecutor) updateSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
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

	return retryWithBackoff(ctx, "updateSchedule", logger, func() error {
		// Create new context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		handle := c.ScheduleClient().GetHandle(opCtx, scheduleID)
		err := handle.Update(opCtx, client.ScheduleUpdateOptions{
			DoUpdate: updateFn,
		})
		if err != nil {
			var notFoundErr *serviceerror.NotFound
			if errors.As(err, &notFoundErr) {
				// Return nil if schedule is not found (already deleted or never existed)
				logger.Debugw("Schedule not found during update operation", "scheduleID", scheduleID)
				return nil
			}
		}
		return err
	})
}

func (s *SchedulerExecutor) deleteSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	return retryWithBackoff(ctx, "deleteSchedule", logger, func() error {
		// Create new context with timeout for this operation
		opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		handle := c.ScheduleClient().GetHandle(opCtx, scheduleID)
		err := handle.Delete(opCtx)
		if err != nil {
			var notFoundErr *serviceerror.NotFound
			if errors.As(err, &notFoundErr) {
				// Return nil if schedule is not found (already deleted or never existed)
				logger.Debugw("Schedule not found during delete operation", "scheduleID", scheduleID)
				return nil
			}
		}
		return err
	})
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
		return []enums.ScheduleOverlapPolicy{enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL}
	}
	return l
}
