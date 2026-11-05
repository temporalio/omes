package scenarios

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/loadgen/phaseplan"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/types/known/durationpb"
)

const (
	standaloneLatencyActivityType = "MeasureLatencyActivity"

	// PhasePlanJSONFlag defines the ordered fixed-rate phases for this scenario.
	PhasePlanJSONFlag = "phase-plan-json"
	// ActivityTaskQueueFlag overrides the standalone activity task queue.
	ActivityTaskQueueFlag = "activity-task-queue"
	// StartConcurrencyFlag controls concurrent standalone activity start RPCs.
	StartConcurrencyFlag = "start-concurrency"
	// MinAchievedRateRatioFlag defines the minimum acceptable achieved/requested rate ratio.
	MinAchievedRateRatioFlag = "min-achieved-rate-ratio"
	// RateMeasurementWindowFlag defines the minimum phase duration on which rate is enforced.
	RateMeasurementWindowFlag = "rate-measurement-window"
	// PhasedActivitySleepDurationFlag controls how long each standalone activity sleeps.
	PhasedActivitySleepDurationFlag = "sleep-duration"
	// PhasedActivityMaxConsecutiveErrorsFlag controls when repeated start failures abort the scenario.
	PhasedActivityMaxConsecutiveErrorsFlag = "max-consecutive-errors"
	// PhasedActivityVisibilityVerificationTimeoutFlag bounds final activity visibility verification.
	PhasedActivityVisibilityVerificationTimeoutFlag = "visibility-verification-timeout"
)

type phasedActivityLoadConfig struct {
	// PhasePlan contains the ordered open-loop standalone-activity phases.
	PhasePlan *phaseplan.Plan
	// ActivityTaskQueue is the task queue targeted by every standalone activity.
	ActivityTaskQueue string
	// SleepDuration is how long each generated activity waits before completing.
	SleepDuration time.Duration
	// StartConcurrency is the maximum number of concurrent standalone-activity start RPCs.
	StartConcurrency int
	// MaxConsecutiveErrors is the number of consecutive start errors tolerated before aborting.
	MaxConsecutiveErrors int
	// MinAchievedRateRatio is the minimum successful/requested rate ratio accepted for a measured phase.
	MinAchievedRateRatio float64
	// RateMeasurementWindow is the minimum phase duration for which achieved-rate enforcement applies.
	RateMeasurementWindow time.Duration
	// VisibilityVerificationTimeout bounds the final wait for all accepted activities to complete.
	VisibilityVerificationTimeout time.Duration
}

// phasedActivityLoadExecutor generates standalone activities according to an
// open-loop sequence of fixed-rate phases.
type phasedActivityLoadExecutor struct {
	loadgen.ScenarioInfo
	config *phasedActivityLoadConfig
}

var _ loadgen.Configurable = (*phasedActivityLoadExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Generates standalone activities using ordered, open-loop fixed-rate phases.",
		Options: func(o *loadgen.OptionSet) {
			o.String(PhasePlanJSONFlag, "", "JSON phase plan; use @<file> to read from a file.")
			o.MarkRequired(PhasePlanJSONFlag)
			o.String(ActivityTaskQueueFlag, "", "Standalone activity task queue; defaults to omes-<run-id>.")
			o.Duration(PhasedActivitySleepDurationFlag, time.Millisecond, "How long each activity sleeps.")
			o.Int(StartConcurrencyFlag, 64, "Maximum concurrent standalone activity start RPCs.")
			o.Int(PhasedActivityMaxConsecutiveErrorsFlag, 10, "Consecutive start errors tolerated before stopping.")
			o.Float64(MinAchievedRateRatioFlag, 0.90, "Minimum achieved/requested rate ratio for sustained phases; 0 disables enforcement.")
			o.Duration(RateMeasurementWindowFlag, 10*time.Second, "Minimum non-zero-rate phase duration on which achieved rate is enforced.")
			o.Duration(PhasedActivityVisibilityVerificationTimeoutFlag, 30*time.Second, "Timeout for final activity visibility verification.")
		},
		ExecutorFn: func() loadgen.Executor { return newPhasedActivityLoadExecutor() },
	})
}

func newPhasedActivityLoadExecutor() *phasedActivityLoadExecutor {
	return &phasedActivityLoadExecutor{}
}

