package kitchensink

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ks "github.com/temporalio/omes/loadgen/kitchensink"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/proto"
)

const (
	finiteScheduleOwnerKey  = "omes_finite_schedule"
	finiteScheduleRecipeKey = "omes_finite_recipe"
	finiteScheduleRecipe    = "v1:10s:10s:3+immediate"
)

func finiteScheduleWorkflowInput() *ks.WorkflowInput {
	return &ks.WorkflowInput{InitialActions: ks.ListActionSet(ks.NewReturnResultAction(ks.ConvertToPayload("scheduled")))}
}

// CreateFiniteSchedule creates a schedule with three timed actions and one immediate action.
func (ca *ClientActivities) CreateFiniteSchedule(ctx context.Context, input ks.FiniteScheduleInput) error {
	queue := activity.GetInfo(ctx).TaskQueue
	if !strings.HasPrefix(input.ScheduleID, "omes-finite-") || len(input.ScheduleID) == len("omes-finite-") || queue == "" {
		return temporal.NewNonRetryableApplicationError("finite schedule requires an omes-finite- ID and task queue", "InvalidFiniteScheduleInput", nil)
	}
	schedules := ca.Client.ScheduleClient()
	_, err := schedules.Create(ctx, client.ScheduleOptions{
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
	})
	if err == nil {
		return nil
	}
	if !errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return fmt.Errorf("creating finite schedule %s: %w", input.ScheduleID, err)
	}
	existing, err := schedules.GetHandle(ctx, input.ScheduleID).Describe(ctx)
	if err != nil {
		return fmt.Errorf("reconciling finite schedule %s: %w", input.ScheduleID, err)
	}
	if !matchesFiniteSchedule(existing, input.ScheduleID, queue) {
		return temporal.NewNonRetryableApplicationError("existing finite schedule differs from requested identity or configuration", "FiniteScheduleCollision", nil)
	}
	return nil
}

func matchesFiniteSchedule(existing *client.ScheduleDescription, id, queue string) bool {
	if existing == nil {
		return false
	}
	spec := existing.Schedule.Spec
	state := existing.Schedule.State
	policy := existing.Schedule.Policy
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
	var actual ks.WorkflowInput
	if err := dc.FromPayload(payload, &actual); err != nil || !proto.Equal(&actual, finiteScheduleWorkflowInput()) {
		return false
	}
	var owner string
	if err := dc.FromPayload(existing.Memo.GetFields()[finiteScheduleOwnerKey], &owner); err != nil {
		return false
	}
	var recipe string
	if err := dc.FromPayload(existing.Memo.GetFields()[finiteScheduleRecipeKey], &recipe); err != nil {
		return false
	}
	return owner == id && recipe == finiteScheduleRecipe
}
