package scenarios

import (
	"context"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/types/known/durationpb"
	"strconv"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a number of child workflows and/or activities. " +
			"Additional options: children-per-workflow (default 30), activities-per-workflow (default 30).",
		Executor: loadgen.KitchenSinkExecutor{
			TestInput: &kitchensink.TestInput{
				WorkflowInput: &kitchensink.WorkflowInput{
					InitialActions: []*kitchensink.ActionSet{},
				},
			},
			PrepareTestInput: func(ctx context.Context, opts loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
				actionSet := &kitchensink.ActionSet{
					Actions: []*kitchensink.Action{},
					// We want these executed concurrently
					Concurrent: true,
				}
				params.WorkflowInput.InitialActions =
					append(params.WorkflowInput.InitialActions, actionSet)

				// Get options
				children := opts.ScenarioOptionInt("children-per-workflow", 30)
				activities := opts.ScenarioOptionInt("activities-per-workflow", 30)
				opts.Logger.Infof("Preparing to run with %v child workflow(s) and %v activity execution(s)", children, activities)

				childInput, err := converter.GetDefaultDataConverter().ToPayload(
					&kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							kitchensink.NoOpSingleActivityActionSet(),
						},
					})
				if err != nil {
					return err
				}
				// Add children
				for i := 0; i < children; i++ {
					actionSet.Actions = append(actionSet.Actions, &kitchensink.Action{
						Variant: &kitchensink.Action_ExecChildWorkflow{
							ExecChildWorkflow: &kitchensink.ExecuteChildWorkflowAction{
								WorkflowId:   opts.RunID + "-child-wf-" + strconv.Itoa(i),
								WorkflowType: "kitchenSink",
								Input:        []*common.Payload{childInput},
							},
						},
					})
				}

				// Add activities
				for i := 0; i < activities; i++ {
					actionSet.Actions = append(actionSet.Actions, &kitchensink.Action{
						Variant: &kitchensink.Action_ExecActivity{
							ExecActivity: &kitchensink.ExecuteActivityAction{
								ActivityType:        &kitchensink.ExecuteActivityAction_Noop{},
								StartToCloseTimeout: &durationpb.Duration{Seconds: 5},
							},
						},
					})
				}

				params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions,
					&kitchensink.ActionSet{
						Actions: []*kitchensink.Action{
							{
								Variant: &kitchensink.Action_ReturnResult{
									ReturnResult: &kitchensink.ReturnResultAction{
										ReturnThis: &common.Payload{},
									},
								},
							},
						},
					},
				)
				return nil
			},
		},
	})
}
