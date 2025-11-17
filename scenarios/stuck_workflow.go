package scenarios

import (
	"context"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/converter"
)

// stuckWorkflowExecutor wraps KitchenSinkExecutor and implements Verifier interface
// to detect stuck workflows using WorkflowCompletionVerifier.
type stuckWorkflowExecutor struct {
	*loadgen.KitchenSinkExecutor
	verifier *loadgen.WorkflowCompletionVerifier
}

var _ loadgen.Verifier = (*stuckWorkflowExecutor)(nil)
var _ loadgen.Resumable = (*stuckWorkflowExecutor)(nil)

func (e *stuckWorkflowExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	// Create the verifier before running
	verifier, err := loadgen.NewWorkflowCompletionChecker(ctx, info, 30*time.Second)
	if err != nil {
		return err
	}
	e.verifier = verifier

	// Run the embedded executor
	return e.KitchenSinkExecutor.Run(ctx, info)
}

func (e *stuckWorkflowExecutor) VerifyRun(ctx context.Context, info loadgen.ScenarioInfo, state loadgen.ExecutorState) []error {
	if e.verifier == nil {
		return nil
	}
	return e.verifier.VerifyRun(ctx, info, state)
}

func (e *stuckWorkflowExecutor) Snapshot() any {
	return e.KitchenSinkExecutor.Snapshot()
}

func (e *stuckWorkflowExecutor) LoadState(loader func(any) error) error {
	return e.KitchenSinkExecutor.LoadState(loader)
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Test scenario where the first iteration blocks forever (stuck workflow), " +
			"even iterations use Continue-As-New, and odd iterations complete normally. " +
			"Used for testing workflow completion detection.",
		ExecutorFn: func() loadgen.Executor {
			return &stuckWorkflowExecutor{
				KitchenSinkExecutor: &loadgen.KitchenSinkExecutor{
					TestInput: &kitchensink.TestInput{
						WorkflowInput: &kitchensink.WorkflowInput{},
					},
					UpdateWorkflowOptions: func(ctx context.Context, run *loadgen.Run, options *loadgen.KitchenSinkWorkflowOptions) error {
					// Only the first iteration should block forever.
					if run.Iteration == 1 {
						options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
							{
								Actions: []*kitchensink.Action{
									{
										Variant: &kitchensink.Action_AwaitWorkflowState{
											AwaitWorkflowState: &kitchensink.AwaitWorkflowState{
												Key:   "will-never-be-set",
												Value: "never",
											},
										},
									},
								},
							},
						}
					} else if run.Iteration%2 == 0 {
						// Have some Continue-As-New.
						// ContinueAsNew needs to pass the workflow input as the first argument.
						// We pass a simple completion action to make the continued workflow complete immediately.
						workflowInput, err := converter.GetDefaultDataConverter().ToPayload(
							&kitchensink.WorkflowInput{
								InitialActions: []*kitchensink.ActionSet{
									kitchensink.NoOpSingleActivityActionSet(),
								},
							})
						if err != nil {
							return err
						}
						options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
							{
								Actions: []*kitchensink.Action{
									{
										Variant: &kitchensink.Action_ContinueAsNew{
											ContinueAsNew: &kitchensink.ContinueAsNewAction{
												Arguments: []*common.Payload{workflowInput},
											},
										},
									},
								},
							},
						}
					} else {
						options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
							kitchensink.NoOpSingleActivityActionSet(),
						}
					}
					return nil
				},
			},
		}
	},
})
}
