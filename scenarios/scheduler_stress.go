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
		Description: "Stress test Temporal's scheduler functionality by creating, reading, updating, and deleting multiple schedules concurrently.",
		Options: func(o *loadgen.OptionSet) {
			o.Int(ScheduleCreationPerIterationFlag, DefaultScheduleCreationPerIteration, "Schedules created per iteration.")
			o.Int(ScheduleReadsPerCreationFlag, DefaultScheduleReadsPerCreation, "Schedule reads per created schedule.")
			o.Int(ScheduleUpdatesPerCreationFlag, DefaultScheduleUpdatesPerCreation, "Schedule updates per created schedule.")
			o.Duration(SchedulerDurationPerIterationFlag, DefaultSchedulerDurationPerIteration, "How long each iteration exercises its schedules.")
			o.Int(PayloadSizeFlag, DefaultPayloadSize, "Payload size in bytes for scheduled workflows.")
			o.Duration(WaitTimeBeforeCleanupFlag, DefaultWaitTimeBeforeCleanup, "Wait before deleting schedules.")
			o.Duration(OperationIntervalFlag, DefaultOperationInterval, "Interval between schedule operations.")
			o.String(CronExpressionFlag, DefaultCronExpression, "Cron expression for created schedules.")
			o.String(OverlapPolicyFlag, DefaultOverlapPolicy, "Overlap policy: skip, buffer_one, buffer_all, cancel_other, terminate_other, all.")
			o.String(ScheduledWorkflowTypeFlag, DefaultScheduledWorkflowType,
				"Workflow type to schedule: "+NoopScheduledWorkflowType+" or "+SleepScheduleWorkflowType+".")
			o.Bool(EnableChasmSchedulerFlag, DefaultEnableChasmScheduler, "Route schedule creation to the CHASM scheduler.")
		},
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
)

const (
	DefaultScheduleCreationPerIteration = 10
	DefaultScheduleReadsPerCreation     = 3
	DefaultScheduleUpdatesPerCreation   = 3
	DefaultPayloadSize                  = 1024
	DefaultCronExpression               = "*/5 * * * * *"
	DefaultScheduledWorkflowType        = NoopScheduledWorkflowType
	DefaultOverlapPolicy                = "all"
	DefaultEnableChasmScheduler         = true
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
		ScheduleCreationPerIteration:  info.OptionInt(ScheduleCreationPerIterationFlag),
		ScheduleReadsPerCreation:      info.OptionInt(ScheduleReadsPerCreationFlag),
		ScheduleUpdatesPerCreation:    info.OptionInt(ScheduleUpdatesPerCreationFlag),
		SchedulerDurationPerIteration: info.OptionDuration(SchedulerDurationPerIterationFlag),
		PayloadSize:                   info.OptionInt(PayloadSizeFlag),
		WaitTimeBeforeCleanup:         info.OptionDuration(WaitTimeBeforeCleanupFlag),
		OperationInterval:             info.OptionDuration(OperationIntervalFlag),
		CronExpression:                info.OptionString(CronExpressionFlag),
		OverlapPolicy:                 parseOverlapPolicy(info.OptionString(OverlapPolicyFlag)),
		ScheduledWorkflowType:         info.OptionString(ScheduledWorkflowTypeFlag),
		EnableChasmScheduler:          info.OptionBool(EnableChasmSchedulerFlag),
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

	// Enable chasm-scheduler experiment if configured
	if s.config.EnableChasmScheduler {
		ctx = metadata.AppendToOutgoingContext(ctx, "temporal-experiment", "chasm-scheduler")
	}

	// How many failed iterations a run tolerates is the caller's policy, via
	// RunConfiguration.ContinueOnIterationFailure.
	var (
		mu   sync.Mutex
		errs []error
	)
	record := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, err)
	}

	var wg sync.WaitGroup
	for i := range s.config.ScheduleCreationPerIteration {
		wg.Go(func() {
			ticker := time.NewTicker(s.config.OperationInterval)
			defer ticker.Stop()
			start := time.Now()

			scheduleID := fmt.Sprintf("sched-%s-%d-%d-%s", run.RunID, run.Iteration, i, uuid.New())
			sc, err := s.createSchedule(ctx, client, scheduleID, run.TaskQueue(), logger)
			if err != nil {
				record(fmt.Errorf("creating schedule %s: %w", scheduleID, err))
				return
			}
			<-ticker.C
			// Read and update failures fall through to the delete below; an
			// abandoned schedule would outlive the iteration until its EndAt.
			for range s.config.ScheduleReadsPerCreation {
				if err := s.describeSchedule(ctx, client, sc.ScheduleID, logger); err != nil {
					record(fmt.Errorf("describing schedule %s: %w", sc.ScheduleID, err))
				}
				<-ticker.C // Wait between read operations
			}

			for range s.config.ScheduleUpdatesPerCreation {
				if err := s.updateSchedule(ctx, client, sc.ScheduleID, logger); err != nil {
					record(fmt.Errorf("updating schedule %s: %w", sc.ScheduleID, err))
				}
				<-ticker.C // Wait between update operations
			}
			dur := time.Until(start.Add(s.config.WaitTimeBeforeCleanup))
			select {
			case <-time.After(dur):
				if err := s.deleteSchedule(ctx, client, sc.ScheduleID, logger); err != nil {
					record(fmt.Errorf("deleting schedule %s: %w", sc.ScheduleID, err))
				}
			case <-ctx.Done():
				// Left undeleted; EndAt bounds how long it survives.
			}
		})
	}
	wg.Wait()

	// GenericExecutor checks errors.Is(err, context.Canceled) to tell a stopped
	// iteration from a failed one. The operation errors above are gRPC status
	// errors that do not unwrap to that sentinel, whatever their message says.
	if err := ctx.Err(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

type ScheduleState struct {
	ScheduleID  string
	DeleteAfter time.Duration
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
			EndAt: endTime,
		},
		Action:  action,
		Overlap: pickOverlap(s.config.OverlapPolicy, logger),
	}
	_, err := c.ScheduleClient().Create(ctx, options)
	return sc, err
}

func pickOverlap(policies []enums.ScheduleOverlapPolicy, logger *zap.SugaredLogger) enums.ScheduleOverlapPolicy {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(policies))))
	if err != nil {
		logger.Error("Failed to select overlap policy", "error", err)
		return policies[0]
	}
	return policies[n.Int64()]
}

func (s *SchedulerExecutor) describeSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
	_, err := handle.Describe(ctx)
	if err != nil {
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			// Return nil if schedule is not found (already deleted or never existed)
			logger.Debug("Schedule not found during describe operation", "scheduleID", scheduleID)
			return nil
		}
	}
	return err
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

	err := handle.Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: updateFn,
	})
	if err != nil {
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			// Return nil if schedule is not found (already deleted or never existed)
			logger.Debug("Schedule not found during update operation", "scheduleID", scheduleID)
			return nil
		}
	}
	return err
}

func (s *SchedulerExecutor) deleteSchedule(ctx context.Context, c client.Client, scheduleID string, logger *zap.SugaredLogger) error {
	handle := c.ScheduleClient().GetHandle(ctx, scheduleID)
	err := handle.Delete(ctx)
	if err != nil {
		var notFoundErr *serviceerror.NotFound
		if errors.As(err, &notFoundErr) {
			// Return nil if schedule is not found (already deleted or never existed)
			logger.Debug("Schedule not found during delete operation", "scheduleID", scheduleID)
			return nil
		}
	}
	return err
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
