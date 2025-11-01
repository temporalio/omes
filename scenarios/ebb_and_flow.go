package scenarios

import (
	"context"
	"errors"
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
	// BacklogLogIntervalFlag defines how often the current backlog stats are logged.
	BacklogLogIntervalFlag = "backlog-log-interval"
)

type ebbAndFlowConfig struct {
	MinBacklog                    int64
	MaxBacklog                    int64
	PhaseTime                     time.Duration
	SleepDuration                 time.Duration
	MaxRate                       int64
	ControlInterval               time.Duration
	MaxConsecutiveErrors          int
	BacklogLogInterval            time.Duration
	VisibilityVerificationTimeout time.Duration
	SleepActivityConfig           *loadgen.SleepActivityConfig
}

type ebbAndFlowState struct {
	// TotalCompletedWorkflows tracks the total number of completed workflows across
	// all restarts. It is used to verify workflow counts after the scenario completes.
	TotalCompletedWorkflows int64 `json:"totalCompletedWorkflows"`
}

type ebbAndFlowExecutor struct {
	loadgen.ScenarioInfo
	config              *ebbAndFlowConfig
	rng                 *rand.Rand
	id                  string
	isResuming          bool
	startTime           time.Time
	scheduledActivities atomic.Int64
	completedActivities atomic.Int64
	stateLock           sync.Mutex
	state               *ebbAndFlowState
}

var _ loadgen.Configurable = (*ebbAndFlowExecutor)(nil)
var _ loadgen.Resumable = (*ebbAndFlowExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Oscillates backlog between min and max.\n" +
			"Options:\n" +
			"  min-backlog, max-backlog, phase-time, sleep-duration, max-rate,\n" +
			"  control-interval, max-consecutive-errors, backlog-log-interval.\n" +
			"Duration must be set.",
		ExecutorFn: func() loadgen.Executor { return newEbbAndFlowExecutor() },
	})
}

func newEbbAndFlowExecutor() *ebbAndFlowExecutor {
	return &ebbAndFlowExecutor{state: &ebbAndFlowState{}}
}

func (e *ebbAndFlowExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &ebbAndFlowConfig{
		SleepDuration:                 info.ScenarioOptionDuration(SleepDurationFlag, 1*time.Millisecond),
		MaxRate:                       int64(info.ScenarioOptionInt(MaxRateFlag, 1000)),
		ControlInterval:               info.ScenarioOptionDuration(ControlIntervalFlag, 100*time.Millisecond),
		MaxConsecutiveErrors:          info.ScenarioOptionInt(MaxConsecutiveErrorsFlag, 10),
		BacklogLogInterval:            info.ScenarioOptionDuration(BacklogLogIntervalFlag, 30*time.Second),
		VisibilityVerificationTimeout: info.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 30*time.Second),
	}

	config.MinBacklog = int64(info.ScenarioOptionInt(MinBacklogFlag, 0))
	if config.MinBacklog < 0 {
		return fmt.Errorf("min-backlog must be non-negative, got %d", config.MinBacklog)
	}

	config.MaxBacklog = int64(info.ScenarioOptionInt(MaxBacklogFlag, 30))
	if config.MaxBacklog <= config.MinBacklog {
		return fmt.Errorf("max-backlog must be greater than min-backlog, got max=%d min=%d", config.MaxBacklog, config.MinBacklog)
	}

	config.PhaseTime = info.ScenarioOptionDuration(PhaseTimeFlag, 60*time.Second)
	if config.PhaseTime <= 0 {
		return fmt.Errorf("phase-time must be greater than 0, got %v", config.PhaseTime)
	}

	if sleepActivitiesStr, ok := info.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		// This scenario overrides "count" so do not require it.
		config.SleepActivityConfig, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr, false)
		if err != nil {
			return fmt.Errorf("invalid %s: %w", SleepActivityJsonFlag, err)
		}
	}
	if config.SleepActivityConfig == nil {
		config.SleepActivityConfig = &loadgen.SleepActivityConfig{}
	}
	if len(config.SleepActivityConfig.Groups) == 0 {
		config.SleepActivityConfig.Groups = map[string]loadgen.SleepActivityGroupConfig{"default": {}}
	}
	for name, group := range config.SleepActivityConfig.Groups {
		fixedDist := loadgen.NewFixedDistribution(config.SleepDuration)
		group.SleepDuration = &fixedDist
		config.SleepActivityConfig.Groups[name] = group
	}

	e.config = config
	return nil
}