func (e *phasedActivityLoadExecutor) Configure(info loadgen.ScenarioInfo) error {
	if info.Configuration.StartAt.IsZero() {
		return fmt.Errorf("phased_activity_load requires --start-at")
	}
	plan, err := phaseplan.Parse(info.OptionString(PhasePlanJSONFlag))
	if err != nil {
		return fmt.Errorf("invalid %s: %w", PhasePlanJSONFlag, err)
	}
	config := &phasedActivityLoadConfig{
		PhasePlan:                     plan,
		ActivityTaskQueue:             info.OptionString(ActivityTaskQueueFlag),
		SleepDuration:                 info.OptionDuration(PhasedActivitySleepDurationFlag),
		StartConcurrency:              info.OptionInt(StartConcurrencyFlag),
		MaxConsecutiveErrors:          info.OptionInt(PhasedActivityMaxConsecutiveErrorsFlag),
		MinAchievedRateRatio:          info.OptionFloat64(MinAchievedRateRatioFlag),
		RateMeasurementWindow:         info.OptionDuration(RateMeasurementWindowFlag),
		VisibilityVerificationTimeout: info.OptionDuration(PhasedActivityVisibilityVerificationTimeoutFlag),
	}
	if config.ActivityTaskQueue == "" {
		config.ActivityTaskQueue = loadgen.TaskQueueForRun(info.RunID)
	}
	if config.SleepDuration < 0 {
		return fmt.Errorf("%s must be non-negative, got %v", PhasedActivitySleepDurationFlag, config.SleepDuration)
	}
	if config.StartConcurrency <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %d", StartConcurrencyFlag, config.StartConcurrency)
	}
	if config.MaxConsecutiveErrors <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %d", PhasedActivityMaxConsecutiveErrorsFlag, config.MaxConsecutiveErrors)
	}
	if config.MinAchievedRateRatio < 0 || config.MinAchievedRateRatio > 1 {
		return fmt.Errorf("%s must be between 0 and 1, got %v", MinAchievedRateRatioFlag, config.MinAchievedRateRatio)
	}
	if config.RateMeasurementWindow <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %v", RateMeasurementWindowFlag, config.RateMeasurementWindow)
	}
	if config.VisibilityVerificationTimeout <= 0 {
		return fmt.Errorf("%s must be greater than 0, got %v",
			PhasedActivityVisibilityVerificationTimeoutFlag, config.VisibilityVerificationTimeout)
	}
	e.config = config
	return nil
}

func (e *phasedActivityLoadExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}
	e.ScenarioInfo = info
	return e.runPhasePlan(ctx)
}

type phaseRunStats struct {
	// SuccessfulStarts counts activity start RPCs accepted by the server.
	SuccessfulStarts int64
	// StartFailures counts activity start RPCs that returned an error.
	StartFailures int64
	// OfferedStarts counts pacing opportunities delivered to an available start worker.
	OfferedStarts int64
	// Elapsed is the phase runtime, including the final wait for in-flight start RPCs.
	Elapsed time.Duration
}

