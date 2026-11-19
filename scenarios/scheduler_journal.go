package scenarios

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/loadgen/journal"
	"go.temporal.io/sdk/client"
)

type frozenSchedule struct {
	Options   client.ScheduleOptions
	Action    client.ScheduleWorkflowAction
	Payload   []byte
	Owner     string
	CleanupAt time.Time
}

func (s *SchedulerExecutor) createJournaledSchedule(ctx context.Context, run *loadgen.Run, index int) (ScheduleState, error) {
	prefix := fmt.Sprintf("schedule/%d", index)
	raw, err := run.Journal.Value(ctx, prefix+"/intent", func() ([]byte, error) {
		id := fmt.Sprintf("sched-%s-%d-%d-%s", run.RunID, run.Iteration, index, uuid.NewString())
		end := time.Now().Add(s.config.SchedulerDurationPerIteration + time.Duration(s.config.ScheduleReadsPerCreation+s.config.ScheduleUpdatesPerCreation)*s.config.OperationInterval + 2*time.Second)
		owner := uuid.NewString()
		return json.Marshal(frozenSchedule{
			Options: client.ScheduleOptions{ID: id, Spec: client.ScheduleSpec{CronExpressions: []string{s.config.CronExpression}, EndAt: end}, Overlap: pickOverlap(s.config.OverlapPolicy, run.Logger), Memo: map[string]any{scheduleOwnerMemo: owner}},
			Action:  client.ScheduleWorkflowAction{ID: "w-" + id, Workflow: s.config.ScheduledWorkflowType, TaskQueue: run.TaskQueue()},
			Payload: make([]byte, s.config.PayloadSize), Owner: owner, CleanupAt: time.Now().Add(s.config.WaitTimeBeforeCleanup),
		})
	})
	if err != nil {
		return ScheduleState{}, errors.Join(journal.ErrPersistence, err)
	}
	var plan frozenSchedule
	if err := json.Unmarshal(raw, &plan); err != nil {
		return ScheduleState{}, err
	}
	plan.Action.Args = []any{plan.Payload}
	plan.Options.Action = &plan.Action
	recovery := &scheduleRecovery{policy: &s.config.RecoveryPolicy, api: run.Client.ScheduleClient(), options: plan.Options, owner: plan.Owner, logger: run.Logger, journal: run.Journal, journalPrefix: prefix, ordinals: map[string]int{}}
	state := ScheduleState{ScheduleID: plan.Options.ID, DeleteAfter: s.config.SchedulerDurationPerIteration, CleanupAt: plan.CleanupAt, operations: recovery}
	return state, recovery.create(ctx)
}

type scheduleOperationPlan struct {
	Witness      string
	Policy       scheduleRecoveryPolicy
	PhasedUpdate bool `json:",omitempty"`
	WitnessCalls int  `json:",omitempty"`
}

type recordedScheduleResult struct {
	Outcome scheduleOutcome
	Failure string
}

func (r *recordedScheduleResult) err() error {
	if r.Failure == "" {
		return nil
	}
	return errors.New(r.Failure)
}

