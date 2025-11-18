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
	. "github.com/temporalio/omes/loadgen/kitchensink"
)

const (
	// MinBacklogFlag defines the minimum backlog to target.
	// Note that it controls total backlog size, not counting partitions or priority levels.
	MinBacklogFlag = "min-backlog"
	// MaxBacklogFlag defines the maximum backlog to target.
	// Note that it controls total backlog size, not counting partitions or priority levels.
	MaxBacklogFlag = "max-backlog"
	// PeriodFlag defines the period of oscilation of backlog size.
	PeriodFlag = "period"
	// SleepDurationFlag defines the duration an activity sleeps for (default 1ms).
	// This can be used to slow down activity processing, but
	// --worker-worker-activity-per-second may be more intuitive.
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
	Period                        time.Duration
	SleepDuration                 time.Duration
	MaxRate                       int64
	ControlInterval               time.Duration
	MaxConsecutiveErrors          int
	BacklogLogInterval            time.Duration
	VisibilityVerificationTimeout time.Duration
	SleepActivityConfig           *loadgen.SleepActivityConfig
}

type ebbAndFlowState struct {
	ExecutorState loadgen.ExecutorState `json:"executorState"`
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
	completionVerifier  *loadgen.WorkflowCompletionVerifier
	executorState       *loadgen.ExecutorState
}

var _ loadgen.Configurable = (*ebbAndFlowExecutor)(nil)
var _ loadgen.Resumable = (*ebbAndFlowExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Oscillates backlog between min and max.\n" +
			"Options:\n" +
			"  min-backlog, max-backlog, period, sleep-duration, max-rate,\n" +
			"  control-interval, max-consecutive-errors, backlog-log-interval.\n" +
			"Duration must be set.",
		ExecutorFn: func() loadgen.Executor { return newEbbAndFlowExecutor() },
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*ebbAndFlowExecutor)
			if e.completionVerifier == nil || e.executorState == nil {
				return nil
			}
			return e.completionVerifier.VerifyRun(ctx, info, *e.executorState)
		},
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

	// TODO: backwards-compatibility, remove later
	pt := info.ScenarioOptionDuration("phase-time", 60*time.Second)
	config.Period = info.ScenarioOptionDuration(PeriodFlag, pt)
	if config.Period <= 0 {
		return fmt.Errorf("period must be greater than 0, got %v", config.Period)
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

// VerifyRun implements the Verifier interface.
func (e *ebbAndFlowExecutor) VerifyRun(ctx context.Context, info loadgen.ScenarioInfo, state loadgen.ExecutorState) []error {
	if e.completionVerifier == nil || e.executorState == nil {
		return nil
	}
	return e.completionVerifier.VerifyRun(ctx, info, *e.executorState)
}

// Run executes the ebb and flow scenario.
func (e *ebbAndFlowExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}

	e.ScenarioInfo = info
	e.id = fmt.Sprintf("ebb_and_flow_%s", e.ExecutionID)
	e.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	e.startTime = time.Now()

	// Get parsed configuration
	config := e.config
	if config == nil {
		return fmt.Errorf("configuration not parsed - Parse must be called before run")
	}

	// Initialize executor state if needed
	if e.executorState == nil {
		e.executorState = &loadgen.ExecutorState{
			ExecutionID: info.ExecutionID,
		}
	}

	// Restore state if resuming
	if e.isResuming && e.state != nil {
		*e.executorState = e.state.ExecutorState
	}

	// Initialize workflow completion checker with timeout from scenario options
	timeout := info.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 30*time.Second)
	checker, err := loadgen.NewWorkflowCompletionChecker(ctx, info, timeout)
	if err != nil {
		return fmt.Errorf("failed to initialize completion checker: %w", err)
	}
	e.completionVerifier = checker

	var consecutiveErrCount int
	errCh := make(chan error, 10000)
	ticker := time.NewTicker(config.ControlInterval)
	defer ticker.Stop()

	// Setup configurable backlog logging
	backlogTicker := time.NewTicker(config.BacklogLogInterval)
	defer backlogTicker.Stop()

	var startWG sync.WaitGroup
	var iter int64 = 1

	e.Logger.Infof("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, period=%v, duration=%v",
		config.MinBacklog, config.MaxBacklog, config.Period, e.Configuration.Duration)

	var started, completed, backlog, target, rate int64

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

			target = calculateBacklogTarget(elapsed, config.Period, config.MinBacklog, config.MaxBacklog)
			activities := target - backlog
			activities = max(activities, 0)
			activities = min(activities, config.MaxRate)
			if activities > 0 {
				startWG.Add(1)
				go func(iter, rate int64) {
					defer startWG.Done()
					errCh <- e.spawnWorkflowWithActivities(ctx, iter, activities, config.SleepActivityConfig)
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
	e.Logger.Info("Scenario execution complete")

	return nil
}

// Snapshot returns a snapshot of the current state.
func (e *ebbAndFlowExecutor) Snapshot() any {
	e.stateLock.Lock()
	defer e.stateLock.Unlock()

	return ebbAndFlowState{
		ExecutorState: *e.executorState,
	}
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
	iteration, activities int64,
	template *loadgen.SleepActivityConfig,
) error {
	// Override activity count to fixed rate.
	fixedDist := loadgen.NewFixedDistribution(activities)
	config := loadgen.SleepActivityConfig{
		Count:  &fixedDist,
		Groups: template.Groups,
	}

	// Sample activities from the configuration
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	activities := config.Sample(rng)

	// Build actions for the kitchensink workflow
	var actions []*Action
	for _, activityAction := range activities {
		actions = append(actions, &Action{
			Variant: &Action_ExecActivity{
				ExecActivity: activityAction,
			},
		})
	}

	// Start workflow using kitchensink.
	run := e.NewRun(int(iteration))
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-track-%d", e.id, iteration)
	options.WorkflowExecutionErrorWhenAlreadyStarted = false

	workflowInput := &WorkflowInput{
		InitialActions: []*ActionSet{
			{
				Actions:    actions,
				Concurrent: false,
			},
			{
				Actions: []*Action{
					{
						Variant: &Action_ReturnResult{
							ReturnResult: &ReturnResultAction{},
						},
					},
				},
			},
		},
	}

	// Start workflow using kitchensink.
	_, err := e.Client.ExecuteWorkflow(ctx, options, "kitchenSink", workflowInput)
	if err != nil {
		return fmt.Errorf("failed to start kitchensink workflow for iteration %d: %w", iteration, err)
	}
	e.scheduledActivities.Add(activities)

	// Wait for workflow completion
	err = wf.Get(ctx, nil)
	if err != nil {
		e.Logger.Errorf("kitchensink workflow failed for iteration %d: %v", iteration, err)
	}
	e.completedActivities.Add(activities)
	e.incrementTotalCompletedWorkflow()

	// Record completion in executor state for verification
	e.stateLock.Lock()
	e.executorState.CompletedIterations++
	e.executorState.LastCompletedAt = time.Now()
	e.stateLock.Unlock()

	return nil
}

func calculateBacklogTarget(
	elapsed, period time.Duration,
	minBacklog, maxBacklog int64,
) int64 {
	periods := elapsed.Seconds() / period.Seconds()
	osc := (math.Sin(2*math.Pi*(periods-0.25)) + 1.0) / 2
	backlogRange := float64(maxBacklog - minBacklog)
	baseTarget := float64(minBacklog) + osc*backlogRange
	return int64(math.Round(baseTarget))
}
