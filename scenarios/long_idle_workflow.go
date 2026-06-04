package scenarios

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

// long_idle_workflow exercises Temporal under fleets of long-lived, mostly
// idle workflows. Each workflow alternates an idle period (a timer) with an
// optional burst of activities, repeated `cycles-per-run` times per run, and
// can chain `continue-as-new-iterations` follow-on runs.
//
// What this stresses: history retention for long-lived executions, timer
// scheduling at low duty cycle, and the workflow fleet's steady-state cost.
// Total fleet size is controlled by the standard `--max-concurrent` /
// `--iterations` / `--duration` flags.

const (
	longIdleIdleDurationFlag            = "idle-duration"
	longIdleCyclesPerRunFlag            = "cycles-per-run"
	longIdleActivitiesPerCycleFlag      = "activities-per-cycle"
	longIdleActivityDurationFlag        = "activity-duration"
	longIdleContinueAsNewIterationsFlag = "continue-as-new-iterations"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Run long-lived, mostly idle workflows that alternate idle periods with optional activity bursts. "+
				"Options: '%s' (default 1m), '%s' (default 1), '%s' (default 0), '%s' (default 1s), '%s' (default 0).",
			longIdleIdleDurationFlag,
			longIdleCyclesPerRunFlag,
			longIdleActivitiesPerCycleFlag,
			longIdleActivityDurationFlag,
			longIdleContinueAsNewIterationsFlag,
		),
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{},
					},
				},
				PrepareTestInput: func(_ context.Context, info loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
					cfg, err := parseLongIdleConfig(&info)
					if err != nil {
						return err
					}
					params.WorkflowInput.InitialActions = buildLongIdleActions(cfg, cfg.continueAsNewIterations)
					return nil
				},
			}
		},
	})
}

type longIdleConfig struct {
	idleDuration            time.Duration
	cyclesPerRun            int
	activitiesPerCycle      int
	activityDuration        time.Duration
	continueAsNewIterations int
}

func parseLongIdleConfig(info *loadgen.ScenarioInfo) (*longIdleConfig, error) {
	cfg := &longIdleConfig{
		idleDuration:            info.ScenarioOptionDuration(longIdleIdleDurationFlag, time.Minute),
		cyclesPerRun:            info.ScenarioOptionInt(longIdleCyclesPerRunFlag, 1),
		activitiesPerCycle:      info.ScenarioOptionInt(longIdleActivitiesPerCycleFlag, 0),
		activityDuration:        info.ScenarioOptionDuration(longIdleActivityDurationFlag, time.Second),
		continueAsNewIterations: info.ScenarioOptionInt(longIdleContinueAsNewIterationsFlag, 0),
	}
	if cfg.idleDuration <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %v", longIdleIdleDurationFlag, cfg.idleDuration)
	}
	if cfg.cyclesPerRun <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %d", longIdleCyclesPerRunFlag, cfg.cyclesPerRun)
	}
	if cfg.activitiesPerCycle < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", longIdleActivitiesPerCycleFlag, cfg.activitiesPerCycle)
	}
	if cfg.activityDuration < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %v", longIdleActivityDurationFlag, cfg.activityDuration)
	}
	if cfg.continueAsNewIterations < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %d", longIdleContinueAsNewIterationsFlag, cfg.continueAsNewIterations)
	}
	return cfg, nil
}

// buildLongIdleActions composes the InitialActions for one workflow run. When
// remainingCAN > 0 the run ends with a ContinueAsNewAction whose argument is
// the pre-baked WorkflowInput for the next run; otherwise it ends with a
// ReturnResult.
func buildLongIdleActions(cfg *longIdleConfig, remainingCAN int) []*kitchensink.ActionSet {
	actions := make([]*kitchensink.Action, 0, cfg.cyclesPerRun*(1+cfg.activitiesPerCycle)+1)
	activityStartToClose := durationpb.New(cfg.activityDuration + time.Minute)
	for range cfg.cyclesPerRun {
		actions = append(actions, kitchensink.NewTimerAction(cfg.idleDuration))
		for range cfg.activitiesPerCycle {
			actions = append(actions, &kitchensink.Action{
				Variant: &kitchensink.Action_ExecActivity{
					ExecActivity: &kitchensink.ExecuteActivityAction{
						ActivityType: &kitchensink.ExecuteActivityAction_Delay{
							Delay: durationpb.New(cfg.activityDuration),
						},
						StartToCloseTimeout: activityStartToClose,
						Locality: &kitchensink.ExecuteActivityAction_Remote{
							Remote: &kitchensink.RemoteActivityOptions{},
						},
					},
				},
			})
		}
	}

	if remainingCAN > 0 {
		nextInput := &kitchensink.WorkflowInput{
			InitialActions: buildLongIdleActions(cfg, remainingCAN-1),
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

	return []*kitchensink.ActionSet{{Actions: actions}}
}
