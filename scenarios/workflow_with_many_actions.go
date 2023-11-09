package scenarios

import (
	"context"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a number of child workflows and/or activities. " +
			"Additional options: children-per-workflow (default 30), activities-per-workflow (default 30).",
		Executor: loadgen.KitchenSinkExecutor{
			PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
				actionSet := &kitchensink.ActionSet{
					Actions: []*kitchensink.Action{},
					// We want these executed concurrently
					Concurrent: true,
				}
				params.WorkflowInput.InitialActions[0] = actionSet

				// Get options
				children := opts.ScenarioOptionInt("children-per-workflow", 30)
				activities := opts.ScenarioOptionInt("activities-per-workflow", 30)
				opts.Logger.Infof("Preparing to run with %v child workflow(s) and %v activity execution(s)", children, activities)

				// Add children
				if children > 0 {
					actionSet.Actions = append(actionSet.Actions, &kitchensink.Action{
						Variant: &kitchensink.Action_ExecChildWorkflow{
							ExecChildWorkflow: &kitchensink.ExecuteChildWorkflowAction{
								// TODO: Fill in
								WorkflowId:   "",
								WorkflowType: "",
							},
						},
					})
				}

				// Add activities
				if activities > 0 {
					actionSet.Actions = append(actionSet.Actions, &kitchensink.Action{
						Variant: &kitchensink.Action_ExecActivity{
							ExecActivity: &kitchensink.ExecuteActivityAction{
								ActivityType: "noop",
							},
						},
					})
				}
				return nil
			},
		},
	})
}
