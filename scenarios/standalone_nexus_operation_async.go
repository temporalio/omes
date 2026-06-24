package scenarios

import (
	"context"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/client"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a standalone async Nexus operation (no caller workflow). The operation takes some bytes and " +
			"returns some bytes. It does not retry. Requires --option " + NexusEndpointFlag +
			"=<name> for an endpoint targeting this worker's task queue.",
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, r *loadgen.Run) error {
					endpoint := r.ScenarioOptions[NexusEndpointFlag]
					if endpoint == "" {
						return fmt.Errorf("--option %s=<name> is required", NexusEndpointFlag)
					}
					payloadSize := r.ScenarioOptionInt("payload-size", 0)

					nc, err := r.Client.NewNexusClient(client.NexusClientOptions{
						Endpoint: endpoint,
						Service:  "kitchen-sink",
					})
					if err != nil {
						return err
					}
					startToClose := time.Duration(r.ScenarioOptionInt("start-to-close-timeout-seconds", 30)) * time.Second
					scheduleToClose := time.Duration(r.ScenarioOptionInt("schedule-to-close-timeout-seconds", 120)) * time.Second
					handle, err := nc.ExecuteOperation(
						ctx,
						"payload-async",
						&kitchensink.PayloadNexusInput{
							Input:         make([]byte, payloadSize),
							BytesToReturn: int32(payloadSize),
						},
						client.StartNexusOperationOptions{
							ID:                     fmt.Sprintf("n-%s-%s-%d", r.RunID, r.ExecutionID, r.Iteration),
							StartToCloseTimeout:    startToClose,
							ScheduleToCloseTimeout: scheduleToClose,
						},
					)
					if err != nil {
						return err
					}
					getCtx, cancel := context.WithTimeout(ctx, time.Duration(r.ScenarioOptionInt("get-timeout-seconds", 120))*time.Second)
					defer cancel()
					return handle.Get(getCtx, nil)
				},
			}
		},
	})
}
