package kitchensink

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	commonpb "go.temporal.io/api/common/v1"
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
)

func TestFiniteScheduleOptions(t *testing.T) {
	options, err := FiniteScheduleOptions(FiniteScheduleInput{ScheduleID: "omes-finite-options"}, "owned-queue")
	require.NoError(t, err)
	require.Equal(t, 3, options.RemainingActions)
	require.True(t, options.TriggerImmediately)
	require.Equal(t, 10*time.Second, options.Spec.Jitter)
	require.Equal(t, []client.ScheduleIntervalSpec{{Every: 10 * time.Second}}, options.Spec.Intervals)
	action := options.Action.(*client.ScheduleWorkflowAction)
	require.Equal(t, "kitchenSink", action.Workflow)
	require.Equal(t, "owned-queue", action.TaskQueue)
	require.Zero(t, action.WorkflowRunTimeout)
	require.Zero(t, action.WorkflowExecutionTimeout)
	for _, input := range []struct{ id, queue string }{
		{"", "queue"}, {"omes-finite-", "queue"}, {"foreign", "queue"}, {"omes-finite-owned", ""},
	} {
		_, err := FiniteScheduleOptions(FiniteScheduleInput{ScheduleID: input.id}, input.queue)
		require.Error(t, err)
	}
}

func TestMatchesFiniteSchedule(t *testing.T) {
	const id, queue = "omes-finite-test", "load-queue"
	fixture := func() *client.ScheduleDescription {
		args, err := converter.GetDefaultDataConverter().ToPayload(finiteScheduleWorkflowInput())
		require.NoError(t, err)
		owner, err := converter.GetDefaultDataConverter().ToPayload(id)
		require.NoError(t, err)
		recipe, err := converter.GetDefaultDataConverter().ToPayload(finiteScheduleRecipe)
		require.NoError(t, err)
		return &client.ScheduleDescription{
			Schedule: client.Schedule{
				Spec:   &client.ScheduleSpec{Intervals: []client.ScheduleIntervalSpec{{Every: 10 * time.Second}}, Jitter: 10 * time.Second},
				State:  &client.ScheduleState{LimitedActions: true, RemainingActions: 3},
				Policy: &client.SchedulePolicies{Overlap: enumspb.SCHEDULE_OVERLAP_POLICY_SKIP},
				Action: &client.ScheduleWorkflowAction{
					ID: id + "/workflow", Workflow: "kitchenSink", TaskQueue: queue,
					Args: []any{args},
				},
			},
			Memo: &commonpb.Memo{Fields: map[string]*commonpb.Payload{finiteScheduleOwnerKey: owner, finiteScheduleRecipeKey: recipe}},
		}
	}
	for _, tc := range []struct {
		name   string
		change func(*client.ScheduleDescription)
		match  bool
	}{
		{name: "matching active schedule", match: true},
		{name: "already consumed schedule", match: true, change: func(s *client.ScheduleDescription) { s.Schedule.State.RemainingActions = 0 }},
		{name: "foreign owner", change: func(s *client.ScheduleDescription) { s.Memo = nil }},
		{name: "missing original recipe", change: func(s *client.ScheduleDescription) { delete(s.Memo.Fields, finiteScheduleRecipeKey) }},
		{name: "wrong task queue", change: func(s *client.ScheduleDescription) {
			s.Schedule.Action.(*client.ScheduleWorkflowAction).TaskQueue = "other"
		}},
		{name: "infinite schedule", change: func(s *client.ScheduleDescription) { s.Schedule.State.LimitedActions = false }},
		{name: "excess remaining actions", change: func(s *client.ScheduleDescription) { s.Schedule.State.RemainingActions = 4 }},
		{name: "different interval", change: func(s *client.ScheduleDescription) { s.Schedule.Spec.Intervals[0].Every = time.Second }},
		{name: "paused schedule", change: func(s *client.ScheduleDescription) { s.Schedule.State.Paused = true }},
		{name: "different arguments", change: func(s *client.ScheduleDescription) { s.Schedule.Action.(*client.ScheduleWorkflowAction).Args = nil }},
		{name: "different overlap", change: func(s *client.ScheduleDescription) {
			s.Schedule.Policy.Overlap = enumspb.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			schedule := fixture()
			if tc.change != nil {
				tc.change(schedule)
			}
			require.Equal(t, tc.match, MatchesFiniteSchedule(schedule, id, queue))
		})
	}
	require.False(t, MatchesFiniteSchedule(nil, id, queue))
}
