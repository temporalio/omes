package scenarios

// Scenario: sleepy
//
// Port of the bench-go `sleepy` workload (see /workflows/sleepy/ in temporalio/temporal-bench).
//
// Each iteration starts a single kitchen-sink workflow that, for up to
// `max-continue-as-new-iterations + 1` runs, repeats the following loop:
//
//  1. Sleeps for a random number of seconds in
//     [min-workflow-sleep-duration-seconds, max-workflow-sleep-duration-seconds].
//  2. Sequentially executes `activities-per-run` activities that each delay for
//     `activity-duration-milliseconds` milliseconds.
//  3. If more iterations remain, continues-as-new with the input pre-built for
//     the next iteration; otherwise returns.
//
// The random sleep durations are pre-computed on the omes scenario driver and
// baked into the chain of TimerAction / ContinueAsNewAction protos. This differs
// from bench-go, which selected the sleep via `workflow.SideEffect(rand.Intn)`
// inside the workflow itself. The functional behavior (a random sleep per run
// within the configured bounds) is preserved.
//
// TODO(RE-294): `monitor-max-workflows-to-poll` is intentionally accepted but
// unused. In bench-go a sidecar "monitor" activity polled visibility for a
// random subset of the spawned workflow IDs to drive read-side load while the
// workflows ran. Omes scenarios don't have a direct analogue for a sidecar
// polling routine tied to a scenario run; if we want this in the future it
// should likely be modeled as either (a) a dedicated visibility-polling
// scenario, or (b) a goroutine spun up inside a custom Executor.Run
// implementation (see state_transitions_steady.go for that pattern). Until
// then the option is parsed and logged, but does not affect execution.

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"
	"time"

	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

const (
	sleepyMinWorkflowSleepDurationSecondsFlag = "min-workflow-sleep-duration-seconds"
	sleepyMaxWorkflowSleepDurationSecondsFlag = "max-workflow-sleep-duration-seconds"
	sleepyActivityDurationMillisecondsFlag    = "activity-duration-milliseconds"
	sleepyActivitiesPerRunFlag                = "activities-per-run"
	sleepyMaxContinueAsNewIterationsFlag      = "max-continue-as-new-iterations"
	sleepyMonitorMaxWorkflowsToPollFlag       = "monitor-max-workflows-to-poll"
)

type sleepyConfig struct {
	minWorkflowSleepDurationSeconds int
	maxWorkflowSleepDurationSeconds int
	activityDurationMilliseconds    int
	activitiesPerRun                int
	maxContinueAsNewIterations      int
	monitorMaxWorkflowsToPoll       int
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Port of the bench-go sleepy workload. Each iteration runs a workflow that " +
			"loops [sleep random[min,max] seconds -> run N activities of fixed duration -> " +
			"continue-as-new] up to max-continue-as-new-iterations times. " +
			"Options: " +
			sleepyMinWorkflowSleepDurationSecondsFlag + " (default 360), " +
			sleepyMaxWorkflowSleepDurationSecondsFlag + " (default 360), " +
			sleepyActivityDurationMillisecondsFlag + " (default 1000), " +
			sleepyActivitiesPerRunFlag + " (default 3), " +
			sleepyMaxContinueAsNewIterationsFlag + " (default 0), " +
			sleepyMonitorMaxWorkflowsToPollFlag + " (default 0, currently unused; see file comment).",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{},
					},
				},
				UpdateWorkflowOptions: func(_ context.Context, run *loadgen.Run, opts *loadgen.KitchenSinkWorkflowOptions) error {
					cfg, err := parseSleepyConfig(run.ScenarioInfo)
					if err != nil {
						return err
					}

					// Seed RNG per-iteration so concurrent iterations get distinct
					// sleep values but a given (RunID, Iteration) is reproducible.
					h := fnv.New64a()
					_, _ = h.Write([]byte(fmt.Sprintf("%s/%d", run.RunID, run.Iteration)))
					rng := rand.New(rand.NewSource(int64(h.Sum64())))

					opts.Params.WorkflowInput.InitialActions = buildSleepyActions(cfg, rng)
					return nil
				},
			}
		},
	})
}

