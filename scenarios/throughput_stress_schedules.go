package scenarios

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	stableScheduleKind    = "stable"
	iterationScheduleKind = "iteration"
	scheduleReleaseKey    = "schedule_release"
	scheduleReleaseValue  = "true"

	maxBufferedStartVerifications = 5
)

const (
	OmesScheduleKindSearchAttribute = "OmesScheduleKind"
	OmesScheduleIDSearchAttribute   = "OmesScheduleID"
)

type tpsScheduleController struct {
	executor *tpsExecutor
	info     loadgen.ScenarioInfo
}

type scheduledWorkflowObservation struct {
	WorkflowID string
	RunID      string
	StartTime  time.Time
	CloseTime  time.Time
	Status     enums.WorkflowExecutionStatus
}

func newTpsScheduleController(executor *tpsExecutor, info loadgen.ScenarioInfo) *tpsScheduleController {
	return &tpsScheduleController{
		executor: executor,
		info:     info,
	}
}

func (c *tpsScheduleController) RegisterSearchAttributes(ctx context.Context) error {
	if !c.executor.config.Schedules.Enabled {
		return nil
	}
	if err := c.info.RegisterDefaultSearchAttributes(ctx); err != nil {
		return err
	}
	for _, attr := range []string{OmesScheduleKindSearchAttribute, OmesScheduleIDSearchAttribute} {
		if err := loadgen.InitSearchAttribute(ctx, c.info, attr); err != nil {
			return err
		}
	}
	return nil
}

func (c *tpsScheduleController) StartStableSchedules(ctx context.Context) error {
	if !c.executor.config.Schedules.Enabled || c.executor.config.Schedules.StableCount == 0 {
		return nil
	}

	for i := range c.executor.config.Schedules.StableCount {
		scheduleID := c.stableScheduleID(i)
		handle, err := c.createSchedule(ctx, scheduleID, stableScheduleKind, c.stablePolicy(i), c.executor.config.Schedules.StableWorkflowDuration, false, true)
		if err != nil {
			return fmt.Errorf("create stable schedule %q: %w", scheduleID, err)
		}
		if _, err := handle.Describe(ctx); err != nil {
			return fmt.Errorf("describe stable schedule %q: %w", scheduleID, err)
		}
	}

	now := time.Now()
	c.executor.lock.Lock()
	c.executor.state.StableSchedulesCreated = true
	if c.executor.state.StableWindowStart.IsZero() {
		c.executor.state.StableWindowStart = now
	}
	c.executor.lock.Unlock()
	return nil
}

func (c *tpsScheduleController) ExecuteIterationSchedules(ctx context.Context, run *loadgen.Run) error {
	if !c.executor.config.Schedules.Enabled || c.executor.config.Schedules.IterationSchedulesPerIteration == 0 {
		return nil
	}

	for i := range c.executor.config.Schedules.IterationSchedulesPerIteration {
		scheduleID := c.iterationScheduleID(run.Iteration, i)
		handle, err := c.createSchedule(
			ctx,
			scheduleID,
			iterationScheduleKind,
			enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL,
			c.executor.config.Schedules.IterationWorkflowDuration,
			true,
			false,
		)
		if err != nil {
			return fmt.Errorf("create iteration schedule %q: %w", scheduleID, err)
		}
		if _, err := handle.Describe(ctx); err != nil {
			return fmt.Errorf("describe iteration schedule %q: %w", scheduleID, err)
		}
		if err := handle.Update(ctx, client.ScheduleUpdateOptions{
			DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
				schedule := input.Description.Schedule
				if schedule.State == nil {
					schedule.State = &client.ScheduleState{}
				}
				schedule.State.Note = fmt.Sprintf("throughput_stress iteration %d schedule %d", run.Iteration, i)
				return &client.ScheduleUpdate{Schedule: &schedule}, nil
			},
		}); err != nil {
			return fmt.Errorf("update iteration schedule %q: %w", scheduleID, err)
		}
		if err := c.sleepOperationInterval(ctx); err != nil {
			return err
		}
		if err := handle.Trigger(ctx, client.ScheduleTriggerOptions{
			Overlap: enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL,
		}); err != nil {
			return fmt.Errorf("trigger iteration schedule %q: %w", scheduleID, err)
		}
		if err := c.waitForScheduledWorkflowCount(ctx, iterationScheduleKind, scheduleID, enums.WORKFLOW_EXECUTION_STATUS_COMPLETED, 1, c.executor.config.Schedules.VisibilityTimeout); err != nil {
			return fmt.Errorf("wait for iteration scheduled workflow %q: %w", scheduleID, err)
		}
		if _, err := handle.Describe(ctx); err != nil {
			return fmt.Errorf("describe completed iteration schedule %q: %w", scheduleID, err)
		}
		if err := c.deleteSchedule(ctx, handle); err != nil {
			return fmt.Errorf("delete iteration schedule %q: %w", scheduleID, err)
		}

		c.executor.lock.Lock()
		c.executor.state.CompletedIterationScheduledWorkflows++
		c.executor.lock.Unlock()
	}
	return nil
}

