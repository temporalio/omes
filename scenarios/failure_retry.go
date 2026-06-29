package scenarios

import (
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration starts one workflow with one retryable activity that fails once, retries, and completes.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &TestInput{
					WorkflowInput: &WorkflowInput{
						InitialActions: ListActionSet(
							RetryableErrorActivity(1, RemoteActivityWithRetry(1*time.Second, 2, 500*time.Millisecond, 1.0)),
							NewEmptyReturnResultAction(),
						),
					},
				},
			}
		},
	})
}
