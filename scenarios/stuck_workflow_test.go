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

// TestStuckWorkflowScenario verifies that the stuck_workflow scenario correctly detects
// stuck workflows through its VerifyFn implementation.
func TestStuckWorkflowScenario(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(5*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("stuck-test-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 10,
		},
	}

	// Get the stuck_workflow scenario
	scenario := loadgen.GetScenario("stuck_workflow")
	require.NotNil(t, scenario, "stuck_workflow scenario should be registered")
	require.NotNil(t, scenario.VerifyFn, "stuck_workflow scenario should have a VerifyFn")

	executor := scenario.ExecutorFn()

	// RunExecutorTest will automatically run verification since the scenario has a VerifyFn
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "should fail due to stuck workflow detected by verification")
	require.Contains(t, err.Error(), "deadline exceeded", "should report deadline exceeded for stuck iteration")
}

// TestStuckWorkflowVerifyFnDetectsStuckWorkflow tests that the VerifyFn properly identifies
// stuck workflows after execution.
func TestStuckWorkflowVerifyFnDetectsStuckWorkflow(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(5*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("stuck-verify-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 5,
		},
	}

	// Get the stuck_workflow scenario
	scenario := loadgen.GetScenario("stuck_workflow")
	require.NotNil(t, scenario, "stuck_workflow scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor and expect it to fail
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "should fail due to verification detecting stuck workflow")

	// The error should indicate a non-completed workflow was detected
	require.Contains(t, err.Error(), "non-completed workflow", "verification should report stuck workflow")
}

// TestStuckWorkflowScenarioIterationBehavior tests that the stuck_workflow scenario
// behaves correctly across multiple iterations:
// - Iteration 1: blocks forever (stuck workflow)
// - Even iterations: use Continue-As-New
// - Odd iterations (except 1): complete normally
func TestStuckWorkflowScenarioIterationBehavior(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(5*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("stuck-behavior-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 7, // Test iterations 1-7
		},
	}

	// Get the stuck_workflow scenario
	scenario := loadgen.GetScenario("stuck_workflow")
	require.NotNil(t, scenario, "stuck_workflow scenario should be registered")

	executor := scenario.ExecutorFn()

	// RunExecutorTest will fail because iteration 1 will be stuck
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "should fail due to stuck workflow on iteration 1")

	// Verify the executor state shows 6 completed iterations (all except iteration 1)
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 6, execState.CompletedIterations,
		"should complete 6 iterations (iterations 2-7, skipping stuck iteration 1)")
}