func (c *tpsScheduleController) FinalizeStableSchedules(ctx context.Context) error {
	if !c.executor.config.Schedules.Enabled || c.executor.config.Schedules.StableCount == 0 {
		return nil
	}

	c.executor.lock.Lock()
	stableWindowStart := c.executor.state.StableWindowStart
	c.executor.lock.Unlock()
	if !stableWindowStart.IsZero() && c.executor.config.Schedules.StableWindow > 0 {
		if remaining := time.Until(stableWindowStart.Add(c.executor.config.Schedules.StableWindow)); remaining > 0 {
			if err := sleepContext(ctx, remaining); err != nil {
				return err
			}
		}
	}

	c.executor.lock.Lock()
	if c.executor.state.StableWindowEnd.IsZero() {
		c.executor.state.StableWindowEnd = time.Now()
	}
	c.executor.lock.Unlock()

	var errs []error
	if err := c.VerifyStableScheduleExpectedStarts(ctx); err != nil {
		errs = append(errs, err)
	}

	for i := range c.executor.config.Schedules.StableCount {
		handle := c.info.Client.ScheduleClient().GetHandle(ctx, c.stableScheduleID(i))
		if err := handle.Pause(ctx, client.SchedulePauseOptions{Note: "throughput_stress stable verification complete"}); err != nil {
			if !isNotFound(err) {
				errs = append(errs, fmt.Errorf("pause stable schedule %q: %w", c.stableScheduleID(i), err))
			}
		}
		if err := c.deleteSchedule(ctx, handle); err != nil {
			errs = append(errs, fmt.Errorf("delete stable schedule %q: %w", c.stableScheduleID(i), err))
		}
	}

	if err := c.releaseRunningScheduledWorkflows(ctx, stableScheduleKind, ""); err != nil {
		errs = append(errs, err)
	}
	if err := c.waitForNoRunningScheduledWorkflows(ctx, stableScheduleKind, c.executor.config.Schedules.VisibilityTimeout); err != nil {
		errs = append(errs, err)
	}

	completed, err := c.countScheduledWorkflows(ctx, stableScheduleKind, "", enums.WORKFLOW_EXECUTION_STATUS_COMPLETED)
	if err != nil {
		errs = append(errs, err)
	} else {
		c.executor.lock.Lock()
		c.executor.state.CompletedStableScheduledWorkflows = completed
		c.executor.state.StableSchedulesFinalized = true
		c.executor.lock.Unlock()
	}

	return errors.Join(errs...)
}

