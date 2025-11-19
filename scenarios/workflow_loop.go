package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

const (
	// ActivityCountFlag controls the number of activities to execute sequentially
	ActivityCountFlag = "activity-count"
	// UseUpdateFlag controls whether to use update instead of signal (default: false, use signal)
	UseUpdateFlag = "use-update"
)

type workflowLoopExecutor struct {
	*loadgen.KitchenSinkExecutor
	completionVerifier *loadgen.WorkflowCompletionVerifier
}

func (e *workflowLoopExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Create completion verifier
	verifier, err := loadgen.NewWorkflowCompletionChecker(ctx, info, 30*time.Second)
	if err != nil {
		return err
	}
	e.completionVerifier = verifier

	// Run the kitchen sink executor
	return e.KitchenSinkExecutor.Run(ctx, info)
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf("Creates n activities sequentially, each activity sends one signal or update back to the workflow. "+
			"The workflow waits for each signal/update before proceeding. "+
			"Use --option %s=<number> to set the activity count (default: 1). "+
			"Use --option %s=true to use updates instead of signals (default: false).",
			ActivityCountFlag, UseUpdateFlag),
		ExecutorFn: func() loadgen.Executor {
			return &workflowLoopExecutor{
				KitchenSinkExecutor: &loadgen.KitchenSinkExecutor{
					TestInput: &kitchensink.TestInput{
						WorkflowInput: &kitchensink.WorkflowInput{},
					},
					PrepareTestInput: func(ctx context.Context, info loadgen.ScenarioInfo, params *kitchensink.TestInput) error {
						activityCount := info.ScenarioOptionInt(ActivityCountFlag, 1)
						if activityCount <= 0 {
							return fmt.Errorf("%s must be positive, got %d", ActivityCountFlag, activityCount)
						}

						useUpdate := info.ScenarioOptions[UseUpdateFlag] == "true"
						mechanism := "signal"
						if useUpdate {
							mechanism = "update"
						}
						info.Logger.Infof("Preparing workflow loop with %d activities using %s", activityCount, mechanism)

						// Create actions for the workflow
						var actions []*kitchensink.Action

						// Use a single state variable "loop-index" that tracks the current index
						// This ensures signals/updates are processed consecutively in order
						const stateKey = "loop-index"

						// For each iteration, create a sequential action that:
						// 1. Executes an activity that sends a signal or update back to the workflow
						// 2. Waits for the workflow state to be set to the current index by that signal/update
						for i := 0; i < activityCount; i++ {
							stateValue := fmt.Sprintf("%d", i)

							// Create the activity that sends signal/update to set the workflow state to the current index
							var clientAction *kitchensink.ClientAction
							if useUpdate {
								// Use update
								clientAction = &kitchensink.ClientAction{
									Variant: &kitchensink.ClientAction_DoUpdate{
										DoUpdate: &kitchensink.DoUpdate{
											Variant: &kitchensink.DoUpdate_DoActions{
												DoActions: &kitchensink.DoActionsUpdate{
													Variant: &kitchensink.DoActionsUpdate_DoActions{
														DoActions: &kitchensink.ActionSet{
															Actions: []*kitchensink.Action{
																kitchensink.NewSetWorkflowStateAction(stateKey, stateValue),
															},
														},
													},
												},
											},
										},
									},
								}
							} else {
								// Use signal
								clientAction = &kitchensink.ClientAction{
									Variant: &kitchensink.ClientAction_DoSignal{
										DoSignal: &kitchensink.DoSignal{
											Variant: &kitchensink.DoSignal_DoSignalActions_{
												DoSignalActions: &kitchensink.DoSignal_DoSignalActions{
													Variant: &kitchensink.DoSignal_DoSignalActions_DoActions{
														DoActions: &kitchensink.ActionSet{
															Actions: []*kitchensink.Action{
																kitchensink.NewSetWorkflowStateAction(stateKey, stateValue),
															},
														},
													},
												},
											},
										},
									},
								}
							}

							// Execute an activity that performs the client action (sends signal/update)
							// This activity will use the Temporal client to send the signal/update back to its parent workflow
							actions = append(actions, kitchensink.ClientActivity(
								kitchensink.ClientActions(clientAction),
								kitchensink.DefaultRemoteActivity,
							))

							// Wait for the workflow state to be set to the current index by the signal/update
							// This ensures signals/updates are processed consecutively in order (0, 1, 2, ...)
							actions = append(actions, kitchensink.NewAwaitWorkflowStateAction(stateKey, stateValue))
						}

						// Add final return action
						actions = append(actions, kitchensink.NewEmptyReturnResultAction())

						// Set the actions as sequential (not concurrent)
						params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
							{
								Actions:    actions,
								Concurrent: false,
							},
						}

						return nil
					},
				},
			}
		},
		VerifyFn: func(ctx context.Context, info loadgen.ScenarioInfo, executor loadgen.Executor) []error {
			e := executor.(*workflowLoopExecutor)
			if e.completionVerifier == nil {
				return nil
			}
			state := e.KitchenSinkExecutor.GetState()
			return e.completionVerifier.VerifyRun(ctx, info, state)
		},
	})
}
