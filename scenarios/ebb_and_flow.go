package scenarios

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/ebbandflow"
	"go.temporal.io/api/workflowservice/v1"
)

const (
	EbbAndFlowScenarioIdSearchAttribute = "EbbAndFlowScenarioId"
)

const (
	// MinBacklogFlag defines the minimum backlog to target.
	MinBacklogFlag = "min-backlog"
	// MaxBacklogFlag defines the maximum backlog to target.
	MaxBacklogFlag = "max-backlog"
	// PhaseTimeFlag defines the duration of each growing and draining phase.
	PhaseTimeFlag = "phase-time"
	// SleepDurationFlag defines the duration an activity sleeps for.
	SleepDurationFlag = "sleep-duration"
	// MaxRateFlag defines the maximum number of workflows to spawn per control interval.
	MaxRateFlag = "max-rate"
	// ControlIntervalFlag defines how often the backlog is controlled.
	ControlIntervalFlag = "control-interval"
	// MaxConsecutiveErrorsFlag defines how many consecutive errors are tolerated before stopping the scenario.
	MaxConsecutiveErrorsFlag = "max-consecutive-errors"
	// FairnessReportIntervalFlag defines how often fairness reports are logged.
	FairnessReportIntervalFlag = "fairness-report-interval"
	// BacklogLogIntervalFlag defines how often the current backlog stats are logged.
	BacklogLogIntervalFlag = "backlog-log-interval"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Oscillates backlog between min and max.\n" +
			"Options:\n" +
			"  min-backlog, max-backlog, phase-time, sleep-duration, max-rate,\n" +
			"  control-interval, max-consecutive-errors, fairness-report-interval,\n" +
			"  fairness-threshold, backlog-log-interval.\n" +
			"Duration must be set.",
		Executor: loadgen.ExecutorFunc(func(ctx context.Context, runOptions loadgen.ScenarioInfo) error {
			return (&ebbAndFlow{
				ScenarioInfo: runOptions,
				rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
			}).run(ctx)
		}),
	})
}

type ebbAndFlow struct {
	loadgen.ScenarioInfo
	rng *rand.Rand

	id              string
	startTime       time.Time
	generatedCount  atomic.Int64
	processedCount  atomic.Int64
	fairnessTracker *ebbandflow.FairnessTracker
}