func (c *tpsScheduleController) VerifyStableScheduleExpectedStarts(ctx context.Context) error {
	var errs []error

	for i := range c.executor.config.Schedules.StableCount {
		scheduleID := c.stableScheduleID(i)
		policy := c.stablePolicy(i)

		matchingTimes, err := c.listMatchingTimes(ctx, scheduleID)
		if err != nil {
			errs = append(errs, fmt.Errorf("list matching times for %q: %w", scheduleID, err))
			continue
		}
		if len(matchingTimes) == 0 {
			errs = append(errs, fmt.Errorf("schedule %q had no matching times in stable window", scheduleID))
			continue
		}

		minStarts := expectedStableStarts(policy, len(matchingTimes))
		if policy == enums.SCHEDULE_OVERLAP_POLICY_SKIP && minStarts < 2 {
			minStarts = 2
		}
		if err := c.releaseUntilScheduledWorkflowCount(ctx, scheduleID, minStarts); err != nil {
			errs = append(errs, fmt.Errorf("verify stable schedule %q starts: %w", scheduleID, err))
			continue
		}

		c.info.Logger.Infof(
			"Stable schedule verification: scheduleID=%s policy=%s matchingTimes=%d expectedStartsAtLeast=%d",
			scheduleID,
			policy,
			len(matchingTimes),
			minStarts,
		)
	}

	if len(errs) == 0 {
		c.executor.lock.Lock()
		c.executor.state.MatchingTimesVerified = true
		c.executor.lock.Unlock()
	}
	return errors.Join(errs...)
}

func (c *tpsScheduleController) createSchedule(
	ctx context.Context,
	scheduleID string,
	kind string,
	overlap enums.ScheduleOverlapPolicy,
	workflowDuration time.Duration,
	paused bool,
	triggerImmediately bool,
) (client.ScheduleHandle, error) {
	action := &client.ScheduleWorkflowAction{
		ID:        fmt.Sprintf("w-%s", scheduleID),
		Workflow:  "kitchenSink",
		Args:      []any{c.scheduledWorkflowInput(workflowDuration, c.completionModeFor(kind))},
		TaskQueue: loadgen.TaskQueueForRun(c.info.RunID),
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(loadgen.OmesExecutionIDSearchAttribute).ValueSet(c.info.ExecutionID),
			temporal.NewSearchAttributeKeyString(OmesScheduleKindSearchAttribute).ValueSet(kind),
			temporal.NewSearchAttributeKeyString(OmesScheduleIDSearchAttribute).ValueSet(scheduleID),
		),
	}

	endAfter := c.executor.config.Schedules.StableWindow + c.executor.config.Schedules.VisibilityTimeout + time.Minute
	if kind == iterationScheduleKind {
		endAfter = c.executor.config.Schedules.IterationWorkflowDuration + c.executor.config.Schedules.VisibilityTimeout + time.Minute
	}
	if endAfter <= 0 {
		endAfter = 10 * time.Minute
	}

	handle, err := c.info.Client.ScheduleClient().Create(ctx, client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			Intervals: []client.ScheduleIntervalSpec{
				{Every: c.scheduleIntervalFor(kind)},
			},
			EndAt: time.Now().Add(endAfter),
		},
		Action:             action,
		Overlap:            overlap,
		Paused:             paused,
		TriggerImmediately: triggerImmediately,
		TypedSearchAttributes: temporal.NewSearchAttributes(
			temporal.NewSearchAttributeKeyString(OmesScheduleKindSearchAttribute).ValueSet(kind),
			temporal.NewSearchAttributeKeyString(OmesScheduleIDSearchAttribute).ValueSet(scheduleID),
		),
	})
	if err != nil {
		if isAlreadyExists(err) {
			return c.info.Client.ScheduleClient().GetHandle(ctx, scheduleID), nil
		}
		return nil, err
	}
	return handle, nil
}

