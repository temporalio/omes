package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"google.golang.org/protobuf/proto"
)

// CreateFiniteScheduleActivityName is implemented by the Go worker.
const CreateFiniteScheduleActivityName = "create_finite_schedule"

// FiniteScheduleInput identifies one finite schedule across activity retries.
type FiniteScheduleInput struct {
	ScheduleID string `json:"scheduleId"`
}

const (
	finiteScheduleOwnerKey  = "omes_finite_schedule"
	finiteScheduleRecipeKey = "omes_finite_recipe"
	finiteScheduleRecipe    = "v1:10s:10s:3+immediate"
)

// ErrFiniteScheduleMismatch prevents reconciling or deleting a changed schedule.
var ErrFiniteScheduleMismatch = errors.New("finite schedule identity or configuration differs")

func finiteScheduleWorkflowInput() *WorkflowInput {
	return &WorkflowInput{InitialActions: ListActionSet(NewReturnResultAction(ConvertToPayload("scheduled")))}
}

func validateFiniteScheduleInput(input FiniteScheduleInput, queue string) error {
	if !strings.HasPrefix(input.ScheduleID, "omes-finite-") || len(input.ScheduleID) == len("omes-finite-") || queue == "" {
		return errors.New("finite schedule requires an omes-finite- ID and task queue")
	}
	return nil
}

// FiniteScheduleOptions builds the finite creation recipe on the supplied worker queue.
func FiniteScheduleOptions(input FiniteScheduleInput, queue string) (client.ScheduleOptions, error) {
	if err := validateFiniteScheduleInput(input, queue); err != nil {
		return client.ScheduleOptions{}, err
	}
	return client.ScheduleOptions{
		ID: input.ScheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{{Every: 10 * time.Second}},
			Jitter:    10 * time.Second,
		},
		Action: &client.ScheduleWorkflowAction{
			ID: input.ScheduleID + "/workflow", Workflow: "kitchenSink", TaskQueue: queue,
			Args: []any{finiteScheduleWorkflowInput()},
		},
		RemainingActions: 3, TriggerImmediately: true,
		Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP,
		// RemainingActions is consumed, so retain the original creation recipe.
		Memo: map[string]any{finiteScheduleOwnerKey: input.ScheduleID, finiteScheduleRecipeKey: finiteScheduleRecipe},
	}, nil
}

// MatchesFiniteSchedule checks stored ownership, original recipe and current configuration.
// Remaining actions may have been consumed; a matching schedule need not have settled.
func MatchesFiniteSchedule(existing *client.ScheduleDescription, id, queue string) bool {
	if existing == nil {
		return false
	}
	spec, state, policy := existing.Schedule.Spec, existing.Schedule.State, existing.Schedule.Policy
	if spec == nil || state == nil || policy == nil ||
		len(spec.Intervals) != 1 || spec.Intervals[0].Every != 10*time.Second || spec.Intervals[0].Offset != 0 ||
		spec.Jitter != 10*time.Second || len(spec.Calendars) != 0 || len(spec.CronExpressions) != 0 ||
		len(spec.Skip) != 0 || !spec.StartAt.IsZero() || !spec.EndAt.IsZero() ||
		!state.LimitedActions || state.RemainingActions < 0 || state.RemainingActions > 3 || state.Paused ||
		policy.Overlap != enumspb.SCHEDULE_OVERLAP_POLICY_SKIP || policy.PauseOnFailure {
		return false
	}
	action, ok := existing.Schedule.Action.(*client.ScheduleWorkflowAction)
	if !ok || action == nil || action.ID != id+"/workflow" || action.Workflow != "kitchenSink" || action.TaskQueue != queue ||
		action.WorkflowRunTimeout != 0 || action.WorkflowExecutionTimeout != 0 || action.RetryPolicy != nil || len(action.Args) != 1 {
		return false
	}
	payload, ok := action.Args[0].(*commonpb.Payload)
	if !ok {
		return false
	}
	dc := converter.GetDefaultDataConverter()
	var actual WorkflowInput
	if err := dc.FromPayload(payload, &actual); err != nil || !proto.Equal(&actual, finiteScheduleWorkflowInput()) {
		return false
	}
	var owner, recipe string
	if err := dc.FromPayload(existing.Memo.GetFields()[finiteScheduleOwnerKey], &owner); err != nil {
		return false
	}
	if err := dc.FromPayload(existing.Memo.GetFields()[finiteScheduleRecipeKey], &recipe); err != nil {
		return false
	}
	return owner == id && recipe == finiteScheduleRecipe
}

