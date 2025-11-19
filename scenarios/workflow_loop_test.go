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

// TestWorkflowLoopScenario tests the workflow_loop scenario with default settings (1 activity).
func TestWorkflowLoopScenario(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(60*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 3,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag: "1",
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")
	require.NotNil(t, scenario.VerifyFn, "workflow_loop scenario should have a VerifyFn")

	executor := scenario.ExecutorFn()

	// Run the executor
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "executor should complete successfully")

	// Verify the executor state
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 3, execState.CompletedIterations, "should complete 3 iterations")
}

// TestWorkflowLoopScenarioMultipleActivities tests the workflow_loop scenario with multiple activities.
func TestWorkflowLoopScenarioMultipleActivities(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(30*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-multi-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag: "5",
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "executor should complete successfully with 5 activities")

	// Verify the executor state
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 2, execState.CompletedIterations, "should complete 2 iterations")
}

// TestWorkflowLoopScenarioVerification tests that the VerifyFn properly validates workflow completion.
func TestWorkflowLoopScenarioVerification(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(30*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-verify-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 5,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag: "3",
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor and verification
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "should complete successfully and pass verification")

	// Verify the executor state shows correct number of completed iterations
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 5, execState.CompletedIterations, "should complete all 5 iterations")
}

// TestWorkflowLoopScenarioInvalidConfig tests that invalid configuration is rejected.
func TestWorkflowLoopScenarioInvalidConfig(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(10*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-invalid-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 1,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag: "0", // Invalid: must be positive
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor - should fail due to invalid configuration
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.Error(t, err, "should fail with invalid activity count")
	require.Contains(t, err.Error(), "must be positive", "error should mention positive requirement")
}

// TestWorkflowLoopScenarioWithUpdates tests the workflow_loop scenario using updates instead of signals.
func TestWorkflowLoopScenarioWithUpdates(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(30*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-update-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag:      "3",
			MessageViaFlag: "update", // Use updates instead of signals
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "executor should complete successfully with updates")

	// Verify the executor state
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 2, execState.CompletedIterations, "should complete 2 iterations using updates")
}

// TestWorkflowLoopScenarioWithUpdatesMultipleIterations tests updates with more iterations.
func TestWorkflowLoopScenarioWithUpdatesMultipleIterations(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(30*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-update-multi-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 4,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag:      "2",
			MessageViaFlag: "update",
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "executor should complete successfully with multiple iterations using updates")

	// Verify the executor state
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 4, execState.CompletedIterations, "should complete 4 iterations")
}

// TestWorkflowLoopScenarioWithRandomSignalAndUpdate tests random selection between signals and updates.
func TestWorkflowLoopScenarioWithRandomSignalAndUpdate(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(30*time.Second))

	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("loop-random-via-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Iterations: 2,
		},
		ScenarioOptions: map[string]string{
			LoopsFlag:      "5",
			MessageViaFlag: "random", // Randomly pick between signal and update
		},
	}

	// Get the workflow_loop scenario
	scenario := loadgen.GetScenario("workflow_loop")
	require.NotNil(t, scenario, "workflow_loop scenario should be registered")

	executor := scenario.ExecutorFn()

	// Run the executor
	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "executor should complete successfully with random signal/update")

	// Verify the executor state
	resumable, ok := executor.(loadgen.Resumable)
	require.True(t, ok, "executor should implement Resumable interface")
	execState := resumable.Snapshot().(loadgen.ExecutorState)
	require.Equal(t, 2, execState.CompletedIterations, "should complete 2 iterations with random via")
}
