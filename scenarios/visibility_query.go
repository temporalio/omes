package scenarios

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/workflowservice/v1"
)

type visibilityQueryExecutor struct {
	completed atomic.Int64
}

func visibilityWorkflowQuery(runID string) string {
	return fmt.Sprintf(`WorkflowId STARTS_WITH %q`, "w-"+runID+"-")
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration starts and completes one simple workflow, then verifies visibility count for the run.",
		ExecutorFn: func() loadgen.Executor {
			return &visibilityQueryExecutor{}
		},
	})
}

func (v *visibilityQueryExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	previousOnCompletion := info.Configuration.OnCompletion
	info.Configuration.OnCompletion = func(ctx context.Context, run *loadgen.Run) {
		v.completed.Add(1)
		if previousOnCompletion != nil {
			previousOnCompletion(ctx, run)
		}
	}

	executor := loadgen.KitchenSinkExecutor{
		TestInput: &TestInput{
			WorkflowInput: &WorkflowInput{
				InitialActions: []*ActionSet{
					NoOpSingleActivityActionSet(),
				},
			},
		},
	}
	if err := executor.Run(ctx, info); err != nil {
		return err
	}
	completed := v.completed.Load()
	if completed == 0 {
		return fmt.Errorf("no iterations completed")
	}
	timeout := info.ScenarioOptionDuration(VisibilityVerificationTimeoutFlag, 3*time.Minute)
	return loadgen.MinVisibilityCountEventually(
		ctx,
		info,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: info.Namespace,
			Query:     visibilityWorkflowQuery(info.RunID),
		},
		int(completed),
		timeout,
	)
}