func (c *tpsScheduleController) scheduledWorkflowInput(workflowDuration time.Duration, completionMode string) *WorkflowInput {
	actions := []*Action{
		NewTimerAction(workflowDuration),
	}
	if completionMode == ScheduleCompletionModeRelease {
		actions = append(actions, NewAwaitWorkflowStateAction(scheduleReleaseKey, scheduleReleaseValue))
	}
	actions = append(actions, NewEmptyReturnResultAction())
	return &WorkflowInput{
		InitialActions: []*ActionSet{
			{
				Actions:    actions,
				Concurrent: false,
			},
		},
	}
}

func (c *tpsScheduleController) completionModeFor(kind string) string {
	if kind == iterationScheduleKind {
		return ScheduleCompletionModeTimer
	}
	return c.executor.config.Schedules.StableCompletionMode
}

func (c *tpsScheduleController) scheduleIntervalFor(kind string) time.Duration {
	if kind == iterationScheduleKind {
		return 24 * time.Hour
	}
	return c.executor.config.Schedules.StableInterval
}

func (c *tpsScheduleController) stablePolicy(index int) enums.ScheduleOverlapPolicy {
	policies := c.executor.config.Schedules.OverlapPolicies
	if len(policies) == 0 {
		return enums.SCHEDULE_OVERLAP_POLICY_SKIP
	}
	return policies[index%len(policies)]
}

func (c *tpsScheduleController) stableScheduleID(index int) string {
	return fmt.Sprintf("sched-%s-%s-stable-%d", c.info.RunID, c.info.ExecutionID, index)
}

func (c *tpsScheduleController) iterationScheduleID(iteration, index int) string {
	return fmt.Sprintf("sched-%s-%s-iter-%d-%d", c.info.RunID, c.info.ExecutionID, iteration, index)
}

func (c *tpsScheduleController) listMatchingTimes(ctx context.Context, scheduleID string) ([]time.Time, error) {
	c.executor.lock.Lock()
	start := c.executor.state.StableWindowStart
	end := c.executor.state.StableWindowEnd
	c.executor.lock.Unlock()
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return nil, fmt.Errorf("invalid stable matching window start=%v end=%v", start, end)
	}

	resp, err := c.info.Client.WorkflowService().ListScheduleMatchingTimes(ctx, &workflowservice.ListScheduleMatchingTimesRequest{
		Namespace:  c.info.Namespace,
		ScheduleId: scheduleID,
		StartTime:  timestamppb.New(start),
		EndTime:    timestamppb.New(end),
	})
	if err != nil {
		return nil, err
	}
	times := make([]time.Time, 0, len(resp.StartTime))
	for _, ts := range resp.StartTime {
		times = append(times, ts.AsTime())
	}
	return times, nil
}

func expectedStableStarts(policy enums.ScheduleOverlapPolicy, matchingTimes int) int {
	switch policy {
	case enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE:
		return min(matchingTimes, 2)
	case enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL:
		return min(matchingTimes, maxBufferedStartVerifications)
	case enums.SCHEDULE_OVERLAP_POLICY_SKIP:
		return min(matchingTimes, 1)
	default:
		return min(matchingTimes, 1)
	}
}

func (c *tpsScheduleController) releaseUntilScheduledWorkflowCount(ctx context.Context, scheduleID string, minStarts int) error {
	if minStarts <= 0 {
		return nil
	}
	deadline := time.Now().Add(c.executor.config.Schedules.VisibilityTimeout)
	for {
		count, err := c.countScheduledWorkflows(ctx, stableScheduleKind, scheduleID, enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED)
		if err != nil {
			return err
		}
		if count >= minStarts {
			return nil
		}
		if err := c.releaseRunningScheduledWorkflows(ctx, stableScheduleKind, scheduleID); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("expected at least %d starts for schedule %q, got %d", minStarts, scheduleID, count)
		}
		if err := sleepContext(ctx, min(500*time.Millisecond, time.Until(deadline))); err != nil {
			return err
		}
	}
}

