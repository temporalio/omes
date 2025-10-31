package scenarios

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers"
	"go.uber.org/zap/zaptest"
)

// Test that WorkflowCompletionChecker is able to detect a stuck workflow.
func TestWorkflowCompletionChecker(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(5*time.Second))

	testLogger := zaptest.NewLogger(t).Sugar()

	scenarioInfo := loadgen.ScenarioInfo{
		RunID:       fmt.Sprintf("stuck-%d", time.Now().Unix()),
		ExecutionID: "test-exec-id",
		Configuration: loadgen.RunConfiguration{
			Iterations: 10,
		},
		Client:    env.TemporalClient(),
		Namespace: "default",
		Logger:    testLogger,
	}

	// Create workflow completion verifier
	verifier, err := loadgen.NewWorkflowCompletionChecker(t.Context(), scenarioInfo, 30*time.Second)
	require.NoError(t, err, "failed to create verifier")

	executor := &loadgen.KitchenSinkExecutor{
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
						Concurrent: false,
					},
				}
			} else {
				options.Params.WorkflowInput.InitialActions = []*kitchensink.ActionSet{
					kitchensink.NoOpSingleActivityActionSet(),
				}
			}
			return nil
		},
	}

	_, err = env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "executor should fail because first iteration times out")
	require.Contains(t, err.Error(), "deadline exceeded", "should report timed out iteration")

	execState := executor.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 9, execState.CompletedIterations, "should complete 9 iterations")

	// Verify using the verifier - pass the state directly
	verifyErrs := verifier.VerifyRun(t.Context(), scenarioInfo, execState)
	require.NotEmpty(t, verifyErrs)
	require.Contains(t, verifyErrs[0].Error(), "non-completed workflow: WorkflowID=w-stuck-")
}
