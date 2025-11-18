package scenarios

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
)

// Test that WorkflowCompletionChecker is able to detect a stuck workflow.
// Uses the stuck_workflow scenario which has a workflow that blocks forever on iteration 1.
// The scenario's executor implements the Verifier interface, so env.RunExecutorTest
// automatically runs verification and reports errors.
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

	// Get the stuck_workflow scenario executor
	scenario := loadgen.GetScenario("stuck_workflow")
	require.NotNil(t, scenario, "stuck_workflow scenario should be registered")
	executor := scenario.ExecutorFn()

	// RunExecutorTest will automatically run verification since the executor implements Verifier
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "should fail due to stuck workflow and verification errors")
	require.Contains(t, err.Error(), "deadline exceeded", "should report timed out iteration")
	require.Contains(t, err.Error(), "non-completed workflow: Namespace=default, WorkflowID=w-stuck-", "should report stuck workflow from verifier")

	// Verify the executor state shows 9 completed iterations (all except the stuck one)
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 9, execState.CompletedIterations, "should complete 9 iterations (all except iteration 1 which is stuck)")
}