func (c *tpsScheduleController) releaseRunningScheduledWorkflows(ctx context.Context, kind, scheduleID string) error {
	observed, err := c.listScheduledWorkflows(ctx, kind, scheduleID, enums.WORKFLOW_EXECUTION_STATUS_RUNNING)
	if err != nil {
		return err
	}
	for _, wf := range observed {
		if err := c.signalRelease(ctx, wf.WorkflowID, wf.RunID); err != nil {
			return err
		}
	}

	if kind == stableScheduleKind {
		for i := range c.executor.config.Schedules.StableCount {
			id := c.stableScheduleID(i)
			if scheduleID != "" && scheduleID != id {
				continue
			}
			handle := c.info.Client.ScheduleClient().GetHandle(ctx, id)
			desc, err := handle.Describe(ctx)
			if err != nil {
				if isNotFound(err) {
					continue
				}
				return err
			}
			for _, wf := range desc.Info.RunningWorkflows {
				if err := c.signalRelease(ctx, wf.WorkflowID, wf.FirstExecutionRunID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (c *tpsScheduleController) signalRelease(ctx context.Context, workflowID, runID string) error {
	err := c.info.Client.SignalWorkflow(ctx, workflowID, runID, "do_actions_signal", &DoSignal_DoSignalActions{
		Variant: &DoSignal_DoSignalActions_DoActions{
			DoActions: SingleActionSet(
				NewSetWorkflowStateAction(scheduleReleaseKey, scheduleReleaseValue),
			),
		},
	})
	if isNotFound(err) {
		return nil
	}
	return err
}

func (c *tpsScheduleController) waitForScheduledWorkflowCount(
	ctx context.Context,
	kind string,
	scheduleID string,
	status enums.WorkflowExecutionStatus,
	minCount int,
	waitAtMost time.Duration,
) error {
	deadline := time.Now().Add(waitAtMost)
	var lastCount int
	for {
		count, err := c.countScheduledWorkflows(ctx, kind, scheduleID, status)
		if err != nil {
			return err
		}
		lastCount = count
		if count >= minCount {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("expected at least %d scheduled workflows for kind=%q scheduleID=%q status=%s, got %d after %v",
				minCount, kind, scheduleID, status, lastCount, waitAtMost)
		}
		if err := sleepContext(ctx, min(500*time.Millisecond, time.Until(deadline))); err != nil {
			return err
		}
	}
}

func (c *tpsScheduleController) waitForNoRunningScheduledWorkflows(ctx context.Context, kind string, waitAtMost time.Duration) error {
	deadline := time.Now().Add(waitAtMost)
	for {
		count, err := c.countScheduledWorkflows(ctx, kind, "", enums.WORKFLOW_EXECUTION_STATUS_RUNNING)
		if err != nil {
			return err
		}
		if count == 0 {
			return nil
		}
		if err := c.releaseRunningScheduledWorkflows(ctx, kind, ""); err != nil {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("expected no running scheduled workflows for kind=%q, got %d after %v", kind, count, waitAtMost)
		}
		if err := sleepContext(ctx, min(500*time.Millisecond, time.Until(deadline))); err != nil {
			return err
		}
	}
}

func (c *tpsScheduleController) countScheduledWorkflows(ctx context.Context, kind, scheduleID string, status enums.WorkflowExecutionStatus) (int, error) {
	resp, err := c.info.Client.CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
		Namespace: c.info.Namespace,
		Query:     c.scheduleVisibilityQuery(kind, scheduleID, status),
	})
	if err != nil {
		return 0, err
	}
	maxInt := int64(^uint(0) >> 1)
	if resp.Count > maxInt {
		return 0, fmt.Errorf("scheduled workflow count too large: %d", resp.Count)
	}
	return int(resp.Count), nil
}

func (c *tpsScheduleController) listScheduledWorkflows(ctx context.Context, kind, scheduleID string, status enums.WorkflowExecutionStatus) ([]scheduledWorkflowObservation, error) {
	var observations []scheduledWorkflowObservation
	var nextPageToken []byte
	for {
		resp, err := c.info.Client.ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     c.info.Namespace,
			Query:         c.scheduleVisibilityQuery(kind, scheduleID, status),
			NextPageToken: nextPageToken,
		})
		if err != nil {
			return nil, err
		}
		for _, execution := range resp.Executions {
			observations = append(observations, scheduledWorkflowObservation{
				WorkflowID: execution.Execution.GetWorkflowId(),
				RunID:      execution.Execution.GetRunId(),
				StartTime:  timestampAsTime(execution.GetStartTime()),
				CloseTime:  timestampAsTime(execution.GetCloseTime()),
				Status:     execution.GetStatus(),
			})
		}
		if len(resp.NextPageToken) == 0 {
			return observations, nil
		}
		nextPageToken = resp.NextPageToken
	}
}

func (c *tpsScheduleController) scheduleVisibilityQuery(kind, scheduleID string, status enums.WorkflowExecutionStatus) string {
	query := fmt.Sprintf("%s='%s' AND %s='%s'",
		loadgen.OmesExecutionIDSearchAttribute, c.info.ExecutionID,
		OmesScheduleKindSearchAttribute, kind)
	if scheduleID != "" {
		query += fmt.Sprintf(" AND %s='%s'", OmesScheduleIDSearchAttribute, scheduleID)
	}
	if status != enums.WORKFLOW_EXECUTION_STATUS_UNSPECIFIED {
		query += fmt.Sprintf(" AND ExecutionStatus = '%s'", status)
	}
	return query
}

func (c *tpsScheduleController) deleteSchedule(ctx context.Context, handle client.ScheduleHandle) error {
	err := handle.Delete(ctx)
	if isNotFound(err) {
		return nil
	}
	return err
}

func (c *tpsScheduleController) sleepOperationInterval(ctx context.Context) error {
	if c.executor.config.Schedules.APIOperationInterval <= 0 {
		return nil
	}
	return sleepContext(ctx, c.executor.config.Schedules.APIOperationInterval)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isAlreadyExists(err error) bool {
	var alreadyExists *serviceerror.AlreadyExists
	return errors.As(err, &alreadyExists)
}

func isNotFound(err error) bool {
	var notFound *serviceerror.NotFound
	return errors.As(err, &notFound)
}

func timestampAsTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

func parseTpsScheduleOverlapPolicies(policyStr string) ([]enums.ScheduleOverlapPolicy, error) {
	var policies []enums.ScheduleOverlapPolicy
	for policy := range strings.SplitSeq(policyStr, ",") {
		switch strings.ToLower(strings.TrimSpace(policy)) {
		case "":
			continue
		case "skip":
			policies = append(policies, enums.SCHEDULE_OVERLAP_POLICY_SKIP)
		case "buffer_one":
			policies = append(policies, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE)
		case "buffer_all":
			policies = append(policies, enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL)
		case "allow_all":
			policies = append(policies, enums.SCHEDULE_OVERLAP_POLICY_ALLOW_ALL)
		default:
			return nil, fmt.Errorf("unknown overlap policy %q", policy)
		}
	}
	if len(policies) == 0 {
		return nil, fmt.Errorf("at least one overlap policy is required")
	}
	return policies, nil
}

func hasRequiredStablePolicies(policies []enums.ScheduleOverlapPolicy) bool {
	required := map[enums.ScheduleOverlapPolicy]bool{
		enums.SCHEDULE_OVERLAP_POLICY_SKIP:       false,
		enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ONE: false,
		enums.SCHEDULE_OVERLAP_POLICY_BUFFER_ALL: false,
	}
	for _, policy := range policies {
		if _, ok := required[policy]; ok {
			required[policy] = true
		}
	}
	for _, present := range required {
		if !present {
			return false
		}
	}
	return true
}
