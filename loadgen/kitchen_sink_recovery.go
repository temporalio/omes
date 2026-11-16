package loadgen

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/temporalio/omes/loadgen/journal"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/temporal"
)

type frozenKitchenSink struct {
	Start frozenStartOptions
	Spec  kitchensink.OriginalStartSpec
}
type recordedWorkflowResult struct{ Failure string }
type terminalWorkflowFailure struct{ message string }

func (e *terminalWorkflowFailure) Error() string { return e.message }

func (k KitchenSinkExecutor) executeJournaled(ctx context.Context, run *Run) error {
	item, err := run.Journal.Plan(ctx, "workflow", time.Time{}, func() ([]byte, error) {
		options, err := k.iterationOptions(ctx, run)
		if err != nil {
			return nil, err
		}
		if len(options.Params.GetClientSequence().GetActionSets()) > 0 {
			return nil, errors.New("journaled workflow load does not support client sequences")
		}
		executor := run.newClientActionsExecutor(&options)
		executor.RecoverOriginalStart = true
		spec, err := executor.PrepareOriginalStart(options.Params.WithStartAction)
		if err != nil {
			return nil, err
		}
		start, err := freezeStartOptions(executor.WorkflowOptions)
		if err != nil {
			return nil, err
		}
		return json.Marshal(frozenKitchenSink{Start: start, Spec: spec})
	})
	if err != nil {
		return err
	}
	if item.Done {
		var result recordedWorkflowResult
		if err := json.Unmarshal(item.Observation, &result); err != nil {
			return err
		}
		if result.Failure != "" {
			return &terminalWorkflowFailure{result.Failure}
		}
		return nil
	}
	if run.kitchenSinkExecution == nil {
		var frozen frozenKitchenSink
		if err := json.Unmarshal(item.Plan, &frozen); err != nil {
			return err
		}
		options, err := frozen.Start.restore()
		if err != nil {
			return err
		}
		execution := &kitchenSinkExecution{options: KitchenSinkWorkflowOptions{StartOptions: options}}
		execution.executor = run.newClientActionsExecutor(&execution.options)
		if err := execution.executor.RestoreOriginalStart(frozen.Spec); err != nil {
			return err
		}
		run.kitchenSinkExecution = execution
	}
	if err := run.Journal.Issue(ctx, "workflow"); err != nil {
		return err
	}
	executor := run.kitchenSinkExecution.executor
	if err := executor.Start(ctx, nil); err != nil {
		return err
	}
	binding, err := run.Journal.Plan(ctx, "original-run", time.Time{}, func() ([]byte, error) { return []byte(executor.Handle.GetRunID()), nil })
	if err != nil {
		return err
	}
	if string(binding.Plan) != executor.Handle.GetRunID() {
		return journal.ErrConflict
	}
	if !binding.Done {
		if err := run.Journal.Observe(ctx, "original-run", []byte("verified")); err != nil {
			return err
		}
	}
	resultErr := executor.Handle.Get(ctx, nil)
	var closedFailure *temporal.WorkflowExecutionError
	if resultErr != nil && !errors.As(resultErr, &closedFailure) {
		return resultErr
	}
	result := recordedWorkflowResult{}
	if resultErr != nil {
		result.Failure = truncate(resultErr.Error())
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return err
	}
	if err := run.Journal.Observe(ctx, "workflow", raw); err != nil {
		return fmt.Errorf("recording original workflow result: %w", err)
	}
	if result.Failure != "" {
		return &terminalWorkflowFailure{result.Failure}
	}
	return nil
}