func (e *phasedActivityLoadExecutor) runPhasePlan(ctx context.Context) error {
	if !e.Configuration.DoNotRegisterSearchAttributes {
		if err := e.RegisterDefaultSearchAttributes(ctx); err != nil {
			return err
		}
	}

	config := e.config
	startAt := e.Configuration.StartAt
	var activitySequence atomic.Int64
	var totalSuccessful atomic.Int64
	var totalFailures atomic.Int64
	activityTimeout := config.PhasePlan.TotalDuration() + config.VisibilityVerificationTimeout + time.Minute

	e.Logger.Infof("Waiting for synchronized phase-plan start: start_at=%s",
		startAt.UTC().Format(time.RFC3339Nano))
	if err := phaseplan.WaitForStart(ctx, startAt); err != nil {
		return err
	}
	planStarted := time.Now()

	e.Logger.Infof(
		"Starting phased activity load: phases=%d task_queue=%q start_concurrency=%d started_at=%s",
		len(config.PhasePlan.Phases), config.ActivityTaskQueue, config.StartConcurrency,
		planStarted.UTC().Format(time.RFC3339Nano),
	)

	for index, phase := range config.PhasePlan.Phases {
		scheduledStart := config.PhasePlan.PhaseStart(startAt, index)
		if err := phaseplan.WaitUntil(ctx, scheduledStart); err != nil {
			return err
		}
		phaseStarted := time.Now()
		e.Logger.Infof(
			"Phase started: index=%d name=%q requested_add_rate=%.3f duration=%v scheduled_boundary=%s "+
				"actual_boundary=%s lateness=%v cumulative_submitted=%d",
			index, phase.Name, phase.AddRate, phase.Duration,
			scheduledStart.UTC().Format(time.RFC3339Nano), phaseStarted.UTC().Format(time.RFC3339Nano),
			max(phaseStarted.Sub(scheduledStart), 0), totalSuccessful.Load(),
		)

		stats, err := e.runStandaloneActivityPhase(
			ctx, phase, config.PhasePlan.PhaseEnd(startAt, index), activityTimeout,
			&activitySequence, &totalSuccessful, &totalFailures,
		)
		phaseCompleted := time.Now()
		scheduledEnd := config.PhasePlan.PhaseEnd(startAt, index)
		achievedRate := float64(stats.SuccessfulStarts) / stats.Elapsed.Seconds()
		e.Logger.Infof(
			"Phase completed: index=%d name=%q requested_add_rate=%.3f achieved_add_rate=%.3f "+
				"offered=%d successful=%d failed=%d elapsed=%v scheduled_boundary=%s actual_boundary=%s "+
				"lateness=%v cumulative_submitted=%d",
			index, phase.Name, phase.AddRate, achievedRate, stats.OfferedStarts,
			stats.SuccessfulStarts, stats.StartFailures, stats.Elapsed,
			scheduledEnd.UTC().Format(time.RFC3339Nano), phaseCompleted.UTC().Format(time.RFC3339Nano),
			max(phaseCompleted.Sub(scheduledEnd), 0), totalSuccessful.Load(),
		)
		if err != nil {
			return fmt.Errorf("phase %q failed: %w", phase.Name, err)
		}
		if err := validateAchievedRate(phase, stats, config.MinAchievedRateRatio, config.RateMeasurementWindow); err != nil {
			return err
		}
	}

	submitted := totalSuccessful.Load()
	if failures := totalFailures.Load(); failures > 0 {
		return fmt.Errorf("standalone activity generation had %d failed starts out of %d attempts",
			failures, submitted+failures)
	}
	e.Logger.Infof("Phase plan complete: submitted=%d elapsed=%v; verifying standalone activities",
		submitted, time.Since(planStarted))
	if err := e.verifyStandaloneActivities(ctx, submitted); err != nil {
		return err
	}
	e.Logger.Infof("Standalone activity verification complete: completed=%d", submitted)
	return nil
}

func (e *phasedActivityLoadExecutor) runStandaloneActivityPhase(
	ctx context.Context,
	phase phaseplan.Phase,
	endsAt time.Time,
	activityTimeout time.Duration,
	activitySequence, totalSuccessful, totalFailures *atomic.Int64,
) (phaseRunStats, error) {
	startedAt := time.Now()
	if phase.AddRate == 0 {
		timer := time.NewTimer(max(time.Until(endsAt), 0))
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return phaseRunStats{Elapsed: time.Since(startedAt)}, ctx.Err()
		case <-timer.C:
			return phaseRunStats{Elapsed: time.Since(startedAt)}, nil
		}
	}

	interval := intervalForRate(phase.AddRate)
	jobs := make(chan int64)
	var workers sync.WaitGroup
	var successfulStarts atomic.Int64
	var startFailures atomic.Int64
	var consecutiveFailures atomic.Int64
	fatalStartErr := make(chan error, 1)
	for range e.config.StartConcurrency {
		workers.Go(func() {
			for sequence := range jobs {
				if err := e.startStandaloneActivity(ctx, sequence, activityTimeout); err != nil {
					startFailures.Add(1)
					totalFailures.Add(1)
					e.Logger.Errorf("Standalone activity start failed: sequence=%d error=%v", sequence, err)
					if consecutiveFailures.Add(1) >= int64(e.config.MaxConsecutiveErrors) {
						select {
						case fatalStartErr <- err:
						default:
						}
					}
					continue
				}
				consecutiveFailures.Store(0)
				successfulStarts.Add(1)
				totalSuccessful.Add(1)
			}
		})
	}

	ticker := time.NewTicker(interval)
	timer := time.NewTimer(max(time.Until(endsAt), 0))
	defer ticker.Stop()
	defer timer.Stop()
	sequence := activitySequence.Add(1)
	select {
	case jobs <- sequence:
	case <-ctx.Done():
		close(jobs)
		workers.Wait()
		return phaseRunStats{Elapsed: time.Since(startedAt)}, ctx.Err()
	}
	var offeredStarts int64 = 1
	var fatalErr error

