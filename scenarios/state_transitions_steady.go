package scenarios

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type steadyStateConfig struct {
	MaxConsecutiveErrors int
}

type stateTransitionsSteadyExecutor struct {
	loadgen.ScenarioInfo
	config *steadyStateConfig
}

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Run a certain number of state transitions per second. This requires duration option to be set " +
			"and the state-transitions-per-second scenario option to be set. Any iteration configuration is ignored. For " +
			"example, can be run with: run-scenario-with-worker --scenario state_transitions_steady --language go " +
			"--embedded-server --duration 5m --option state-transitions-per-second=3",
		ExecutorFn: func() loadgen.Executor {
			return &stateTransitionsSteadyExecutor{}
		},
	})
}

var _ loadgen.Configurable = (*stateTransitionsSteadyExecutor)(nil)

// Configure initializes the steadyStateConfig by reading scenario options
func (s *stateTransitionsSteadyExecutor) Configure(info loadgen.ScenarioInfo) error {
	s.ScenarioInfo = info
	s.config = &steadyStateConfig{
		MaxConsecutiveErrors: s.ScenarioOptionInt(MaxConsecutiveErrorsFlag, 5),
	}
	if s.config.MaxConsecutiveErrors < 1 {
		return fmt.Errorf("%s must be at least 1, got %d", MaxConsecutiveErrorsFlag, s.config.MaxConsecutiveErrors)
	}
	return nil
}

// Run executes the state transitions steady scenario
func (s *stateTransitionsSteadyExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := s.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}
	return s.run(ctx)
}

func (s *stateTransitionsSteadyExecutor) run(ctx context.Context) error {
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

	completionChecker, err := loadgen.NewWorkflowCompletionChecker(ctx, s.ScenarioInfo, time.Minute)
	if err != nil {
		return fmt.Errorf("failed to create workflow completion checker: %w", err)
	}

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
				if consecutiveErrCount >= s.config.MaxConsecutiveErrors {
					return fmt.Errorf("got %v consecutive errors, most recent: %w", s.config.MaxConsecutiveErrors, err)
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

	return completionChecker.VerifyNoRunningWorkflows(ctx)
}