func (e *ebbAndFlow) run(ctx context.Context) error {
	e.startTime = time.Now()
	e.id = fmt.Sprintf("ebb_and_flow_%s", e.RunID)
	e.fairnessTracker = ebbandflow.NewFairnessTracker()

	// Parse and validate scenario options.
	minBacklog := int64(e.ScenarioOptionInt(MinBacklogFlag, 0))
	maxBacklog := int64(e.ScenarioOptionInt(MaxBacklogFlag, 30))
	phaseTime := e.ScenarioOptionDuration(PhaseTimeFlag, 60*time.Second)
	sleepDuration := e.ScenarioOptionDuration(SleepDurationFlag, 1*time.Millisecond)
	maxRate := e.ScenarioOptionInt(MaxRateFlag, 1000)
	controlInterval := e.ScenarioOptionDuration(ControlIntervalFlag, 100*time.Millisecond)
	maxConsecutiveErrors := e.ScenarioOptionInt(MaxConsecutiveErrorsFlag, 10)
	fairnessReportInterval := e.ScenarioOptionDuration(FairnessReportIntervalFlag, phaseTime) // default to phase time
	backlogLogInterval := e.ScenarioOptionDuration(BacklogLogIntervalFlag, 30*time.Second)
	visibilityVerificationTimeout := e.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 30*time.Second)

	if minBacklog < 0 {
		return fmt.Errorf("min-backlog must be non-negative")
	}
	if maxBacklog <= minBacklog {
		return fmt.Errorf("max-backlog must be greater than min-backlog")
	}
	if phaseTime <= 0 {
		return fmt.Errorf("phase-time must be greater than 0")
	}

	// Initialize search attribute for visibility tracking
	err := loadgen.InitSearchAttribute(
		ctx,
		e.ScenarioInfo,
		EbbAndFlowScenarioIdSearchAttribute,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize search attribute %s: %w", EbbAndFlowScenarioIdSearchAttribute, err)
	}

	// Activity config
	var sleepActivityConfig *loadgen.SleepActivityConfig
	if sleepActivitiesStr, ok := e.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		sleepActivityConfig, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", SleepActivityJsonFlag, err)
		}
	}
	if sleepActivityConfig == nil {
		sleepActivityConfig = &loadgen.SleepActivityConfig{}
	}
	if len(sleepActivityConfig.Groups) == 0 {
		sleepActivityConfig.Groups = map[string]loadgen.SleepActivityGroupConfig{"default": {}}
	}
	for name, group := range sleepActivityConfig.Groups {
		fixedDist := loadgen.NewFixedDistribution(sleepDuration)
		group.SleepDuration = &fixedDist
		sleepActivityConfig.Groups[name] = group
	}

	var consecutiveErrCount int
	errCh := make(chan error, 10000)
	ticker := time.NewTicker(controlInterval)
	defer ticker.Stop()

	// Setup fairness reporting
	fairnessTicker := time.NewTicker(fairnessReportInterval)
	defer fairnessTicker.Stop()
	go e.fairnessReportLoop(ctx, fairnessTicker)

	// Setup configurable backlog logging
	backlogTicker := time.NewTicker(backlogLogInterval)
	defer backlogTicker.Stop()

	var startWG sync.WaitGroup
	iter := 1

	e.Logger.Infof("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, phase_time=%v, duration=%v",
		minBacklog, maxBacklog, phaseTime, e.Configuration.Duration)

	var rate int
	var isDraining bool // true = draining mode, false = growing mode
	var generated, processed, backlog, target int64
	cycleStartTime := e.startTime

	for elapsed := time.Duration(0); elapsed < e.Configuration.Duration; elapsed = time.Since(e.startTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				e.Logger.Errorf("Failed to spawn workflow: %v", err)
				consecutiveErrCount++
				if consecutiveErrCount >= maxConsecutiveErrors {
					return fmt.Errorf("got %v consecutive errors, most recent: %w", maxConsecutiveErrors, err)
				}
			} else {
				consecutiveErrCount = 0
			}
		case <-ticker.C:
			generated = e.generatedCount.Load()
			processed = e.processedCount.Load()
			backlog = generated - processed

			// Check if we need to switch modes.
			if isDraining && backlog <= minBacklog {
				e.Logger.Infof("Backlog reached %d, switching to growing mode", backlog)
				isDraining = false
				cycleStartTime = time.Now()
			} else if !isDraining && backlog >= maxBacklog {
				e.Logger.Infof("Backlog reached %d, switching to draining mode", backlog)
				isDraining = true
				cycleStartTime = time.Now()
			}

			target = calculateBacklogTarget(isDraining, cycleStartTime, phaseTime, minBacklog, maxBacklog)
			rate = calculateSpawnRate(target, backlog, minBacklog, maxBacklog, maxRate)

			if rate > 0 {
				startWG.Add(1)
				go func(iteration, count int) {
					defer startWG.Done()
					errCh <- e.spawnWorkflowWithActivities(ctx, iteration, count, sleepActivityConfig)
				}(iter, rate)
				iter++
			}
		case <-backlogTicker.C:
			e.Logger.Infof("Backlog: %d, target: %d, rate: %d/s, gen: %d, proc: %d",
				backlog, target, rate, generated, processed)
		}
	}

	e.Logger.Info("Scenario complete; waiting for all workflows to finish...")
	startWG.Wait()
	e.Logger.Info("Verifying scenario completion...")

	// Post-scenario: verify that at least one iteration was completed.
	completedWorkflows := int(e.processedCount.Load())
	if completedWorkflows == 0 {
		return fmt.Errorf("no workflows completed")
	}

	// Post-scenario: verify reported workflow completion count from Visibility.
	return loadgen.MinVisibilityCountEventually(
		ctx,
		e.ScenarioInfo,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: e.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				EbbAndFlowScenarioIdSearchAttribute, e.id),
		},
		completedWorkflows,
		visibilityVerificationTimeout,
	)
}

