package scenarios

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/ebbandflow"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/temporal"
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
	// MinAddRateFlag defines the minimum add rate in activities/sec. When both
	// min-add-rate and max-add-rate are set, the scenario operates in rate mode.
	MinAddRateFlag = "min-add-rate"
	// MaxAddRateFlag defines the maximum add rate in activities/sec.
	MaxAddRateFlag = "max-add-rate"
	// RatePeriodFlag defines the period of rate oscillation.
	RatePeriodFlag = "rate-period"
	// BacklogPeriodFlag defines the period of backlog oscillation. Defaults to
	// the value of PeriodFlag for backwards compatibility.
	BacklogPeriodFlag = "backlog-period"
	// BatchSizeFlag defines the max activities per workflow (0 = unlimited).
	BatchSizeFlag = "batch-size"
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
	// Rate mode: enabled when MinAddRate and MaxAddRate are both set.
	RateMode      bool
	MinAddRate    float64
	MaxAddRate    float64
	RatePeriod    time.Duration
	BacklogPeriod time.Duration
	BatchSize     int64
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
		Description: "Oscillates backlog between min and max. Set both min-add-rate and\n" +
			"max-add-rate to switch to rate mode. Duration must be set.",
		Options: func(o *loadgen.OptionSet) {
			o.Int(MinBacklogFlag, 0, "Minimum total backlog to target.")
			o.Int(MaxBacklogFlag, 30, "Maximum total backlog to target.")
			o.Duration(PeriodFlag, 60*time.Second, "Period of backlog-size oscillation.")
			o.Duration(SleepDurationFlag, time.Millisecond, "How long each activity sleeps.")
			o.Int(MaxRateFlag, 1000, "Maximum workflows spawned per control interval.")
			o.Duration(ControlIntervalFlag, 100*time.Millisecond, "How often the backlog is controlled.")
			o.Int(MaxConsecutiveErrorsFlag, 10, "Consecutive errors tolerated before stopping.")
			o.Duration(BacklogLogIntervalFlag, 30*time.Second, "How often backlog stats are logged.")
			o.Duration(VisibilityVerificationTimeoutFlag, 30*time.Second, "Timeout for the visibility count check.")
			o.String(SleepActivityJsonFlag, "", "JSON sleep-activity configuration; use @<file> to read from a file.")
			o.Float64(MinAddRateFlag, 0, "Minimum activities added per second; enables rate mode with max-add-rate.")
			o.Float64(MaxAddRateFlag, 0, "Maximum activities added per second; enables rate mode with min-add-rate.")
			o.Duration(RatePeriodFlag, 10*time.Minute, "Period of add-rate oscillation.")
			o.Duration(BacklogPeriodFlag, 0, "Period of backlog-size oscillation in rate mode; defaults to period.")
			o.Int(BatchSizeFlag, 0, "Maximum activities per spawned workflow; 0 means unlimited.")
		},
		ExecutorFn: func() loadgen.Executor { return newEbbAndFlowExecutor() },
	})
}

func newEbbAndFlowExecutor() *ebbAndFlowExecutor {
	return &ebbAndFlowExecutor{state: &ebbAndFlowState{}}
}

