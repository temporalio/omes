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
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Oscillates backlog between min and max over frequency period using simple proportional control. " +
			"Options: min-backlog, max-backlog, frequency, sleep-duration, max-rate, control-interval, max-consecutive-errors, max-activities-per-workflow. Duration must be set.",
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

	startTime      time.Time
	generatedCount atomic.Int64
	processedCount atomic.Int64
	startWG        sync.WaitGroup
	errCh          chan error
	iter           int
}

func (e *ebbAndFlow) run(ctx context.Context) error {
	e.startTime = time.Now()
	e.errCh = make(chan error, 10000)
	e.iter = 1

	// Parse and validate scenario options.
	minBacklog := int64(e.ScenarioOptionInt("min-backlog", 10))
	maxBacklog := int64(e.ScenarioOptionInt("max-backlog", 30))
	frequency := e.ScenarioOptionDuration("frequency", 60*time.Second)
	sleepDuration := e.ScenarioOptionDuration("sleep-duration", 1*time.Millisecond)
	maxRate := e.ScenarioOptionInt("max-rate", 1000)
	controlInterval := e.ScenarioOptionDuration("control-interval", 100*time.Millisecond)
	maxConsecutiveErrors := e.ScenarioOptionInt("max-consecutive-errors", 10)
	batchSize := e.ScenarioOptionInt("max-activities-per-workflow", 1)

	if minBacklog < 0 {
		return fmt.Errorf("min-backlog must be non-negative")
	}
	if maxBacklog <= minBacklog {
		return fmt.Errorf("max-backlog must be greater than min-backlog")
	}
	if frequency <= 0 {
		return fmt.Errorf("frequency must be greater than 0")
	}
	if batchSize <= 0 {
		return fmt.Errorf("max-activities-per-workflow must be greater than 0")
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
	ticker := time.NewTicker(controlInterval)
	defer ticker.Stop()

	e.Logger.Debugf("Starting ebb and flow scenario: min_backlog=%d, max_backlog=%d, frequency=%v, duration=%v",
		minBacklog, maxBacklog, frequency, e.Configuration.Duration)

	isDraining := false // true = draining mode, false = growing mode
	cycleStartTime := e.startTime

	for elapsed := time.Duration(0); elapsed < e.Configuration.Duration; elapsed = time.Since(e.startTime) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-e.errCh:
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

			e.Logger.Infof("Backlog: %d, target: %d, rate: %d/s, gen: %d, proc: %d",
				backlog, target, rate, generated, processed)

			if rate > 0 {
				// Generate activities for this iteration
				fixedDist := loadgen.NewFixedDistribution(int64(rate))
				config := loadgen.SleepActivityConfig{
					Count:  &fixedDist,
					Groups: sleepActivityConfig.Groups,
				}
				activities := config.Sample(e.rng)

				e.spawnWorkflows(ctx, activities, batchSize)
				e.iter++
			}
		}
	}

	e.Logger.Info("Scenario complete; waiting for all workflows to finish...")
	e.startWG.Wait()
	return nil
}

func (e *ebbAndFlow) spawnWorkflows(
	ctx context.Context,
	activities []*kitchensink.ExecuteActivityAction,
	batchSize int,
) {
	// Batch activities into workflows
	workflowBatches := make([][]*kitchensink.ExecuteActivityAction, 0)
	for i := 0; i < len(activities); i += batchSize {
		end := i + batchSize
		if end > len(activities) {
			end = len(activities)
		}
		workflowBatches = append(workflowBatches, activities[i:end])
	}

	// Start one goroutine per batch
	for batchIndex, batch := range workflowBatches {
		e.startWG.Add(1)
		go func(iteration, batchIndex int, batch []*kitchensink.ExecuteActivityAction) {
			defer e.startWG.Done()
			e.errCh <- e.spawnWorkflow(ctx, iteration, batchIndex, batch)
		}(e.iter, batchIndex, batch)
	}
}

func (e *ebbAndFlow) spawnWorkflow(
	ctx context.Context,
	iteration, batchIndex int,
	batch []*kitchensink.ExecuteActivityAction,
) error {
	// Set retry policy for all activities in this batch.
	for _, activity := range batch {
		activity.RetryPolicy = &common.RetryPolicy{
			MaximumAttempts:    1,
			BackoffCoefficient: 1.0,
		}
	}

	// Create actions for all activities in this batch.
	actions := make([]*kitchensink.Action, 0, len(batch)+1)
	for _, activity := range batch {
		actions = append(actions, &kitchensink.Action{
			Variant: &kitchensink.Action_ExecActivity{
				ExecActivity: activity,
			},
		})
	}

	// Add return result action at the end.
	actions = append(actions, &kitchensink.Action{
		Variant: &kitchensink.Action_ReturnResult{
			ReturnResult: &kitchensink.ReturnResultAction{
				ReturnThis: &common.Payload{},
			},
		},
	})

	// Create action set with all activities in this batch.
	actionSet := &kitchensink.ActionSet{
		Actions:    actions,
		Concurrent: true,
	}

	// Start workflow.
	run := e.NewRun(iteration*10000 + batchIndex)
	options := run.DefaultStartWorkflowOptions()
	options.ID = fmt.Sprintf("%s-%d", options.ID, e.startTime.UnixMilli()) // avoid collision on a restart
	workflowInput := &kitchensink.WorkflowInput{InitialActions: []*kitchensink.ActionSet{actionSet}}
	wf, err := e.Client.ExecuteWorkflow(ctx, options, "kitchenSink", workflowInput)
	if err != nil {
		return fmt.Errorf("failed to start workflow for iteration %d batch %d: %w", iteration, batchIndex, err)
	}
	e.generatedCount.Add(1)

	// Wait for workflow completion.
	err = wf.Get(ctx, nil)
	if err != nil {
		e.Logger.Errorf("Workflow failed for iteration %d batch %d: %v", iteration, batchIndex, err)
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
