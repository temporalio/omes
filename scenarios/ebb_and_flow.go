package scenarios

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
)

const maxConsecutiveErrors = 5

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Spawns activities to upper bound, drains to lower bound, rinse and repeat. " +
			"Options: min-activities, max-activities, activity-sleep. Duration must be set.",
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
	runningActivities atomic.Int64
	rng               *rand.Rand
}

func (e *ebbAndFlow) run(ctx context.Context) error {
	var sleepActivityConfig *loadgen.SleepActivityConfig
	if sleepActivitiesStr, ok := e.ScenarioOptions[SleepActivityJsonFlag]; ok {
		var err error
		sleepActivityConfig, err = loadgen.ParseAndValidateSleepActivityConfig(sleepActivitiesStr)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", SleepActivityJsonFlag, err)
		}
	}

	minActivities := e.ScenarioOptionInt("min-activities", 5)
	maxActivities := e.ScenarioOptionInt("max-activities", 25)
	spawnRatePerSec := e.ScenarioOptionInt("spaw-rate-per-sec", 5)

	if minActivities < 1 {
		return fmt.Errorf("min-activities must be at least 1")
	}
	if maxActivities < minActivities {
		return fmt.Errorf("max-activities must be greater than or equal to min-activities")
	}
	if spawnRatePerSec <= 0 {
		return fmt.Errorf("spawn-rate-per-sec must be greater than 0")
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var consecutiveErrCount int
	errCh := make(chan error, 10000)

	iter := 1
	var draining bool
	var startWG sync.WaitGroup

	e.Logger.Infof("Starting ebb and flow scenario: min=%d, max=%d, rate=%d/s, duration=%v",
		minActivities, maxActivities, spawnRatePerSec, e.Configuration.Duration)

	startedAt := time.Now()
	for elapsed := time.Duration(0); elapsed < e.Configuration.Duration; elapsed = time.Since(startedAt) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err == nil {
				consecutiveErrCount = 0
			} else {
				consecutiveErrCount++
				if consecutiveErrCount >= maxConsecutiveErrors {
					return fmt.Errorf("got %v consecutive errors, most recent: %w", maxConsecutiveErrors, err)
				}
			}
		case <-ticker.C:
			iter++

			// Check if we need to start spawning or draining.
			runningActivities := int(e.runningActivities.Load())
			fmt.Println(runningActivities)
			if runningActivities <= minActivities && draining {
				e.Logger.Infof("%d running activities: start spawning again", runningActivities)
				draining = false
			}
			if runningActivities >= maxActivities && !draining {
				e.Logger.Infof("%d running activities: stop spawning, start draining", runningActivities)
				draining = true
			}
			if draining {
				continue
			}

			startWG.Add(1)
			go func(iteration int) {
				defer startWG.Done()
				err := e.spawnWorkflowWithActivities(ctx, iteration, spawnRatePerSec, sleepActivityConfig)
				select {
				case errCh <- err:
				default:
				}
			}(iter)
		}
	}

	e.Logger.Infof("Scenario complete, ran %d iterations, waiting for all workflow starts to complete", iter)
	startWG.Wait()

	return nil
}

func (e *ebbAndFlow) spawnWorkflowWithActivities(
	ctx context.Context,
	iteration, count int,
	sleepActivityConfig *loadgen.SleepActivityConfig,
) error {
	e.runningActivities.Add(int64(count))
	defer e.runningActivities.Add(-int64(count))

	actionSet := &kitchensink.ActionSet{
		Actions:    []*kitchensink.Action{},
		Concurrent: true,
	}

	// Generate activities using SleepActivityConfig.Sample()
	activities := sleepActivityConfig.Sample(e.rng)
	for _, activity := range activities {
		activity.RetryPolicy = &common.RetryPolicy{
			MaximumAttempts:    1,
			BackoffCoefficient: 1.0,
		}
		actionSet.Actions = append(actionSet.Actions,
			&kitchensink.Action{
				Variant: &kitchensink.Action_ExecActivity{
					ExecActivity: activity,
				},
			})
	}

	workflowInput := &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			actionSet,
			{
				Actions: []*kitchensink.Action{
					{
						Variant: &kitchensink.Action_ReturnResult{
							ReturnResult: &kitchensink.ReturnResultAction{
								ReturnThis: &common.Payload{},
							},
						},
					},
				},
			},
		},
	}

	run := e.NewRun(iteration)
	wf, err := e.Client.ExecuteWorkflow(ctx, run.DefaultStartWorkflowOptions(), "kitchenSink", workflowInput)
	if err != nil {
		return fmt.Errorf("failed to start workflow for iteration %d: %w", iteration, err)
	}
	return wf.Get(ctx, nil)
}
