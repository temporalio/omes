package scenarios

import (
	"context"
	"fmt"
	"hash/fnv"
	"maps"
	"math/rand"
	"time"

	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

// out_of_order_signals exercises signal delivery and workflow ordering by
// sending each workflow's N signals in a deterministically shuffled order
// while the workflow processes them strictly in event sequence.
//
// What this stresses: signal queuing, workflow buffering across signal
// arrivals, and forward progress when an early-expected event arrives last.
// Optionally inserts a "processing" activity between events to add
// per-event work.
//
// Per-iteration determinism: shuffle decisions and orderings are seeded from
// (RunID, iteration) so a given run is reproducible across re-executions.
//
// Implementation note: kitchen-sink's SetWorkflowState action replaces the
// workflow state map rather than merging into it (verified across all
// language workers). To accumulate event keys reliably regardless of arrival
// order, each signal carries the cumulative state through its send position;
// the workflow's awaits poll for keys appearing in the (growing) state map.

const (
	oooSignalsCountFlag             = "signals-per-workflow"
	oooSignalsShufflePercentageFlag = "shuffle-percentage"
	oooSignalsProcessingTimeFlag    = "processing-time-per-signal"
)

const oooSignalReadyValue = "ready"

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"Send N signals per workflow in a deterministically shuffled order; the workflow processes them in event sequence. "+
				"Options: '%s' (default 10), '%s' (default 100), '%s' (default 0).",
			oooSignalsCountFlag,
			oooSignalsShufflePercentageFlag,
			oooSignalsProcessingTimeFlag,
		),
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{},
					},
				},
				UpdateWorkflowOptions: func(_ context.Context, run *loadgen.Run, opts *loadgen.KitchenSinkWorkflowOptions) error {
					cfg, err := parseOutOfOrderSignalsConfig(run.ScenarioInfo)
					if err != nil {
						return err
					}
					rng := oooSeededRng(run.RunID, run.Iteration)
					opts.Params.WorkflowInput.InitialActions = buildOrderedAwaitActions(cfg.signalsPerWorkflow, cfg.processingTime)
					opts.Params.ClientSequence = buildShuffledSignals(cfg.signalsPerWorkflow, cfg.shufflePercentage, rng)
					return nil
				},
			}
		},
	})
}

type outOfOrderSignalsConfig struct {
	signalsPerWorkflow int
	shufflePercentage  int
	processingTime     time.Duration
}

func parseOutOfOrderSignalsConfig(info *loadgen.ScenarioInfo) (*outOfOrderSignalsConfig, error) {
	cfg := &outOfOrderSignalsConfig{
		signalsPerWorkflow: info.ScenarioOptionInt(oooSignalsCountFlag, 10),
		shufflePercentage:  info.ScenarioOptionInt(oooSignalsShufflePercentageFlag, 100),
		processingTime:     info.ScenarioOptionDuration(oooSignalsProcessingTimeFlag, 0),
	}
	if cfg.signalsPerWorkflow <= 0 {
		return nil, fmt.Errorf("%s must be > 0, got %d", oooSignalsCountFlag, cfg.signalsPerWorkflow)
	}
	if cfg.shufflePercentage < 0 || cfg.shufflePercentage > 100 {
		return nil, fmt.Errorf("%s must be in [0,100], got %d", oooSignalsShufflePercentageFlag, cfg.shufflePercentage)
	}
	if cfg.processingTime < 0 {
		return nil, fmt.Errorf("%s must be >= 0, got %v", oooSignalsProcessingTimeFlag, cfg.processingTime)
	}
	return cfg, nil
}

func oooSeededRng(runID string, iter int) *rand.Rand {
	h := fnv.New64a()
	_, _ = fmt.Fprintf(h, "%s/%d", runID, iter)
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

func buildOrderedAwaitActions(n int, processingTime time.Duration) []*kitchensink.ActionSet {
	actions := make([]*kitchensink.Action, 0, 2*n+1)
	processingStartToClose := durationpb.New(processingTime + time.Minute)
	for i := range n {
		actions = append(actions, kitchensink.NewAwaitWorkflowStateAction(oooEventKey(i+1), oooSignalReadyValue))
		if processingTime > 0 {
			actions = append(actions, &kitchensink.Action{
				Variant: &kitchensink.Action_ExecActivity{
					ExecActivity: &kitchensink.ExecuteActivityAction{
						ActivityType: &kitchensink.ExecuteActivityAction_Delay{
							Delay: durationpb.New(processingTime),
						},
						StartToCloseTimeout: processingStartToClose,
						Locality: &kitchensink.ExecuteActivityAction_Remote{
							Remote: &kitchensink.RemoteActivityOptions{},
						},
					},
				},
			})
		}
	}
	actions = append(actions, kitchensink.NewEmptyReturnResultAction())
	return []*kitchensink.ActionSet{{Actions: actions}}
}

func buildShuffledSignals(n int, shufflePct int, rng *rand.Rand) *kitchensink.ClientSequence {
	shuffle := shufflePct > 0 && rng.Intn(100) < shufflePct

	sendOrder := make([]int, n)
	for i := range sendOrder {
		sendOrder[i] = i + 1
	}
	if shuffle {
		rng.Shuffle(n, func(i, j int) {
			sendOrder[i], sendOrder[j] = sendOrder[j], sendOrder[i]
		})
	}

	cumulative := make(map[string]string, n)
	signals := make([]*kitchensink.ClientAction, n)
	for sendPos, eventID := range sendOrder {
		cumulative[oooEventKey(eventID)] = oooSignalReadyValue
		snapshot := make(map[string]string, len(cumulative))
		maps.Copy(snapshot, cumulative)
		signals[sendPos] = makeSetStateSignal(snapshot)
	}

	return &kitchensink.ClientSequence{
		ActionSets: []*kitchensink.ClientActionSet{
			{Actions: signals},
		},
	}
}

func makeSetStateSignal(kvs map[string]string) *kitchensink.ClientAction {
	return &kitchensink.ClientAction{
		Variant: &kitchensink.ClientAction_DoSignal{
			DoSignal: &kitchensink.DoSignal{
				Variant: &kitchensink.DoSignal_DoSignalActions_{
					DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
						Variant: &kitchensink.DoSignal_DoSignalActions_DoActions{
							DoActions: kitchensink.SingleActionSet(
								&kitchensink.Action{
									Variant: &kitchensink.Action_SetWorkflowState{
										SetWorkflowState: &kitchensink.WorkflowState{Kvs: kvs},
									},
								},
							),
						},
					},
				},
			},
		},
	}
}

func oooEventKey(i int) string {
	return fmt.Sprintf("event_%d", i)
}
