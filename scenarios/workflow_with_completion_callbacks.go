package scenarios

import (
	"context"

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

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "For this scenario, Iterations is not supported and Duration is required. We run a single" +
			" iteration which will spawn a number of workflows, execute them, and verify that all callbacks are" +
			" eventually delivered.",
		Executor: &loadgen.GenericExecutor{
			Execute: func(ctx context.Context, run *loadgen.Run) error {
				//if run.Configuration.Iterations != 0 {
				//	return fmt.Errorf("iterations not supported")
				//}
				//duration := run.Configuration.Duration
				//if duration == 0 {
				//	return fmt.Errorf("duration required for this scenario")
				//}

				options := run.DefaultStartWorkflowOptions()
				completionCallbacks := []*commonpb.Callback{{
					Variant: &commonpb.Callback_Nexus_{
						Nexus: &commonpb.Callback_Nexus{
							Url: "http://localhost:8080/callbacks",
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
				execution, err := run.Client.DescribeWorkflowExecution(ctx, workflowRun.GetID(), workflowRun.GetRunID())
				if err != nil {
					return errors.Wrap(err, "failed to describe workflow")
				}
				callbacks := execution.Callbacks
				if len(callbacks) != len(completionCallbacks) {
					return errors.Errorf("expected %d callbacks, got %d", len(completionCallbacks), len(callbacks))
				}
				for _, callback := range callbacks {
					if callback.State != enums.CALLBACK_STATE_SUCCEEDED {
						return errors.Errorf("expected callback state to be SUCCEEDED, got %s", callback.State.String())
					}
				}
				return nil
			},
		},
	})
}
