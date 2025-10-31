package scenarios

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/workers"
)

// Test that WorkflowCompletionChecker is able to detect a stuck workflow.
func TestWorkflowCompletionChecker(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(5*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID:       fmt.Sprintf("stuck-%d", time.Now().Unix()),
		ExecutionID: "test-exec-id",
		Configuration: loadgen.RunConfiguration{
			Iterations: 10,
		},
	}

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

	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "Executor should fail because first workflow times out")

	errorMsg := err.Error()
	require.True(t,
		strings.Contains(errorMsg, "timeout") ||
			strings.Contains(errorMsg, "Timeout") ||
			strings.Contains(errorMsg, "deadline") ||
			strings.Contains(errorMsg, "DeadlineExceeded"),
		"Expected timeout-related error, got: %s", errorMsg)

	execState := executor.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 9, execState.CompletedIterations, "Should complete 9 iterations")
}
