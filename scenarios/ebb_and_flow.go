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

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Oscillates backlog between min and max over frequency period using simple proportional control. " +
			"Options: min-backlog, max-backlog, frequency, sleep-duration, max-rate, control-interval, max-consecutive-errors, fairness-report-interval, fairness-threshold. Duration must be set.",
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

	startTime       time.Time
	generatedCount  atomic.Int64
	processedCount  atomic.Int64
	fairnessTracker *ebbandflow.FairnessTracker
}

func (e *ebbAndFlow) run(ctx context.Context) error {
	e.startTime = time.Now()

	// Initialize search attribute for visibility tracking
	err := loadgen.InitSearchAttribute(
		ctx,
		e.ScenarioInfo,
		EbbAndFlowScenarioIdSearchAttribute,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize search attribute %s: %w", EbbAndFlowScenarioIdSearchAttribute, err)
	}

	// Initialize fairness tracker
	e.fairnessTracker = ebbandflow.NewFairnessTracker()

	// Parse and validate scenario options.
	minBacklog := int64(e.ScenarioOptionInt("min-backlog", 10))
	maxBacklog := int64(e.ScenarioOptionInt("max-backlog", 30))
	frequency := e.ScenarioOptionDuration("frequency", 60*time.Second)
	sleepDuration := e.ScenarioOptionDuration("sleep-duration", 1*time.Millisecond)
	maxRate := e.ScenarioOptionInt("max-rate", 1000)
	controlInterval := e.ScenarioOptionDuration("control-interval", 100*time.Millisecond)
	maxConsecutiveErrors := e.ScenarioOptionInt("max-consecutive-errors", 10)
	fairnessReportInterval := e.ScenarioOptionDuration("fairness-report-interval", frequency)
	fairnessThreshold := e.ScenarioOptionFloat("fairness-threshold", 1.5)

	if minBacklog < 0 {
		return fmt.Errorf("min-backlog must be non-negative")
	}
	if maxBacklog <= minBacklog {
		return fmt.Errorf("max-backlog must be greater than min-backlog")
	}
	if frequency <= 0 {
		return fmt.Errorf("frequency must be greater than 0")
	}
	if fairnessThreshold <= 1.0 {
		return fmt.Errorf("fairness-threshold must be greater than 1.0")
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
	go e.fairnessReportLoop(ctx, fairnessTicker, fairnessThreshold)

	var startWG sync.WaitGroup
	iter := 1

	e.Logger.Infof("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, frequency=%v, duration=%v",
		minBacklog, maxBacklog, frequency, e.Configuration.Duration)

	isDraining := false // true = draining mode, false = growing mode
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
			generated := e.generatedCount.Load()
			processed := e.processedCount.Load()
			backlog := generated - processed

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

			target := calculateBacklogTarget(isDraining, cycleStartTime, frequency, minBacklog, maxBacklog)
			rate := calculateSpawnRate(target, backlog, minBacklog, maxBacklog, maxRate)

			e.Logger.Debugf("Backlog: %d, target: %d, rate: %d/s, gen: %d, proc: %d",
				backlog, target, rate, generated, processed)

			if rate > 0 {
				startWG.Add(1)
				go func(iteration, count int) {
					defer startWG.Done()
					errCh <- e.spawnWorkflowWithActivities(ctx, iteration, count, sleepActivityConfig)
				}(iter, rate)
				iter++
			}
		}
	}

	e.Logger.Info("Scenario complete; waiting for all workflows to finish...")
	startWG.Wait()

	// Post-scenario: verify that at least one iteration was completed.
	completedWorkflows := int(e.processedCount.Load())
	if completedWorkflows == 0 {
		return fmt.Errorf("no workflows completed")
	}

	// Post-scenario: verify reported workflow completion count from Visibility.
	visibilityVerificationTimeout := e.ScenarioOptionDuration("visibility-count-timeout", 30*time.Second)
	return loadgen.MinVisibilityCountEventually(
		ctx,
		e.ScenarioInfo,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: e.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				EbbAndFlowScenarioIdSearchAttribute, e.RunID),
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
	runID := iteration * 10000
	run := e.NewRun(runID)
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-%d", options.ID, e.startTime.UnixMilli()) // avoid collision on a restart
	options.SearchAttributes = map[string]interface{}{
		EbbAndFlowScenarioIdSearchAttribute: e.RunID,
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
				e.fairnessTracker.Track(activityResult.FairnessKey, activityResult.ScheduleToStartMS)
			}
		}
	}
	e.processedCount.Add(1)

	return nil
}

func calculateBacklogTarget(
	isDraining bool,
	cycleStartTime time.Time,
	frequency time.Duration,
	minBacklog, maxBacklog int64,
) int64 {
	// Compute elapsed time since mode switch.
	elapsed := time.Since(cycleStartTime)
	progress := math.Min(1.0, elapsed.Seconds()/frequency.Seconds())

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
func (e *ebbAndFlow) fairnessReportLoop(ctx context.Context, ticker *time.Ticker, threshold float64) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Generate fairness report.
			report, err := e.fairnessTracker.GetReport(threshold)
			if err != nil {
				e.Logger.Errorf("Skipping fairness report: %v", err)
				continue
			}

			// Log the report.
			options := e.NewRun(0).DefaultStartWorkflowOptions()
			options.ID = fmt.Sprintf("fairness-report-%s-%d", e.RunID, time.Now().UnixMilli())
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