runLoop:
	for {
		select {
		case <-ctx.Done():
			break runLoop
		case <-timer.C:
			break runLoop
		case fatalErr = <-fatalStartErr:
			break runLoop
		case <-ticker.C:
			sequence := activitySequence.Add(1)
			select {
			case jobs <- sequence:
				offeredStarts++
			default:
				// Do not queue missed opportunities: catching up later would create a burst
				// and invalidate the open-loop rate shape.
			}
		}
	}
	close(jobs)
	workers.Wait()
	stats := phaseRunStats{
		SuccessfulStarts: successfulStarts.Load(),
		StartFailures:    startFailures.Load(),
		OfferedStarts:    offeredStarts,
		Elapsed:          time.Since(startedAt),
	}
	if ctx.Err() != nil {
		return stats, ctx.Err()
	}
	if fatalErr != nil {
		return stats, fmt.Errorf(
			"%d consecutive standalone activity starts failed; ensure activity.enableStandalone is enabled: %w",
			e.config.MaxConsecutiveErrors, fatalErr,
		)
	}
	if stats.StartFailures > 0 {
		return stats, fmt.Errorf(
			"%d standalone activity starts failed; ensure activity.enableStandalone is enabled and inspect frontend errors",
			stats.StartFailures,
		)
	}
	return stats, nil
}

func intervalForRate(rate float64) time.Duration {
	interval := time.Duration(float64(time.Second) / rate)
	return max(interval, time.Nanosecond)
}

func validateAchievedRate(
	phase phaseplan.Phase,
	stats phaseRunStats,
	minRatio float64,
	measurementWindow time.Duration,
) error {
	if phase.AddRate == 0 || minRatio == 0 || phase.Duration < measurementWindow {
		return nil
	}
	achievedRate := float64(stats.SuccessfulStarts) / stats.Elapsed.Seconds()
	ratio := achievedRate / phase.AddRate
	if ratio < minRatio {
		return fmt.Errorf(
			"phase %q was under-driven: requested %.3f activities/s, achieved %.3f activities/s (%.1f%%, minimum %.1f%%); "+
				"the generator or frontend was the limiting component",
			phase.Name, phase.AddRate, achievedRate, ratio*100, minRatio*100,
		)
	}
	return nil
}

func (e *phasedActivityLoadExecutor) startStandaloneActivity(
	ctx context.Context,
	sequence int64,
	scheduleToCloseTimeout time.Duration,
) error {
	input := &kitchensink.ExecuteActivityAction{
		ActivityType: &kitchensink.ExecuteActivityAction_Delay{
			Delay: durationpb.New(e.config.SleepDuration),
		},
	}
	_, err := e.Client.ExecuteActivity(ctx, client.StartActivityOptions{
		ID:                     standaloneActivityID(e.ExecutionID, sequence),
		TaskQueue:              e.config.ActivityTaskQueue,
		ScheduleToCloseTimeout: scheduleToCloseTimeout,
		StartToCloseTimeout:    e.config.SleepDuration + time.Minute,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(loadgen.OmesExecutionIDSearchAttribute).ValueSet(e.ExecutionID),
		),
	}, standaloneLatencyActivityType, input)
	return err
}

func standaloneActivityID(executionID string, sequence int64) string {
	return fmt.Sprintf("phased-activity-load-%s-activity-%012d", executionID, sequence)
}

func (e *phasedActivityLoadExecutor) verifyStandaloneActivities(ctx context.Context, submitted int64) error {
	query := fmt.Sprintf("%s = '%s' GROUP BY ExecutionStatus",
		loadgen.OmesExecutionIDSearchAttribute, e.ExecutionID)
	verificationCtx, cancel := context.WithTimeout(ctx, e.config.VisibilityVerificationTimeout)
	defer cancel()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastCounts map[string]int64
	var lastErr error
	for {
		result, err := e.Client.CountActivities(verificationCtx, client.CountActivitiesOptions{Query: query})
		if err != nil {
			lastErr = err
		} else {
			lastErr = nil
			lastCounts = make(map[string]int64, len(result.Groups))
			for _, group := range result.Groups {
				if len(group.GroupValues) == 0 {
					continue
				}
				status := fmt.Sprint(group.GroupValues[0])
				lastCounts[status] = group.Count
			}
			if result.Count == submitted && lastCounts["Completed"] == submitted {
				return nil
			}
			if result.Count == submitted && terminalActivityFailures(lastCounts) > 0 {
				return fmt.Errorf("standalone activity verification found terminal failures: %v", lastCounts)
			}
		}

		select {
		case <-verificationCtx.Done():
			if lastErr != nil {
				return fmt.Errorf("standalone activity visibility verification failed: %w", lastErr)
			}
			return fmt.Errorf("standalone activities did not all complete before %v: submitted=%d statuses=%v",
				e.config.VisibilityVerificationTimeout, submitted, lastCounts)
		case <-ticker.C:
		}
	}
}

func terminalActivityFailures(counts map[string]int64) int64 {
	return counts["Failed"] + counts["TimedOut"] + counts["Canceled"] + counts["Terminated"]
}