func (e *ebbAndFlow) spawnWorkflowWithActivities(
	ctx context.Context,
	iteration, rate int,
	template *loadgen.SleepActivityConfig,
) error {
	// Override activity count to fixed rate.
	fixedDist := loadgen.NewFixedDistribution(int64(rate))
	config := loadgen.SleepActivityConfig{
		Count:  &fixedDist,
		Groups: template.Groups,
	}

	// Start workflow.
	run := e.NewRun(iteration)
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-track-%d", e.id, iteration) // avoid collision on a restart
	options.SearchAttributes = map[string]interface{}{
		EbbAndFlowScenarioIdSearchAttribute: e.id,
	}

	workflowInput := &ebbandflow.WorkflowParams{
		SleepActivities: &config,
	}

	// Start workflow to track activity timings.
	wf, err := e.Client.ExecuteWorkflow(ctx, options, "ebbAndFlowTrack", workflowInput)
	if err != nil {
		return fmt.Errorf("failed to start ebbAndFlowTrack workflow for iteration %d: %w", iteration, err)
	}
	e.generatedCount.Add(1)

	// Wait for workflow completion and collect results.
	var result ebbandflow.WorkflowOutput
	err = wf.Get(ctx, &result)
	if err != nil {
		e.Logger.Errorf("ebbAndFlowTrack workflow failed for iteration %d: %v", iteration, err)
	} else {
		for _, activityResult := range result.Timings {
			if activityResult.FairnessKey != "" {
				e.fairnessTracker.Track(activityResult.FairnessKey, activityResult.FairnessWeight, activityResult.ScheduleToStartMS)
			}
		}
	}
	e.processedCount.Add(1)

	return nil
}

func calculateBacklogTarget(
	isDraining bool,
	cycleStartTime time.Time,
	phaseTime time.Duration,
	minBacklog, maxBacklog int64,
) int64 {
	// Compute elapsed time since mode switch.
	elapsed := time.Since(cycleStartTime)
	progress := math.Min(1.0, elapsed.Seconds()/phaseTime.Seconds())

	// Oscillation curve: cosine easing
	var osc float64
	if isDraining {
		osc = (1 + math.Cos(math.Pi*progress)) / 2 // 1 → 0
	} else {
		osc = (1 - math.Cos(math.Pi*progress)) / 2 // 0 → 1
	}

	backlogRange := float64(maxBacklog - minBacklog)
	baseTarget := float64(minBacklog) + osc*backlogRange
	inflatedTarget := baseTarget * 1.10 // apply 10% overshoot
	inflatedTarget = max(0, inflatedTarget)
	return int64(math.Round(inflatedTarget))
}

func calculateSpawnRate(
	target int64,
	backlog int64,
	minBacklog, maxBacklog int64,
	maxRate int,
) int {
	backlogDelta := float64(target - backlog)                                     // how far backlog is from target
	scaledBacklogDelta := math.Abs(backlogDelta) / float64(maxBacklog-minBacklog) // normalize to 0.0-1.0 range
	gain := 1.0 + 2.0*scaledBacklogDelta                                          // smooth gain scheduling: 1.0 (small errors) to 3.0 (large errors)
	rate := int(backlogDelta * gain)                                              // calculate desired spawn rate (workflows/second)
	rate = min(maxRate, rate)                                                     // cap at maximum allowed rate
	rate = max(0, rate)
	return rate
}

// fairnessReportLoop periodically generates fairness reports and starts reporting workflows.
func (e *ebbAndFlow) fairnessReportLoop(ctx context.Context, ticker *time.Ticker) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Generate fairness report.
			report, err := e.fairnessTracker.GetReport()
			if err != nil {
				e.Logger.Warnf("Skipping fairness report: %v", err)
				continue
			}

			// Log the report.
			options := e.NewRun(0).DefaultStartWorkflowOptions()
			options.ID = fmt.Sprintf("%s-report-%d", e.id, time.Now().UnixMilli())
			_, err = e.Client.ExecuteWorkflow(ctx, options, "ebbAndFlowReport", *report)
			if err != nil {
				e.Logger.Errorf("Failed to start fairness report workflow: %v", err)
			} else {
				// Clear the data after successfully creating and submitting the report.
				e.fairnessTracker.Clear()
			}
		}
	}
}
