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
			PrepareWorkflowParams: func(ctx context.Context, opts loadgen.RunOptions, params *kitchensink.WorkflowParams) error {
				// We want these executed concurrently
				params.ActionSet.Concurrent = true

				// Get options
				children := opts.ScenarioOptionInt("children-per-workflow", 30)
				activities := opts.ScenarioOptionInt("activities-per-workflow", 30)
				opts.Logger.Infof("Preparing to run with %v child workflow(s) and %v activity execution(s)", children, activities)

				// Add children
				if children > 0 {
					params.ActionSet.Actions = append(params.ActionSet.Actions, &kitchensink.Action{
						ExecuteChildWorkflow: &kitchensink.ExecuteChildWorkflow{
							Count: children,
						},
					})
				}

				// Add activities
				if activities > 0 {
					params.ActionSet.Actions = append(params.ActionSet.Actions, &kitchensink.Action{
						ExecuteActivity: &kitchensink.ExecuteActivityAction{
							Name:  kitchensink.NopActivityName,
							Count: activities,
						},
					})
				}
				return nil
			},
		},
	})
}
