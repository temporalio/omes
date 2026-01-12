package cli

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"go.temporal.io/api/batch/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.uber.org/zap"
)

func cleanupScenarioCmd() *cobra.Command {
	var c scenarioCleaner
	cmd := &cobra.Command{
		Use:   "cleanup-scenario",
		Short: "Cleanup scenario",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := withCancelOnInterrupt(cmd.Context())
			defer cancel()
			if err := c.run(ctx); err != nil {
				c.logger.Fatal(err)
			}
		},
	}
	c.addCLIFlags(cmd.Flags())
	cmd.MarkFlagRequired("scenario")
	cmd.MarkFlagRequired("run-id")
	return cmd
}

type scenarioCleaner struct {
	logger         *zap.SugaredLogger
	pollInterval   time.Duration
	scenario       clioptions.ScenarioID
	clientOptions  clioptions.ClientOptions
	metricsOptions clioptions.MetricsOptions
	loggingOptions clioptions.LoggingOptions
}

func (c *scenarioCleaner) addCLIFlags(fs *pflag.FlagSet) {
	c.scenario.AddCLIFlags(fs)
	fs.DurationVar(&c.pollInterval, "poll-interval", time.Second, "Interval for polling completion of job")
	fs.AddFlagSet(c.clientOptions.FlagSet())
	fs.AddFlagSet(c.metricsOptions.FlagSet(""))
	fs.AddFlagSet(c.loggingOptions.FlagSet())
}

func (c *scenarioCleaner) run(ctx context.Context) error {
	c.logger = c.loggingOptions.MustCreateLogger()
	scenario := loadgen.GetScenario(c.scenario.Scenario)
	if scenario == nil {
		return fmt.Errorf("scenario not found")
	} else if c.scenario.RunID == "" {
		return fmt.Errorf("run ID not found")
	}
	metrics := c.metricsOptions.MustCreateMetrics(ctx, c.logger)
	defer metrics.Shutdown(ctx, c.logger, c.scenario.Scenario, c.scenario.RunID, c.scenario.RunFamily)
	client := c.clientOptions.MustDial(metrics, c.logger)
	defer client.Close()
	taskQueue := loadgen.TaskQueueForRun(c.scenario.RunID)
	jobID := "omes-cleanup-" + taskQueue + "-" + uuid.New().String()
	username, hostname := "anonymous", "unknown"
	if user, err := user.Current(); err == nil {
		username = user.Name
	}
	if host, err := os.Hostname(); err == nil {
		hostname = host
	}

	// Start
	_, err := client.WorkflowService().StartBatchOperation(ctx, &workflowservice.StartBatchOperationRequest{
		Namespace: c.clientOptions.Namespace,
		JobId:     jobID,
		Reason:    "omes cleanup",
		// Clean based on task queue to avoid relying on search attributes and
		// reducing the requirements of this framework
		VisibilityQuery: fmt.Sprintf("TaskQueue = %q", taskQueue),
		Operation: &workflowservice.StartBatchOperationRequest_DeletionOperation{
			DeletionOperation: &batch.BatchOperationDeletion{Identity: username + "@" + hostname},
		},
	})
	if err != nil {
		return fmt.Errorf("failed starting batch: %w", err)
	}

	// Loop waiting for batch complete
	for {
		time.Sleep(c.pollInterval)
		resp, err := client.WorkflowService().DescribeBatchOperation(ctx, &workflowservice.DescribeBatchOperationRequest{
			Namespace: c.clientOptions.Namespace,
			JobId:     jobID,
		})
		if err != nil {
			return fmt.Errorf("failed checking batch: %w", err)
		}
		switch resp.State {
		case enums.BATCH_OPERATION_STATE_FAILED:
			return fmt.Errorf("cleanup batch failed: %w", err)
		case enums.BATCH_OPERATION_STATE_COMPLETED:
			return nil
		case enums.BATCH_OPERATION_STATE_RUNNING:
			continue
		default:
			return fmt.Errorf("unexpected batch state %v, reason: %v", resp.State, resp.Reason)
		}
	}
}
