// Package scenarios — tiny_queues port from bench-go (RE-263, RE-257 epic).
//
// Each scenario iteration represents one "batch" (a logical event ID step)
// in bench-go terms: the driver fans out SignalWithStart calls carrying an
// EventID payload to a slice of target workflows. The slice is rotated
// across iterations using modular arithmetic so the same target workflow
// observes events in a different order than its neighbours — that is what
// stresses the worker-side out-of-order signal buffering.
//
// Distinguishing math (do NOT drop): for inner index iter in
// [0, signals-per-workflow), the offset workflow group is
//
//	offsetIter = (batchID + 31*iter) % totalIterations
//
// IsCoprime(31, totalIterations) is a precondition; without it the rotation
// fails to cover every workflow group. We enforce that in Configure.
//
// Knobs are kebab-case per omes convention. Fields from bench-go's
// TinyQueueParameters that did not survive the port:
//   - UseLocalActivity, MaxWorkflowsToMonitor: monitor/activity logic is not
//     reproduced in this first cut (see worker package TODO).
//   - ContinueAsNewInfo: internal to the workflow, not surfaced.
package scenarios

import (
	"context"
	"fmt"
	"sync"

	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/tinyqueues"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
)

const (
	TinyQueuesSignaledWorkflowsPerIterationFlag = "signaled-workflows-per-iteration"
	TinyQueuesMaxQueueLifetimeSecondsFlag       = "maximum-queue-lifetime-seconds"
	TinyQueuesPercentageInOrderFlag             = "percentage-workflows-in-order"
	TinyQueuesMaxProcessTimePerMessageFlag      = "max-process-time-per-message-seconds"
	TinyQueuesSignalsPerWorkflowFlag            = "signals-per-workflow"
	TinyQueuesTotalIterationsFlag               = "total-iterations"
	TinyQueuesTotalBatchesFlag                  = "total-batches"
)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: fmt.Sprintf(
			"tiny_queues: port of bench-go tinyqueues. Each iteration is a batch; "+
				"the driver fans out SignalWithStart calls with EventID payloads to a "+
				"rotating slice of target workflows. Options: '%s', '%s', '%s', '%s', "+
				"'%s', '%s' (must be coprime with 31), '%s'.",
			TinyQueuesSignaledWorkflowsPerIterationFlag,
			TinyQueuesMaxQueueLifetimeSecondsFlag,
			TinyQueuesPercentageInOrderFlag,
			TinyQueuesMaxProcessTimePerMessageFlag,
			TinyQueuesSignalsPerWorkflowFlag,
			TinyQueuesTotalIterationsFlag,
			TinyQueuesTotalBatchesFlag),
		ExecutorFn: func() loadgen.Executor {
			return &loadgen.GenericExecutor{
				Execute: func(ctx context.Context, run *loadgen.Run) error {
					exec := &tinyQueuesExecutor{}
					return exec.Execute(ctx, run)
				},
			}
		},
	})
}

type tinyQueuesConfig struct {
	SignaledWorkflowsPerIteration   int
	MaxQueueLifetimeSeconds         int
	PercentageInOrder               int
	MaxProcessTimePerMessageSeconds int
	SignalsPerWorkflow              int
	TotalIterations                 int
	TotalBatches                    int
}

type tinyQueuesExecutor struct {
	config *tinyQueuesConfig
}

var _ loadgen.Configurable = (*tinyQueuesExecutor)(nil)

// Configure parses & validates scenario options.
func (t *tinyQueuesExecutor) Configure(info loadgen.ScenarioInfo) error {
	c := &tinyQueuesConfig{
		SignaledWorkflowsPerIteration:   info.ScenarioOptionInt(TinyQueuesSignaledWorkflowsPerIterationFlag, 1),
		MaxQueueLifetimeSeconds:         info.ScenarioOptionInt(TinyQueuesMaxQueueLifetimeSecondsFlag, 3600),
		PercentageInOrder:               info.ScenarioOptionInt(TinyQueuesPercentageInOrderFlag, 100),
		MaxProcessTimePerMessageSeconds: info.ScenarioOptionInt(TinyQueuesMaxProcessTimePerMessageFlag, 0),
		SignalsPerWorkflow:              info.ScenarioOptionInt(TinyQueuesSignalsPerWorkflowFlag, 1),
		TotalIterations:                 info.ScenarioOptionInt(TinyQueuesTotalIterationsFlag, 1),
		TotalBatches:                    info.ScenarioOptionInt(TinyQueuesTotalBatchesFlag, 1),
	}

	if c.SignaledWorkflowsPerIteration <= 0 {
		return fmt.Errorf("%s must be positive, got %d", TinyQueuesSignaledWorkflowsPerIterationFlag, c.SignaledWorkflowsPerIteration)
	}
	if c.MaxQueueLifetimeSeconds <= 0 {
		return fmt.Errorf("%s must be positive, got %d", TinyQueuesMaxQueueLifetimeSecondsFlag, c.MaxQueueLifetimeSeconds)
	}
	if c.PercentageInOrder < 0 || c.PercentageInOrder > 100 {
		return fmt.Errorf("%s must be in [0,100], got %d", TinyQueuesPercentageInOrderFlag, c.PercentageInOrder)
	}
	if c.SignalsPerWorkflow <= 0 {
		return fmt.Errorf("%s must be positive, got %d", TinyQueuesSignalsPerWorkflowFlag, c.SignalsPerWorkflow)
	}
	if c.TotalIterations <= 0 {
		return fmt.Errorf("%s must be positive, got %d", TinyQueuesTotalIterationsFlag, c.TotalIterations)
	}
	if c.TotalBatches <= 0 {
		return fmt.Errorf("%s must be positive, got %d", TinyQueuesTotalBatchesFlag, c.TotalBatches)
	}
	if !isCoprime(31, c.TotalIterations) {
		return fmt.Errorf("%s (%d) must be coprime with 31 — otherwise the modular fan-out does not cover all workflows",
			TinyQueuesTotalIterationsFlag, c.TotalIterations)
	}

	t.config = c
	return nil
}