// VerifyFiniteSchedule requires four distinct successful executions, checking known outcomes before all actions settle.
// The caller supplies the observation deadline through ctx.
func VerifyFiniteSchedule(ctx context.Context, c client.Client, input FiniteScheduleInput, queue string) error {
	if err := validateFiniteScheduleInput(input, queue); err != nil {
		return err
	}
	handle := c.ScheduleClient().GetHandle(ctx, input.ScheduleID)
	verified := make(map[string]bool)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		description, err := handle.Describe(ctx)
		if err != nil {
			return fmt.Errorf("describing finite schedule %s: %w", input.ScheduleID, err)
		}
		if !MatchesFiniteSchedule(description, input.ScheduleID, queue) {
			return fmt.Errorf("verifying finite schedule %s: %w", input.ScheduleID, ErrFiniteScheduleMismatch)
		}
		if description.Info.NumActions > 4 || description.Info.NumActionsSkippedOverlap != 0 || description.Info.NumActionsMissedCatchupWindow != 0 {
			return fmt.Errorf("finite schedule %s has extra, skipped or missed actions", input.ScheduleID)
		}
		seen := make(map[string]bool)
		for _, action := range description.Info.RecentActions {
			execution := action.StartWorkflowResult
			if execution == nil ||
				(execution.WorkflowID != input.ScheduleID+"/workflow" && !strings.HasPrefix(execution.WorkflowID, input.ScheduleID+"/workflow-")) ||
				execution.FirstExecutionRunID == "" || seen[execution.FirstExecutionRunID] {
				return fmt.Errorf("finite schedule %s has missing or repeated execution identities", input.ScheduleID)
			}
			seen[execution.FirstExecutionRunID] = true
			if !verified[execution.FirstExecutionRunID] {
				var result string
				if err := c.GetWorkflow(ctx, execution.WorkflowID, execution.FirstExecutionRunID).Get(ctx, &result); err != nil {
					return fmt.Errorf("settling finite schedule %s execution %s: %w", input.ScheduleID, execution.FirstExecutionRunID, err)
				}
				if result != "scheduled" {
					return fmt.Errorf("finite schedule %s execution %s returned %q", input.ScheduleID, execution.FirstExecutionRunID, result)
				}
				closed, err := c.DescribeWorkflowExecution(ctx, execution.WorkflowID, execution.FirstExecutionRunID)
				if err != nil {
					return fmt.Errorf("checking finite schedule %s execution %s: %w", input.ScheduleID, execution.FirstExecutionRunID, err)
				}
				if closed.GetWorkflowExecutionInfo().GetStatus() != enumspb.WORKFLOW_EXECUTION_STATUS_COMPLETED {
					return fmt.Errorf("finite schedule %s execution %s did not complete directly", input.ScheduleID, execution.FirstExecutionRunID)
				}
				verified[execution.FirstExecutionRunID] = true
			}
		}
		if description.Schedule.State.RemainingActions == 0 && description.Info.NumActions == 4 {
			if len(seen) != 4 || len(verified) != 4 {
				return fmt.Errorf("finite schedule %s is missing final outcomes", input.ScheduleID)
			}
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

// DeleteFiniteSchedule removes only a matching schedule and confirms its absence.
// It does not settle workflows; callers must verify outcomes or arrange their cancellation separately.
func DeleteFiniteSchedule(ctx context.Context, schedules client.ScheduleClient, input FiniteScheduleInput, queue string) error {
	if err := validateFiniteScheduleInput(input, queue); err != nil {
		return err
	}
	handle := schedules.GetHandle(ctx, input.ScheduleID)
	description, err := handle.Describe(ctx)
	var notFound *serviceerror.NotFound
	if errors.As(err, &notFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("checking finite schedule %s before deletion: %w", input.ScheduleID, err)
	}
	if !MatchesFiniteSchedule(description, input.ScheduleID, queue) {
		return fmt.Errorf("refusing to delete finite schedule %s: %w", input.ScheduleID, ErrFiniteScheduleMismatch)
	}
	if err := handle.Delete(ctx); err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("deleting finite schedule %s: %w", input.ScheduleID, err)
	}
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	for {
		_, err := handle.Describe(ctx)
		if errors.As(err, &notFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("confirming finite schedule %s deletion: %w", input.ScheduleID, err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}
