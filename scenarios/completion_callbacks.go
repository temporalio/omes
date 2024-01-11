package scenarios

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/temporalio/omes/common"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
)

// This file contains a scenario where each iteration executes a single workflow that has a completion callback attached
// targeting one of a given set of addresses. After each iteration, we query all workflows to verify that all callbacks
// have been delivered.

const (
	startingPortOptionKey     = "startingPort"
	numCallbackHostsOptionKey = "numCallbackHosts"
)

func ExponentialSample(n int, rate float64, sample float64) int {
	totalProbability := 1 - math.Exp(-rate*float64(n))
	for i := 1; i < n; i++ {
		cdf := 1 - math.Exp(-rate*float64(i))
		if sample <= cdf/totalProbability {
			return i - 1
		}
	}
	return n - 1
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "For this scenario, Iterations is not supported and Duration is required. We run a single" +
			" iteration which will spawn a number of workflows, execute them, and verify that all callbacks are" +
			" eventually delivered.",
		Executor: &loadgen.GenericExecutor{
			Execute: func(ctx context.Context, run *loadgen.Run) error {
				options := run.DefaultStartWorkflowOptions()

				startingPort := run.ScenarioOptionInt(startingPortOptionKey, 0)
				if startingPort == 0 {
					return errors.Errorf("%q is required", startingPortOptionKey)
				}

				numCallbackHosts := run.ScenarioOptionInt(numCallbackHostsOptionKey, 0)
				if numCallbackHosts == 0 {
					return errors.Errorf("%q is required", numCallbackHostsOptionKey)
				}

				completionCallbacks := []*commonpb.Callback{{
					Variant: &commonpb.Callback_Nexus_{
						Nexus: &commonpb.Callback_Nexus{
							Url: "http://localhost:9000",
						},
					},
				}}
				options.CompletionCallbacks = completionCallbacks
				input := &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{
						kitchensink.NoOpSingleActivityActionSet(),
					},
				}
				workflowRun, err := run.Client.ExecuteWorkflow(ctx, options, common.WorkflowNameKitchenSink, input)
				if err != nil {
					return errors.Wrap(err, "failed to execute workflow")
				}
				run.Logger.Info("Started workflow", "workflowID", workflowRun.GetID(), "runID", workflowRun.GetRunID())
				err = workflowRun.Get(ctx, nil)
				if err != nil {
					return errors.Wrap(err, "failed to get workflow result")
				}

				for {
					execution, err := run.Client.DescribeWorkflowExecution(ctx, workflowRun.GetID(), workflowRun.GetRunID())
					if err != nil {
						return errors.Wrap(err, "failed to describe workflow")
					}
					callbacks := execution.Callbacks
					if len(callbacks) != len(completionCallbacks) {
						return errors.Errorf("expected %d callbacks, got %d", len(completionCallbacks), len(callbacks))
					}
					allSucceeded := true
					anyFailed := false
					for _, callback := range callbacks {
						if callback.State != enums.CALLBACK_STATE_SUCCEEDED {
							allSucceeded = false
						}
						if callback.State == enums.CALLBACK_STATE_FAILED {
							anyFailed = true
							run.Logger.Error("Callback failed", "failure", callback.LastAttemptFailure)
						}
					}
					if anyFailed {
						return errors.New("one or more callbacks failed")
					}
					if allSucceeded {
						break
					}
					time.Sleep(time.Second)
				}
				return nil
			},
		},
	})
}