// Execute is invoked once per omes iteration. We treat the iteration number
// as the bench-go "batchID" and inside it drive the
// SignalWithStartWorkflow fan-out across signals-per-workflow * signaled-
// workflows-per-iteration target workflows.
func (t *tinyQueuesExecutor) Execute(ctx context.Context, run *loadgen.Run) error {
	if err := t.Configure(*run.ScenarioInfo); err != nil {
		return err
	}
	c := t.config

	// bench-go's batchID is 0-indexed; omes iterations start at 1. Map 1:1
	// minus one so iter==1 corresponds to batchID==0.
	batchID := run.Iteration - 1

	var wg sync.WaitGroup
	errCh := make(chan error, c.SignalsPerWorkflow*c.SignaledWorkflowsPerIteration*c.TotalBatches+1)

	for iter := 0; iter < c.SignalsPerWorkflow; iter++ {
		// Modular rotation that defines tiny_queues. EventID payload for
		// this (batchID, iter) is iter+1 (1-indexed to mirror bench-go).
		offsetIter := ((batchID + 31*iter) % c.TotalIterations)
		if offsetIter < 0 { // defensive against negative-mod surprises
			offsetIter += c.TotalIterations
		}
		startWF := offsetIter * c.SignaledWorkflowsPerIteration
		endWF := (offsetIter + 1) * c.SignaledWorkflowsPerIteration
		eventID := iter + 1

		for wfNum := startWF; wfNum < endWF; wfNum++ {
			wfNum := wfNum
			if wfNum%100 < c.PercentageInOrder {
				// "Strict order" path: only the matching batch sends ALL
				// EventIDs to this workflow in one go. Other batches skip
				// signalling it entirely so events arrive in-order.
				if c.TotalBatches > 0 && wfNum%c.TotalBatches == batchID {
					for j := 1; j <= c.TotalBatches; j++ {
						j := j
						wg.Add(1)
						go func() {
							defer wg.Done()
							if err := t.signalOne(ctx, run, wfNum, j); err != nil {
								select {
								case errCh <- err:
								default:
								}
							}
						}()
					}
				}
			} else {
				// "Out of order" path: every batch sends its own EventID.
				wg.Add(1)
				go func() {
					defer wg.Done()
					if err := t.signalOne(ctx, run, wfNum, eventID); err != nil {
						select {
						case errCh <- err:
						default:
						}
					}
				}()
			}
		}
	}

	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *tinyQueuesExecutor) signalOne(ctx context.Context, run *loadgen.Run, wfNum, eventID int) error {
	c := t.config
	wfID := fmt.Sprintf("tinyqueue-%s-%d", run.RunID, wfNum)
	opts := client.StartWorkflowOptions{
		ID:                                       wfID,
		TaskQueue:                                run.TaskQueue(),
		WorkflowIDReusePolicy:                    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		WorkflowExecutionErrorWhenAlreadyStarted: false,
	}
	input := tinyqueues.TinyQueueInput{
		MaximumQueueLifetimeSeconds:     c.MaxQueueLifetimeSeconds,
		MaxProcessTimePerMessageSeconds: c.MaxProcessTimePerMessageSeconds,
	}
	msg := tinyqueues.TinyQueueMessage{
		EventID:                         eventID,
		MaxProcessTimePerMessageSeconds: c.MaxProcessTimePerMessageSeconds,
	}
	_, err := run.Client.SignalWithStartWorkflow(
		ctx,
		opts.ID,
		tinyqueues.ResourceEventSignalName,
		msg,
		opts,
		tinyqueues.WorkflowName,
		input,
	)
	if err != nil {
		return fmt.Errorf("SignalWithStartWorkflow failed (wfID=%s eventID=%d): %w", wfID, eventID, err)
	}
	return nil
}

// isCoprime returns true iff gcd(a, b) == 1. Mirrors bench-go's
// common.IsCoprime/GreatestCommonDivisor helpers — duplicated locally so the
// scenario stays self-contained.
func isCoprime(a, b int) bool {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	for b != 0 {
		a, b = b, a%b
	}
	return a == 1
}
