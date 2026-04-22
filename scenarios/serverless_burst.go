package scenarios

// Scenario: serverless_burst
//
// Designed to stress-test auto-scaling of serverless (Lambda) workers under burst workloads.
//
// Each iteration executes one workflow whose activities model two common real-world patterns:
//   - Fast I/O-bound tasks (noop): these complete quickly and test how rapidly Lambda workers
//     can be invoked and become productive from zero.
//   - Slow compute-bound tasks (delay): these hold workers for a configurable duration and
//     reveal saturation issues and Lambda timeout sensitivity.
//
// Both groups run concurrently within each workflow, so a single workflow exercises the
// worker on both ends of the latency spectrum simultaneously.
//
// Burst behavior is controlled by the standard --max-concurrent flag (burst width) and
// --iterations / --duration (burst volume). Pair this scenario with a short or zero pre-burst
// idle period (provided externally, e.g. by running with scale-to-zero in between) to observe
// how quickly the WCI can invoke enough Lambda workers to drain the resulting backlog.
//
// Options:
//
//	fast-activity-count   number of concurrent noop (I/O-bound) activities per workflow (default: 5)
//	slow-activity-count   number of concurrent delay (compute-bound) activities per workflow (default: 2)
//	slow-activity-duration duration of each slow activity, e.g. "5s", "1m" (default: 2s)

import (
	"context"
	"time"

	. "github.com/temporalio/omes/loadgen/kitchensink"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
)

const (
	sbFastActivityCountFlag = "fast-activity-count"
	sbSlowActivityCountFlag = "slow-activity-count"
	sbSlowActivityDuration  = "slow-activity-duration"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Burst workload for serverless workers: each iteration runs a workflow with " +
			"concurrent fast (noop) and slow (delay) activities. Tests auto-scale speed from zero " +
			"and sustained throughput under mixed I/O and compute load. " +
			"Options: " + sbFastActivityCountFlag + " (default 5), " +
			sbSlowActivityCountFlag + " (default 2), " +
			sbSlowActivityDuration + " (default 2s).",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &TestInput{
					WorkflowInput: &WorkflowInput{
						InitialActions: []*ActionSet{},
					},
				},
				PrepareTestInput: func(_ context.Context, info loadgen.ScenarioInfo, params *TestInput) error {
					fastCount := info.ScenarioOptionInt(sbFastActivityCountFlag, 5)
					slowCount := info.ScenarioOptionInt(sbSlowActivityCountFlag, 2)
					slowDuration := info.ScenarioOptionDuration(sbSlowActivityDuration, 2*time.Second)

					// StartToCloseTimeout must exceed the activity's own sleep duration.
					slowTimeout := max(60*time.Second, slowDuration*2)

					fastActions := make([]*Action, fastCount)
					for i := range fastActions {
						fastActions[i] = GenericActivity("noop", DefaultRemoteActivity)
					}

					slowActions := make([]*Action, slowCount)
					for i := range slowActions {
						slowActions[i] = DelayActivity(slowDuration, func(a *ExecuteActivityAction) *Action {
							a.StartToCloseTimeout = durationpb.New(slowTimeout)
							a.Locality = &ExecuteActivityAction_Remote{Remote: &RemoteActivityOptions{}}
							return &Action{Variant: &Action_ExecActivity{ExecActivity: a}}
						})
					}

					params.WorkflowInput.InitialActions = []*ActionSet{
						// Fast I/O-bound activities — all fire concurrently.
						{Actions: fastActions, Concurrent: true},
						// Slow compute-bound activities — all fire concurrently after fast ones complete.
						{Actions: slowActions, Concurrent: true},
						{Actions: []*Action{NewEmptyReturnResultAction()}},
					}
					return nil
				},
			}
		},
	})
}
