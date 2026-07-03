package scenarios

import (
	"testing"
	"time"

	"github.com/temporalio/omes/loadgen"
)

func TestFailureRetryAllowsOneFailureAndOneRetry(t *testing.T) {
	scenario := loadgen.GetScenario("failure_retry")
	executor, ok := scenario.ExecutorFn().(loadgen.KitchenSinkExecutor)
	if !ok {
		t.Fatal("failure_retry does not use KitchenSinkExecutor")
	}
	activity := executor.TestInput.WorkflowInput.InitialActions[0].Actions[0].GetExecActivity()
	if got := activity.StartToCloseTimeout.AsDuration(); got != 5*time.Second {
		t.Fatalf("got StartToClose timeout %v, want 5s", got)
	}
	if got := activity.RetryPolicy.MaximumAttempts; got != 2 {
		t.Fatalf("got %d maximum attempts, want 2", got)
	}
	if got := activity.GetRetryableError().FailAttempts; got != 1 {
		t.Fatalf("got %d failed attempts, want 1", got)
	}
}