func (b *scheduleOperation) restoreJournal(ctx context.Context, r *scheduleRecovery) {
	b.journal = r.journal
	if b.journal == nil {
		return
	}
	if r.ordinals == nil {
		r.ordinals = map[string]int{}
	}
	ordinal := r.ordinals[b.name]
	r.ordinals[b.name]++
	b.key = fmt.Sprintf("%s/%s/%d", r.journalPrefix, b.name, ordinal)
	deadline, _ := b.ctx.Deadline()
	item, err := b.journal.Plan(ctx, b.key, deadline, func() ([]byte, error) {
		plan := scheduleOperationPlan{Witness: "omes-update-" + uuid.NewString(), Policy: b.recoveryPolicy(), PhasedUpdate: b.name == "update"}
		if plan.PhasedUpdate {
			plan.WitnessCalls = max(6, plan.Policy.Calls)
		}
		return json.Marshal(plan)
	})
	if err != nil {
		b.beginErr = errors.Join(journal.ErrPersistence, err)
		return
	}
	var plan scheduleOperationPlan
	if err := json.Unmarshal(item.Plan, &plan); err != nil {
		b.beginErr = err
		return
	}
	b.witness = plan.Witness
	b.policy = &plan.Policy
	b.restored = item.Calls > 0
	b.phasedUpdate = plan.PhasedUpdate
	b.witnessCalls = plan.WitnessCalls
	b.calls = int(item.Calls)
	if b.witnessCalls > 0 {
		// The first persisted issue belongs to the mutation. Later issues count witness reads.
		b.witnessReads = max(0, b.calls-1)
	}
	if item.Done {
		b.recorded = &recordedScheduleResult{}
		if err := json.Unmarshal(item.Observation, b.recorded); err != nil {
			b.beginErr = err
		}
	}
	b.ctx, b.cancelDeadline = context.WithDeadline(ctx, item.Deadline)
}
func (b *scheduleOperation) persistResult(err error) error {
	if b.cancelDeadline != nil {
		defer b.cancelDeadline()
	}
	if b.journal == nil || b.recorded != nil || b.beginErr != nil || (b.parentCtx != nil && b.parentCtx.Err() != nil) ||
		(b.phasedUpdate && errors.Is(err, journal.ErrPersistence)) {
		return err
	}
	result := recordedScheduleResult{Outcome: b.outcome}
	if err != nil {
		result.Failure = err.Error()
		if len(result.Failure) > 1024 {
			result.Failure = result.Failure[:1024]
		}
	}
	raw, encodeErr := json.Marshal(result)
	if encodeErr != nil {
		return errors.Join(err, encodeErr)
	}
	unknown := b.name != "describe" && b.outcome == outcomeUnresolved && b.calls > 0
	recordCtx, cancel := context.WithTimeout(context.WithoutCancel(b.parentCtx), 30*time.Second)
	defer cancel()
	if persistErr := b.journal.ObserveStatus(recordCtx, b.key, raw, unknown); persistErr != nil {
		return errors.Join(journal.ErrPersistence, err, persistErr)
	}
	return err
}
func (r *scheduleRecovery) recoverUpdate(b *scheduleOperation) error {
	if b.witnessCalls > 0 {
		return r.reconcileUpdate(b)
	}
	for {
		d, err := r.lookup(b)
		if err != nil {
			return errors.Join(errScheduleUpdateUnknown, err)
		}
		if d.Schedule.State != nil && d.Schedule.State.Note == b.witness {
			b.outcome = outcomeObservedWitness
			return nil
		}
	}
}

func (r *scheduleRecovery) reconcileUpdate(b *scheduleOperation) error {
	deadline, _ := b.ctx.Deadline()
	var lastErr error
	for b.witnessReads < b.witnessCalls {
		if err := b.ctx.Err(); err != nil {
			return errors.Join(errScheduleUpdateUnknown, lastErr, err)
		}
		remaining := b.witnessCalls - b.witnessReads
		if b.witnessReads > 0 {
			timer := time.NewTimer(max(b.recoveryPolicy().Backoff, time.Until(deadline)/time.Duration(remaining+1)))
			select {
			case <-timer.C:
			case <-b.ctx.Done():
				timer.Stop()
				return errors.Join(errScheduleUpdateUnknown, lastErr, b.ctx.Err())
			}
		}
		if err := b.journal.Issue(b.ctx, b.key); err != nil {
			return errors.Join(journal.ErrPersistence, err)
		}
		b.witnessReads++
		readCtx, cancel := context.WithTimeout(b.ctx, time.Until(deadline)/time.Duration(remaining+1))
		d, err := r.api.GetHandle(readCtx, r.options.ID).Describe(readCtx)
		cancel()
		if err != nil {
			if ctxErr := b.ctx.Err(); ctxErr != nil {
				return errors.Join(errScheduleUpdateUnknown, err, ctxErr)
			}
			lastErr = err
			if transientScheduleError(err) || errors.Is(err, context.DeadlineExceeded) {
				continue
			}
			return errors.Join(errScheduleUpdateUnknown, err)
		}
		if err := r.check(d); err != nil {
			return errors.Join(errScheduleUpdateUnknown, err)
		}
		if d.Schedule.State != nil && d.Schedule.State.Note == b.witness {
			b.outcome = outcomeObservedWitness
			return nil
		}
	}
	return errors.Join(errScheduleUpdateUnknown, lastErr, errScheduleRecoveryBudget)
}
