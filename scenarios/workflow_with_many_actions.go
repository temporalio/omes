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
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
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
					schedules := opts.ScenarioOptionInt("schedules-per-workflow", 10)
					opts.Logger.Infof("Preparing to run with %v child workflow(s), %v activity execution(s), and %v schedule operation(s)", children, activities, schedules)

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

					// Add schedule operations in 5 phases
					// Phase 1: Create schedules concurrently
					createScheduleActions := &kitchensink.ActionSet{
						Actions:    []*kitchensink.Action{},
						Concurrent: true,
					}
					for i := 0; i < schedules; i++ {
						scheduleID := opts.RunID + "-schedule-" + strconv.Itoa(i)
						createScheduleActions.Actions = append(createScheduleActions.Actions, &kitchensink.Action{
							Variant: &kitchensink.Action_CreateSchedule{
								CreateSchedule: &kitchensink.CreateScheduleAction{
									ScheduleId: scheduleID,
									Spec: &kitchensink.ScheduleSpec{
										CronExpressions: []string{"0 0 * * *"}, // Fires daily at midnight
									},
									Action: &kitchensink.ScheduleAction{
										WorkflowId:   scheduleID + "-wf",
										WorkflowType: "kitchenSink",
									},
									Policies: &kitchensink.SchedulePolicies{
										RemainingActions:    1,    // Only run once if triggered
										TriggerImmediately: true, // Fire immediately when created
									},
								},
							},
						})
					}
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, createScheduleActions)

					// Phase 2: Describe schedules concurrently
					describeScheduleActions1 := &kitchensink.ActionSet{
						Actions:    []*kitchensink.Action{},
						Concurrent: true,
					}
					for i := 0; i < schedules; i++ {
						scheduleID := opts.RunID + "-schedule-" + strconv.Itoa(i)
						describeScheduleActions1.Actions = append(describeScheduleActions1.Actions, &kitchensink.Action{
							Variant: &kitchensink.Action_DescribeSchedule{
								DescribeSchedule: &kitchensink.DescribeScheduleAction{
									ScheduleId: scheduleID,
								},
							},
						})
					}
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, describeScheduleActions1)

					// Phase 3: Update schedules concurrently
					updateScheduleActions := &kitchensink.ActionSet{
						Actions:    []*kitchensink.Action{},
						Concurrent: true,
					}
					for i := 0; i < schedules; i++ {
						scheduleID := opts.RunID + "-schedule-" + strconv.Itoa(i)
						updateScheduleActions.Actions = append(updateScheduleActions.Actions, &kitchensink.Action{
							Variant: &kitchensink.Action_UpdateSchedule{
								UpdateSchedule: &kitchensink.UpdateScheduleAction{
									ScheduleId: scheduleID,
									Spec: &kitchensink.ScheduleSpec{
										CronExpressions: []string{"0 12 1 1 *"}, // Changed cron (Jan 1st noon)
									},
								},
							},
						})
					}
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, updateScheduleActions)

					// Phase 4: Describe schedules again concurrently (verify update)
					describeScheduleActions2 := &kitchensink.ActionSet{
						Actions:    []*kitchensink.Action{},
						Concurrent: true,
					}
					for i := 0; i < schedules; i++ {
						scheduleID := opts.RunID + "-schedule-" + strconv.Itoa(i)
						describeScheduleActions2.Actions = append(describeScheduleActions2.Actions, &kitchensink.Action{
							Variant: &kitchensink.Action_DescribeSchedule{
								DescribeSchedule: &kitchensink.DescribeScheduleAction{
									ScheduleId: scheduleID,
								},
							},
						})
					}
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, describeScheduleActions2)

					// Phase 5: Delete schedules sequentially (for cleanup)
					deleteScheduleActions := &kitchensink.ActionSet{
						Actions:    []*kitchensink.Action{},
						Concurrent: false, // Sequential to ensure cleanup
					}
					for i := 0; i < schedules; i++ {
						scheduleID := opts.RunID + "-schedule-" + strconv.Itoa(i)
						deleteScheduleActions.Actions = append(deleteScheduleActions.Actions, &kitchensink.Action{
							Variant: &kitchensink.Action_DeleteSchedule{
								DeleteSchedule: &kitchensink.DeleteScheduleAction{
									ScheduleId: scheduleID,
								},
							},
						})
					}
					params.WorkflowInput.InitialActions = append(params.WorkflowInput.InitialActions, deleteScheduleActions)

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
			}
		},
	})
}