func (e *ebbAndFlowExecutor) Configure(info loadgen.ScenarioInfo) error {
	config := &ebbAndFlowConfig{
		MinBacklog:                    int64(info.OptionInt(MinBacklogFlag)),
		MaxBacklog:                    int64(info.OptionInt(MaxBacklogFlag)),
		Period:                        info.OptionDuration(PeriodFlag),
		SleepDuration:                 info.OptionDuration(SleepDurationFlag),
		MaxRate:                       int64(info.OptionInt(MaxRateFlag)),
		ControlInterval:               info.OptionDuration(ControlIntervalFlag),
		MaxConsecutiveErrors:          info.OptionInt(MaxConsecutiveErrorsFlag),
		BacklogLogInterval:            info.OptionDuration(BacklogLogIntervalFlag),
		VisibilityVerificationTimeout: info.OptionDuration(VisibilityVerificationTimeoutFlag),
		MinAddRate:                    info.OptionFloat64(MinAddRateFlag),
		MaxAddRate:                    info.OptionFloat64(MaxAddRateFlag),
		RatePeriod:                    info.OptionDuration(RatePeriodFlag),
		BacklogPeriod:                 info.OptionDuration(BacklogPeriodFlag),
		BatchSize:                     int64(info.OptionInt(BatchSizeFlag)),
	}

	if config.MinBacklog < 0 {
		return fmt.Errorf("min-backlog must be non-negative, got %d", config.MinBacklog)
	}

	if config.MaxBacklog <= config.MinBacklog {
		return fmt.Errorf("max-backlog must be greater than min-backlog, got max=%d min=%d", config.MaxBacklog, config.MinBacklog)
	}

	if config.Period <= 0 {
		return fmt.Errorf("period must be greater than 0, got %v", config.Period)
	}

	// Backlog oscillation falls back to the shared period when not set explicitly.
	if config.BacklogPeriod <= 0 {
		config.BacklogPeriod = config.Period
	}

	config.RateMode = config.MinAddRate > 0 && config.MaxAddRate > 0
	if config.RateMode && config.MaxAddRate <= config.MinAddRate {
		return fmt.Errorf("max-add-rate must be greater than min-add-rate, got max=%v min=%v", config.MaxAddRate, config.MinAddRate)
	}
	if config.RateMode && config.RatePeriod <= 0 {
		return fmt.Errorf("rate-period must be greater than 0, got %v", config.RatePeriod)
	}

	if sleepActivitiesStr := info.OptionString(SleepActivityJsonFlag); sleepActivitiesStr != "" {
		var err error
		// This scenario overrides "count" and "sleepDuration" so do not require them.
		config.SleepActivityConfig, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr, false, false)
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
	e.id = fmt.Sprintf("ebb_and_flow_%s", e.ExecutionID)
	e.rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	e.startTime = time.Now()

	// Get parsed configuration
	config := e.config
	if config == nil {
		return fmt.Errorf("configuration not parsed - Parse must be called before run")
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

	if config.RateMode {
		e.Logger.Infof("Starting ebb and flow scenario (rate mode): min_add_rate=%.1f, max_add_rate=%.1f, rate_period=%v, "+
			"min_backlog=%d, max_backlog=%d, backlog_period=%v, batch_size=%d, duration=%v",
			config.MinAddRate, config.MaxAddRate, config.RatePeriod,
			config.MinBacklog, config.MaxBacklog, config.BacklogPeriod,
			config.BatchSize, e.Configuration.Duration)
	} else {
		e.Logger.Infof("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, period=%v, duration=%v",
			config.MinBacklog, config.MaxBacklog, config.Period, e.Configuration.Duration)
	}

	e.RegisterDefaultSearchAttributes(ctx)

	var started, completed, backlog, target int64
	lastIteration := time.Now()

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

			backlogTarget := calculateSineTarget(
				elapsed, config.BacklogPeriod, float64(config.MinBacklog), float64(config.MaxBacklog))
			target = int64(math.Round(backlogTarget))
			deficit := math.Max(0, backlogTarget-float64(backlog))

			if config.RateMode {
				rateTarget := calculateSineTarget(elapsed, config.RatePeriod, config.MinAddRate, config.MaxAddRate)
				now := time.Now()
				rateActivities := rateTarget * now.Sub(lastIteration).Seconds()
				lastIteration = now
				deficit = math.Max(deficit, rateActivities)
			}
			toStart := int64(math.Round(deficit))
			toStart = min(toStart, config.MaxRate)
			for batch := range batches(toStart, config.BatchSize) {
				// iter is bound per batch: the closure must not observe the
				// increment below, which races with the spawned goroutine.
				iteration := iter
				startWG.Go(func() {
					errCh <- e.spawnWorkflowWithActivities(ctx, iteration, batch, config.SleepActivityConfig)
				})
				iter++
			}
		case <-backlogTicker.C:
			e.Logger.Infof("Backlog: %d, target: %d, started: %d, completed: %d",
				backlog, target, started, completed)
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
				loadgen.OmesExecutionIDSearchAttribute, e.ExecutionID),
		},
		totalCompletedWorkflows,
		config.VisibilityVerificationTimeout,
	); err != nil {
		return err
	}

	// Post-scenario: ensure there are no failed or terminated workflows for this run.
	return loadgen.VerifyNoFailedWorkflows(ctx, e.ScenarioInfo, loadgen.OmesExecutionIDSearchAttribute, e.ExecutionID)
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
	iteration, activities int64,
	template *loadgen.SleepActivityConfig,
) error {
	// Override activity count to fixed rate.
	fixedDist := loadgen.NewFixedDistribution(activities)
	config := loadgen.SleepActivityConfig{
		Count:  &fixedDist,
		Groups: template.Groups,
	}

	// Start workflow.
	run := e.NewRun(int(iteration))
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-track-%d", e.id, iteration)
	options.WorkflowExecutionErrorWhenAlreadyStarted = false
	options.TypedSearchAttributes = temporal.NewSearchAttributes(
		temporal.NewSearchAttributeKeyString(loadgen.OmesExecutionIDSearchAttribute).ValueSet(e.ExecutionID),
	)

	workflowInput := &ebbandflow.WorkflowParams{
		SleepActivities: &config,
	}

	// Start workflow to track activity timings.
	wf, err := e.Client.ExecuteWorkflow(ctx, options, "ebbAndFlowTrack", workflowInput)
	if err != nil {
		return fmt.Errorf("failed to start ebbAndFlowTrack workflow for iteration %d: %w", iteration, err)
	}
	e.scheduledActivities.Add(activities)

	// Wait for workflow completion
	var result ebbandflow.WorkflowOutput
	err = wf.Get(ctx, &result)
	if err != nil {
		e.Logger.Errorf("ebbAndFlowTrack workflow failed for iteration %d: %v", iteration, err)
	}
	e.completedActivities.Add(activities)
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

// calculateSineTarget oscillates between minVal and maxVal over the given
// period, starting at minVal.
func calculateSineTarget(elapsed, period time.Duration, minVal, maxVal float64) float64 {
	periods := elapsed.Seconds() / period.Seconds()
	osc := (math.Sin(2*math.Pi*(periods-0.25)) + 1.0) / 2
	return minVal + osc*(maxVal-minVal)
}

// batches yields batch sizes that partition total into chunks of at most
// batchSize. If batchSize <= 0, yields total as a single batch.
func batches(total, batchSize int64) iter.Seq[int64] {
	return func(yield func(int64) bool) {
		for total > 0 {
			batch := total
			if batchSize > 0 {
				batch = min(batch, batchSize)
			}
			total -= batch
			if !yield(batch) {
				return
			}
		}
	}
}
