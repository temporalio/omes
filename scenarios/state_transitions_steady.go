package scenarios

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/workflowservice/v1"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a certain number of state transitions per second. This requires duration option to be set " +
			"and the state-transitions-per-second scenario option to be set. Any iteration configuration is ignored. For " +
			"example, can be run with: run-scenario-with-worker --scenario state_transitions_steady --language go " +
			"--embedded-server --duration 5m --option state-transitions-per-second=3",
		Executor: loadgen.ExecutorFunc(func(ctx context.Context, runOptions loadgen.ScenarioInfo) error {
			return (&stateTransitionsSteady{runOptions}).run(ctx)
		}),
	})
}

type stateTransitionsSteady struct{ loadgen.ScenarioInfo }

func (s *stateTransitionsSteady) run(ctx context.Context) error {
	// The goal here is to meet a certain number of state transitions per second.
	// For us this means a certain number of workflows per second. So we must
	// first execute a basic workflow (i.e. with a simple activity) and get the
	// number of state transitions it took

	// TODO(cretz): Note, this is a known naive approach because if workflows are
	// backed up and/or slow down this won't match.

	if s.Configuration.Duration == 0 {
		return fmt.Errorf("duration required for this scenario")
	}
	stateTransitionsPerSecond := s.ScenarioOptionInt("state-transitions-per-second", 0)
	if stateTransitionsPerSecond == 0 {
		return fmt.Errorf("state-transitions-per-second scenario option required")
	}
	durationPerStateTransition := time.Second / time.Duration(stateTransitionsPerSecond)
	s.Logger.Infof(
		"State transitions per second is %v, which means %v between every state transition",
		stateTransitionsPerSecond,
		durationPerStateTransition,
	)

	// Execute initial workflow and get the transition count
	workflowParams := &kitchensink.WorkflowInput{
		InitialActions: []*kitchensink.ActionSet{
			kitchensink.NoOpSingleActivityActionSet(),
		},
	}
	workflowRun, err := s.Client.ExecuteWorkflow(
		ctx,
		s.NewRun(0).DefaultStartWorkflowOptions(),
		"kitchenSink",
		workflowParams,
	)
	if err != nil {
		return fmt.Errorf("failed starting initial workflow: %w", err)
	} else if err := workflowRun.Get(ctx, nil); err != nil {
		return fmt.Errorf("failed executing initial workflow: %w", err)
	}
	resp, err := s.Client.DescribeWorkflowExecution(ctx, workflowRun.GetID(), workflowRun.GetRunID())
	if err != nil {
		return fmt.Errorf("failed describing workflow: %w", err)
	}
	stateTransitionsPerWorkflow := resp.WorkflowExecutionInfo.StateTransitionCount
	workflowStartInterval := durationPerStateTransition * time.Duration(stateTransitionsPerWorkflow)
	s.Logger.Infof(
		"Simple workflow takes %v state transitions, which means we need to start a workflow every %v. "+
			"Running for %v now...",
		stateTransitionsPerWorkflow,
		workflowStartInterval,
		s.Configuration.Duration,
	)

	// Start a workflow every X interval until duration reached or there are N
	// start failures in a row

	const maxConsecutiveErrors = 5
	errCh := make(chan error, 10000)
	ticker := time.NewTicker(workflowStartInterval)
	defer ticker.Stop()
	var consecutiveErrCount int
	iter := 1
	var startWG sync.WaitGroup
	for begin := time.Now(); time.Since(begin) < s.Configuration.Duration; iter++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err == nil {
				consecutiveErrCount = 0
			} else {
				consecutiveErrCount++
				if consecutiveErrCount >= maxConsecutiveErrors {
					return fmt.Errorf("got %v consecutive errors, most recent: %w", maxConsecutiveErrors, err)
				}
			}
		case <-ticker.C:
			s.Logger.Debugf("Running iteration %v", iter)
			startWG.Add(1)
			go func(iter int) {
				defer startWG.Done()
				_, err := s.Client.ExecuteWorkflow(
					ctx,
					s.NewRun(iter).DefaultStartWorkflowOptions(),
					"kitchenSink",
					workflowParams,
				)
				// Only send to err ch if there is room just in case
				select {
				case errCh <- err:
				default:
				}
			}(iter)
		}
	}

	// We are waiting to let workflows complete here, knowing that this part of
	// the scenario is not steady state transitions. We will wait a minute max to
	// confirm all workflows on the task queue are no longer running.
	s.Logger.Infof("Run complete, ran %v iterations, waiting on all workflows to complete", iter)
	// First, wait for all starts to have started (they are done in goroutine)
	startWG.Wait()
	return loadgen.MinVisibilityCountEventually(
		ctx,
		s.ScenarioInfo,
		&workflowservice.CountWorkflowExecutionsRequest{
			Namespace: s.Namespace,
			Query: fmt.Sprintf("TaskQueue = %q and ExecutionStatus = 'Running'",
				loadgen.TaskQueueForRun(s.ScenarioName, s.RunID)),
		},
		0,
		time.Minute,
	)
}
