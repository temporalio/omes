package scenarios

import (
	"context"
	"fmt"

	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/sdk/client"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a workflow whose only job is to invoke a single Nexus operation. " +
			"Requires --option " + NexusEndpointFlag + "=<name>.",
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, r *loadgen.Run) error {
					endpoint := r.ScenarioOptions[NexusEndpointFlag]
					if endpoint == "" {
						return fmt.Errorf("--option %s=<name> is required", NexusEndpointFlag)
					}
					payloadSize := r.ScenarioOptionInt("payload-size", 0)
					handle, err := r.Client.ExecuteWorkflow(
						ctx,
						client.StartWorkflowOptions{
							TaskQueue: loadgen.TaskQueueForRun(r.RunID),
							ID:        fmt.Sprintf("w-%s-%s-%d", r.RunID, r.ExecutionID, r.Iteration),
							WorkflowExecutionErrorWhenAlreadyStarted: !r.Configuration.IgnoreAlreadyStarted,
						},
						"singleNexusOpWorkflow",
						endpoint,
						make([]byte, payloadSize),
						int32(payloadSize),
					)
					if err != nil {
						return err
					}
					return handle.Get(ctx, nil)
				},
			}
		},
	})
}
