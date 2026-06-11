package scenarios

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

// workflow_with_many_timers exercises the timer subsystem with workflows that
// hold many concurrent active timers. Each workflow runs `iterations`
// sequential batches; within each batch, `concurrent-timers` timers fire in
// parallel. Each timer fires after `timer-duration`, optionally jittered by a
// random offset in [0, `timer-duration-jitter`).
//
// What this stresses: timer scheduling and firing under high per-workflow
// timer cardinality, and the worker's ability to manage many active timer
// futures concurrently. With zero jitter all timers share one deadline and
// fire as a single burst (thundering herd); a non-zero jitter spreads the
// deadlines so the timer queue holds many distinct fire times and the workflow
// wakes repeatedly as they trickle in.

const (
	manyTimersConcurrentTimersFlag = "concurrent-timers"
	manyTimersTimerDurationFlag    = "timer-duration"
	manyTimersTimerJitterFlag      = "timer-duration-jitter"
	manyTimersIterationsFlag       = "iterations"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Run workflows that hold many concurrent active timers. "+
				"Options: '%s' (default 30), '%s' (default 10s), '%s' (default 0), '%s' (default 1).",
			manyTimersConcurrentTimersFlag,
			manyTimersTimerDurationFlag,
			manyTimersTimerJitterFlag,
			manyTimersIterationsFlag,
		),
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{},
					},
				},
				PrepareTestInput: func(_ context.Context, info loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
					cfg, err := parseManyTimersConfig(&info)
					if err != nil {
						return err
					}
					params.WorkflowInput.InitialActions = buildManyTimersActions(cfg, manyTimersSeededRng(info.RunID))
					return nil
				},
			}
		},
	})
}

type manyTimersConfig struct {
	concurrentTimers int
	timerDuration    time.Duration
	timerJitter      time.Duration
	iterations       int
}

func parseManyTimersConfig(info *loadgen.ScenarioInfo) (*manyTimersConfig, error) {
	cfg := &manyTimersConfig{
		concurrentTimers: info.ScenarioOptionInt(manyTimersConcurrentTimersFlag, 30),
		timerDuration:    info.ScenarioOptionDuration(manyTimersTimerDurationFlag, 10*time.Second),
		timerJitter:      info.ScenarioOptionDuration(manyTimersTimerJitterFlag, 0),
		iterations:       info.ScenarioOptionInt(manyTimersIterationsFlag, 1),
	}
	if cfg.concurrentTimers <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %d", manyTimersConcurrentTimersFlag, cfg.concurrentTimers)
	}
	if cfg.timerDuration <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %v", manyTimersTimerDurationFlag, cfg.timerDuration)
	}
	if cfg.timerJitter < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %v", manyTimersTimerJitterFlag, cfg.timerJitter)
	}
	if cfg.iterations <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %d", manyTimersIterationsFlag, cfg.iterations)
	}
	return cfg, nil
}

// manyTimersSeededRng derives a deterministic RNG from the run ID so a given
// run produces the same jittered timer durations across re-executions.
func manyTimersSeededRng(runID string) *rand.Rand {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%s", runID)
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

// buildManyTimersActions composes the InitialActions for one workflow run: one
// concurrent batch of `concurrentTimers` timers per iteration, followed by a
// terminal ReturnResult. Each timer fires after `timerDuration`; when
// `timerJitter` is positive, a random offset in [0, timerJitter) is added so
// the timers in a batch fire at distinct times. rng is only consumed when
// jitter is positive.
func buildManyTimersActions(cfg *manyTimersConfig, rng *rand.Rand) []*kitchensink.ActionSet {
	sets := make([]*kitchensink.ActionSet, 0, cfg.iterations+1)
	for range cfg.iterations {
		batch := make([]*kitchensink.Action, cfg.concurrentTimers)
		for j := range batch {
			d := cfg.timerDuration
			if cfg.timerJitter > 0 {
				d += time.Duration(rng.Int63n(int64(cfg.timerJitter)))
			}
			batch[j] = kitchensink.NewTimerAction(d)
		}
		sets = append(sets, &kitchensink.ActionSet{
			Actions:    batch,
			Concurrent: true,
		})
	}
	sets = append(sets, &kitchensink.ActionSet{
		Actions: []*kitchensink.Action{kitchensink.NewEmptyReturnResultAction()},
	})
	return sets
}