// Run executes the ebb and flow scenario.
func (e *ebbAndFlowExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}

	e.ScenarioInfo = info
	e.id = fmt.Sprintf("ebb_and_flow_%s", e.RunID)
	e.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	e.startTime = time.Now()

	// Get parsed configuration
	config := e.config
	if config == nil {
		return fmt.Errorf("configuration not parsed - Parse must be called before run")
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

	var consecutiveErrCount int
	errCh := make(chan error, 10000)
	ticker := time.NewTicker(config.ControlInterval)
	defer ticker.Stop()

	// Setup configurable backlog logging
	backlogTicker := time.NewTicker(config.BacklogLogInterval)
	defer backlogTicker.Stop()

	var startWG sync.WaitGroup
	var iter int64 = 1

	e.Logger.Infof("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, phase_time=%v, duration=%v",
		config.MinBacklog, config.MaxBacklog, config.PhaseTime, e.Configuration.Duration)

	var rate int64
	var isDraining bool // true = draining mode, false = growing mode
	var started, completed, backlog, target int64
	cycleStartTime := e.startTime

	for elapsed := time.Duration(0); elapsed < e.Configuration.Duration; elapsed = time.Since(e.startTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err != nil {
				e.Logger.Errorf("Failed to spawn workflow: %v", err)
				consecutiveErrCount++
				if consecutiveErrCount >= config.MaxConsecutiveErrors {
					return fmt.Errorf("got %v consecutive errors, most recent: %w", config.MaxConsecutiveErrors, err)
				}
			} else {
				consecutiveErrCount = 0
			}
		case <-ticker.C:
			started = e.scheduledActivities.Load()
			completed = e.completedActivities.Load()
			backlog = started - completed

			// Check if we need to switch modes.
			if isDraining && backlog <= config.MinBacklog {
				e.Logger.Infof("Backlog reached %d, switching to growing mode", backlog)
				isDraining = false
				cycleStartTime = time.Now()
			} else if !isDraining && backlog >= config.MaxBacklog {
				e.Logger.Infof("Backlog reached %d, switching to draining mode", backlog)
				isDraining = true
				cycleStartTime = time.Now()
			}

			target = calculateBacklogTarget(isDraining, cycleStartTime, config.PhaseTime, config.MinBacklog, config.MaxBacklog)
			rate = calculateSpawnRate(target, backlog, config.MinBacklog, config.MaxBacklog, config.MaxRate)

			if rate > 0 {
				startWG.Add(1)
				go func(iter, rate int64) {
					defer startWG.Done()
					errCh <- e.spawnWorkflowWithActivities(ctx, iter, rate, config.SleepActivityConfig)
				}(iter, rate)
				iter++
			}
		case <-backlogTicker.C:
			e.Logger.Debugf("Backlog: %d, target: %d, rate: %d/s, started: %d, completed: %d",
				backlog, target, rate, started, completed)
		}
	}

	e.Logger.Info("Scenario complete; waiting for all workflows to finish...")
	startWG.Wait()
	e.Logger.Info("Verifying scenario completion...")

	e.stateLock.Lock()
	totalCompletedWorkflows := int(e.state.TotalCompletedWorkflows)
	e.stateLock.Unlock()

	// Post-scenario: verify that at least one workflow was completed.
	if totalCompletedWorkflows == 0 {
		return errors.New("No iterations completed. Either the scenario never ran, or it failed to resume correctly.")
	}

	// Post-scenario: verify reported workflow completion count from Visibility.
	if err := loadgen.MinVisibilityCountEventually(
		ctx,
		e.ScenarioInfo,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: e.Namespace,
			Query: fmt.Sprintf("%s='%s'",
				EbbAndFlowScenarioIdSearchAttribute, e.id),
		},
		totalCompletedWorkflows,
		config.VisibilityVerificationTimeout,
	); err != nil {
		return err
	}

	// Post-scenario: ensure there are no failed or terminated workflows for this run.
	return loadgen.VerifyNoFailedWorkflows(ctx, e.ScenarioInfo, EbbAndFlowScenarioIdSearchAttribute, e.ScenarioInfo.RunID)
}

// Snapshot returns a snapshot of the current state.
func (e *ebbAndFlowExecutor) Snapshot() any {
	e.stateLock.Lock()
	defer e.stateLock.Unlock()

	return *e.state
}

// LoadState loads the state from the provided loader function.
func (e *ebbAndFlowExecutor) LoadState(loader func(any) error) error {
	var state ebbAndFlowState
	if err := loader(&state); err != nil {
		return err
	}

	e.stateLock.Lock()
	defer e.stateLock.Unlock()

	e.state = &state
	e.isResuming = true

	return nil
}

func (e *ebbAndFlowExecutor) spawnWorkflowWithActivities(
	ctx context.Context,
	iteration, rate int64,
	template *loadgen.SleepActivityConfig,
) error {
	// Override activity count to fixed rate.
	fixedDist := loadgen.NewFixedDistribution(rate)
	config := loadgen.SleepActivityConfig{
		Count:  &fixedDist,
		Groups: template.Groups,
	}

	// Start workflow.
	run := e.NewRun(int(iteration))
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-track-%d", e.id, iteration)
	options.WorkflowExecutionErrorWhenAlreadyStarted = false
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
	e.scheduledActivities.Add(rate)

	// Wait for workflow completion
	var result ebbandflow.WorkflowOutput
	err = wf.Get(ctx, &result)
	if err != nil {
		e.Logger.Errorf("ebbAndFlowTrack workflow failed for iteration %d: %v", iteration, err)
	}
	e.completedActivities.Add(rate)
	e.incrementTotalCompletedWorkflow()

	return nil
}

func (e *ebbAndFlowExecutor) incrementTotalCompletedWorkflow() {
	e.stateLock.Lock()
	if e.state != nil {
		e.state.TotalCompletedWorkflows++
	}
	e.stateLock.Unlock()
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
	maxRate int64,
) int64 {
	backlogDelta := float64(target - backlog)                                     // how far backlog is from target
	scaledBacklogDelta := math.Abs(backlogDelta) / float64(maxBacklog-minBacklog) // normalize to 0.0-1.0 range
	gain := 1.0 + 2.0*scaledBacklogDelta                                          // smooth gain scheduling: 1.0 (small errors) to 3.0 (large errors)
	rate := int64(backlogDelta * gain)                                            // calculate desired spawn rate (workflows/second)
	rate = min(maxRate, rate)                                                     // cap at maximum allowed rate
	rate = max(0, rate)
	return rate
}
