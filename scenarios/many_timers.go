package scenarios

// Scenario: many_timers
//
// Ports the bench-go `manytimers` workload (RE-295) to omes.
//
// Each workflow iteration runs `concurrent-timers-per-workflow` parallel timers,
// each lasting `time-duration-seconds`. This is repeated `iterations` times
// sequentially within a single workflow execution. The omes analog of bench-go's
// `ConcurrentWorkflows` driver knob is the standard omes flag `--max-concurrent`
// — this scenario also accepts a `concurrent-workflows` scenario option for
// bench-go knob parity, but it is informational only (a warning is emitted if
// it does not match `--max-concurrent`). Users should pass
// `--max-concurrent <N>` to actually parallelize workflow starts.
//
// Burst mode (`burst=true`): bench-go's burst variant schedules timers so they
// all fire near-simultaneously (it adjusts each timer's duration based on a
// shared deadline) and forces iterations=1. In this first-cut port, burst mode
// is approximated by running a single iteration of N parallel timers of the
// same duration — kitchen-sink does not currently support a per-timer absolute
// "fire-at-this-time" deadline like bench-go does. The "stagger so they all
// fire together" behavior is documented as a TODO; the practical effect of
// "many timers waiting and firing at roughly the same time" is preserved
// because all timers in the ActionSet start at workflow start.
//
// Knobs (kebab-case per omes convention):
//
//	concurrent-timers-per-workflow  int  (default: 5)   timers per parallel batch
//	concurrent-workflows            int  (default: 0)   if > 0, overrides --max-concurrent
//	iterations                      int  (default: 1)   sequential repeats per workflow
//	time-duration-seconds           int  (default: 10)  per-timer duration in seconds
//	burst                           bool (default: false) burst mode (forces iterations=1)
//
// Dropped from bench-go ManyTimersParameters:
//   - `Type` "continuous" (continue-as-new variant) — not ported in first cut; TODO.
//   - `TimerFireDeadline` — internal bench-go field; not user-facing.

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
)

const (
	mtConcurrentTimersPerWorkflowFlag = "concurrent-timers-per-workflow"
	mtConcurrentWorkflowsFlag         = "concurrent-workflows"
	mtIterationsFlag                  = "iterations"
	mtTimeDurationSecondsFlag         = "time-duration-seconds"
	mtBurstFlag                       = "burst"
)

const (
	mtDefaultConcurrentTimersPerWorkflow = 5
	mtDefaultIterations                  = 1
	mtDefaultTimeDurationSeconds         = 10
	mtDefaultBurst                       = false
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Many-timers workload (ported from bench-go). Each workflow runs '%s' parallel timers of "+
				"'%s' seconds, repeated '%s' times. '%s' is accepted for bench-go knob parity but "+
				"users should pass --max-concurrent to actually parallelize workflow starts. "+
				"'%s'=true approximates bench-go burst mode (forces iterations=1).",
			mtConcurrentTimersPerWorkflowFlag,
			mtTimeDurationSecondsFlag,
			mtIterationsFlag,
			mtConcurrentWorkflowsFlag,
			mtBurstFlag,
		),
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &TestInput{
					WorkflowInput: &WorkflowInput{
						InitialActions: []*ActionSet{},
					},
				},
				PrepareTestInput: func(_ context.Context, info loadgen.ScenarioInfo, params *TestInput) error {
					concurrentTimers := info.ScenarioOptionInt(mtConcurrentTimersPerWorkflowFlag, mtDefaultConcurrentTimersPerWorkflow)
					concurrentWorkflows := info.ScenarioOptionInt(mtConcurrentWorkflowsFlag, 0)
					iterations := info.ScenarioOptionInt(mtIterationsFlag, mtDefaultIterations)
					timerSeconds := info.ScenarioOptionInt(mtTimeDurationSecondsFlag, mtDefaultTimeDurationSeconds)
					burst := info.ScenarioOptionBool(mtBurstFlag, mtDefaultBurst)

					if concurrentTimers <= 0 {
						return fmt.Errorf("%s must be positive, got %d", mtConcurrentTimersPerWorkflowFlag, concurrentTimers)
					}
					if iterations <= 0 {
						return fmt.Errorf("%s must be positive, got %d", mtIterationsFlag, iterations)
					}
					if timerSeconds <= 0 {
						return fmt.Errorf("%s must be positive, got %d", mtTimeDurationSecondsFlag, timerSeconds)
					}

					// In bench-go, burst mode forces iterations=1.
					if burst && iterations != 1 {
						info.Logger.Warnf(
							"%s=true forces %s to 1 (was %d) to match bench-go semantics",
							mtBurstFlag, mtIterationsFlag, iterations,
						)
						iterations = 1
					}

					// NOTE: `concurrent-workflows` is accepted for bench-go knob parity but
					// cannot mutate scenario configuration here (ScenarioInfo is passed by
					// value into the executor). Users should additionally pass
					// `--max-concurrent <N>` to actually parallelize workflow starts.
					// We log a hint if the user supplied it without --max-concurrent matching.
					if concurrentWorkflows > 0 && concurrentWorkflows != info.Configuration.MaxConcurrent {
						info.Logger.Warnf(
							"%s=%d but --max-concurrent=%d; pass --max-concurrent=%d to actually run that many workflows in parallel",
							mtConcurrentWorkflowsFlag, concurrentWorkflows,
							info.Configuration.MaxConcurrent, concurrentWorkflows,
						)
					}

					timerDuration := time.Duration(timerSeconds) * time.Second

					// Build N parallel timer actions.
					buildTimerBatch := func() *ActionSet {
						actions := make([]*Action, concurrentTimers)
						for i := range actions {
							actions[i] = NewTimerAction(timerDuration)
						}
						return &ActionSet{Actions: actions, Concurrent: true}
					}

					// Compose `iterations` sequential batches of concurrent timers, then return.
					// Each ActionSet in InitialActions runs sequentially; within each set,
					// Concurrent=true makes the N timers fire in parallel.
					actionSets := make([]*ActionSet, 0, iterations+1)
					for i := 0; i < iterations; i++ {
						actionSets = append(actionSets, buildTimerBatch())
					}
					actionSets = append(actionSets, &ActionSet{
						Actions: []*Action{NewEmptyReturnResultAction()},
					})

					params.WorkflowInput.InitialActions = actionSets

					// TODO(RE-295): Implement bench-go "continuous" variant via ContinueAsNewAction.
					// TODO(RE-295): True burst semantics — adjust per-timer durations so they all
					//   fire at the same wall-clock instant — requires either a kitchen-sink
					//   absolute-deadline timer action or per-timer per-iteration duration support.
					return nil
				},
			}
		},
	})
}