func parseSleepyConfig(info *loadgen.ScenarioInfo) (*sleepyConfig, error) {
	cfg := &sleepyConfig{
		minWorkflowSleepDurationSeconds: info.ScenarioOptionInt(sleepyMinWorkflowSleepDurationSecondsFlag, 360),
		maxWorkflowSleepDurationSeconds: info.ScenarioOptionInt(sleepyMaxWorkflowSleepDurationSecondsFlag, 360),
		activityDurationMilliseconds:    info.ScenarioOptionInt(sleepyActivityDurationMillisecondsFlag, 1000),
		activitiesPerRun:                info.ScenarioOptionInt(sleepyActivitiesPerRunFlag, 3),
		maxContinueAsNewIterations:      info.ScenarioOptionInt(sleepyMaxContinueAsNewIterationsFlag, 0),
		monitorMaxWorkflowsToPoll:       info.ScenarioOptionInt(sleepyMonitorMaxWorkflowsToPollFlag, 0),
	}
	if cfg.minWorkflowSleepDurationSeconds < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", sleepyMinWorkflowSleepDurationSecondsFlag, cfg.minWorkflowSleepDurationSeconds)
	}
	if cfg.maxWorkflowSleepDurationSeconds < cfg.minWorkflowSleepDurationSeconds {
		return nil, fmt.Errorf("%s (%d) must be >= %s (%d)",
			sleepyMaxWorkflowSleepDurationSecondsFlag, cfg.maxWorkflowSleepDurationSeconds,
			sleepyMinWorkflowSleepDurationSecondsFlag, cfg.minWorkflowSleepDurationSeconds)
	}
	if cfg.activityDurationMilliseconds < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", sleepyActivityDurationMillisecondsFlag, cfg.activityDurationMilliseconds)
	}
	if cfg.activitiesPerRun < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", sleepyActivitiesPerRunFlag, cfg.activitiesPerRun)
	}
	if cfg.maxContinueAsNewIterations < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", sleepyMaxContinueAsNewIterationsFlag, cfg.maxContinueAsNewIterations)
	}
	if cfg.monitorMaxWorkflowsToPoll != 0 {
		info.Logger.Warnf(
			"%s is set to %d but is not implemented in the omes sleepy scenario; ignoring (see scenarios/sleepy.go for details)",
			sleepyMonitorMaxWorkflowsToPollFlag, cfg.monitorMaxWorkflowsToPoll,
		)
	}
	return cfg, nil
}

// buildSleepyActions builds the InitialActions for the first run of a sleepy
// workflow. The returned slice contains a single ActionSet that performs the
// first run's sleep + activities, then either returns (if no continue-as-new
// iterations remain) or continues-as-new with a pre-built WorkflowInput that
// recursively encodes the remaining runs.
func buildSleepyActions(cfg *sleepyConfig, rng *rand.Rand) []*kitchensink.ActionSet {
	// `remaining` is the number of continue-as-new transitions still to perform
	// AFTER the current run. bench-go's loop continues-as-new while
	// `iteration < MaxContinueAsNewIterations`, starting from iteration 0, so
	// the total number of runs is maxContinueAsNewIterations + 1.
	return buildSleepyActionsForRun(cfg, rng, cfg.maxContinueAsNewIterations)
}

// buildSleepyActionsForRun returns the action set for a single run of the
// sleepy workflow. If remaining > 0 the last action is a ContinueAsNewAction
// whose argument is the WorkflowInput for the next run; otherwise the last
// action is a ReturnResultAction.
func buildSleepyActionsForRun(cfg *sleepyConfig, rng *rand.Rand, remaining int) []*kitchensink.ActionSet {
	actions := make([]*kitchensink.Action, 0, cfg.activitiesPerRun+2)

	// 1. Sleep for a random duration in [min, max] seconds.
	sleepSeconds := cfg.minWorkflowSleepDurationSeconds
	span := cfg.maxWorkflowSleepDurationSeconds - cfg.minWorkflowSleepDurationSeconds + 1
	if span > 1 {
		sleepSeconds += rng.Intn(span)
	}
	actions = append(actions, &kitchensink.Action{
		Variant: &kitchensink.Action_Timer{
			Timer: &kitchensink.TimerAction{
				Milliseconds: uint64(sleepSeconds) * 1000,
			},
		},
	})

	// 2. Execute N activities, each delaying for activityDurationMilliseconds.
	activityDelay := durationpb.New(time.Duration(cfg.activityDurationMilliseconds) * time.Millisecond)
	// Match bench-go's StartToCloseTimeout: activity duration + 1 minute.
	startToClose := durationpb.New(
		time.Duration(cfg.activityDurationMilliseconds)*time.Millisecond + time.Minute,
	)
	for i := 0; i < cfg.activitiesPerRun; i++ {
		actions = append(actions, &kitchensink.Action{
			Variant: &kitchensink.Action_ExecActivity{
				ExecActivity: &kitchensink.ExecuteActivityAction{
					ActivityType: &kitchensink.ExecuteActivityAction_Delay{
						Delay: activityDelay,
					},
					StartToCloseTimeout: startToClose,
					Locality: &kitchensink.ExecuteActivityAction_Remote{
						Remote: &kitchensink.RemoteActivityOptions{},
					},
				},
			},
		})
	}

	// 3. Continue-as-new (with the next run's actions pre-baked) or return.
	if remaining > 0 {
		nextInput := &kitchensink.WorkflowInput{
			InitialActions: buildSleepyActionsForRun(cfg, rng, remaining-1),
		}
		actions = append(actions, &kitchensink.Action{
			Variant: &kitchensink.Action_ContinueAsNew{
				ContinueAsNew: &kitchensink.ContinueAsNewAction{
					Arguments: []*common.Payload{kitchensink.ConvertToPayload(nextInput)},
				},
			},
		})
	} else {
		actions = append(actions, kitchensink.NewEmptyReturnResultAction())
	}

	return []*kitchensink.ActionSet{
		{
			Actions:    actions,
			Concurrent: false,
		},
	}
}
