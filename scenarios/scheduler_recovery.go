package scenarios

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
)

const (
	scheduleOwnerMemo        = "omes-schedule-owner"
	scheduleOperationCalls   = 4
	scheduleOperationTimeout = 5 * time.Second
	scheduleOperationBackoff = 100 * time.Millisecond
)

var (
	errScheduleRecoveryBudget = errors.New("schedule operation recovery budget exhausted")
	errScheduleIntentMismatch = errors.New("schedule ownership or intent mismatch")
	errScheduleUpdateUnknown  = errors.New("schedule update outcome unknown")
)

type scheduleRecovery struct {
	api     client.ScheduleClient
	options client.ScheduleOptions
	owner   string
	logger  *zap.SugaredLogger
}

type scheduleOperation struct {
	ctx      context.Context
	calls    int
	lastErr  error
	firstErr error
	started  time.Time
	outcome  string
	name     string
	logger   *zap.SugaredLogger
}

func (r *scheduleRecovery) operation(ctx context.Context, name string) (*scheduleOperation, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(ctx, scheduleOperationTimeout)
	// Deletion may clean up after EndAt, but cannot outlive the caller.
	if name != "delete" {
		var endCancel context.CancelFunc
		ctx, endCancel = context.WithDeadline(ctx, r.options.Spec.EndAt)
		parentCancel := cancel
		cancel = func() { endCancel(); parentCancel() }
	}
	return &scheduleOperation{ctx: ctx, name: name, logger: r.logger.With("schedule_id", r.options.ID), outcome: "unresolved", started: time.Now()}, cancel
}

func (b *scheduleOperation) call(fn func(context.Context) error) error {
	if err := b.ctx.Err(); err != nil {
		return errors.Join(b.lastErr, err)
	}
	if b.calls == scheduleOperationCalls {
		return errors.Join(b.lastErr, errScheduleRecoveryBudget)
	}
	if b.calls > 0 {
		timer := time.NewTimer(scheduleOperationBackoff)
		defer timer.Stop()
		select {
		case <-b.ctx.Done():
			return errors.Join(b.lastErr, b.ctx.Err())
		case <-timer.C:
		}
	}
	if err := b.ctx.Err(); err != nil {
		return errors.Join(b.lastErr, err)
	}
	b.calls++
	err := fn(b.ctx)
	if contextErr := b.ctx.Err(); contextErr != nil {
		err = errors.Join(err, contextErr)
	}
	if err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		b.lastErr = err
	}
	return err
}

func (b *scheduleOperation) finish(err error) {
	if b.calls > 1 || err != nil || b.outcome != "acknowledged" {
		b.logger.Infow("Schedule operation recovery", "operation", b.name, "logical_calls", b.calls,
			"outcome", b.outcome, "error_code", serviceerror.ToStatus(err).Code().String(),
			"initial_error_code", serviceerror.ToStatus(b.firstErr).Code().String(), "elapsed", time.Since(b.started))
	}
}

func transientScheduleError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errScheduleRecoveryBudget) {
		return false
	}
	switch serviceerror.ToStatus(err).Code() {
	case codes.DeadlineExceeded, codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

func scheduleMissing(err error) bool { return serviceerror.ToStatus(err).Code() == codes.NotFound }

func scheduleExists(err error) bool {
	return errors.Is(err, temporal.ErrScheduleAlreadyRunning) || serviceerror.ToStatus(err).Code() == codes.AlreadyExists
}

func (r *scheduleRecovery) lookup(b *scheduleOperation) (*client.ScheduleDescription, error) {
	for {
		var d *client.ScheduleDescription
		err := b.call(func(ctx context.Context) (err error) {
			d, err = r.api.GetHandle(ctx, r.options.ID).Describe(ctx)
			return err
		})
		if err == nil {
			err = r.check(d)
		}
		if !transientScheduleError(err) {
			return d, err
		}
	}
}

func (r *scheduleRecovery) check(d *client.ScheduleDescription) error {
	if d == nil || d.Memo == nil || d.Schedule.Spec == nil {
		return errScheduleIntentMismatch
	}
	var owner string
	if err := converter.GetDefaultDataConverter().FromPayload(d.Memo.Fields[scheduleOwnerMemo], &owner); err != nil || owner != r.owner {
		return errScheduleIntentMismatch
	}
	action, ok := d.Schedule.Action.(*client.ScheduleWorkflowAction)
	expected := r.options.Action.(*client.ScheduleWorkflowAction)
	if !ok || action.ID != expected.ID || action.Workflow != expected.Workflow || action.TaskQueue != expected.TaskQueue || !d.Schedule.Spec.EndAt.Equal(r.options.Spec.EndAt) {
		return errScheduleIntentMismatch
	}
	if !schedulePayloadMatches(action.Args, expected.Args) {
		return errScheduleIntentMismatch
	}
	if d.Schedule.Policy == nil || d.Schedule.Policy.Overlap != r.options.Overlap {
		return errScheduleIntentMismatch
	}
	return nil
}

func schedulePayloadMatches(actual, expected []any) bool {
	if len(actual) != len(expected) {
		return false
	}
	for i, value := range actual {
		var data []byte
		switch v := value.(type) {
		case []byte:
			data = v
		case *commonpb.Payload:
			if converter.GetDefaultDataConverter().FromPayload(v, &data) != nil {
				return false
			}
		default:
			return false
		}
		want, ok := expected[i].([]byte)
		if !ok || !bytes.Equal(data, want) {
			return false
		}
	}
	return true
}

func (r *scheduleRecovery) create(ctx context.Context) (err error) {
	b, cancel := r.operation(ctx, "create")
	defer cancel()
	defer func() { b.finish(err) }()
	for {
		err = b.call(func(ctx context.Context) error { _, err := r.api.Create(ctx, r.options); return err })
		if err == nil {
			b.outcome = "acknowledged"
			return nil
		}
		if !transientScheduleError(err) && !scheduleExists(err) {
			return err
		}
		writeErr := err
		_, err = r.lookup(b)
		if err == nil {
			b.outcome = "observed_owned_create"
			return nil
		}
		if !scheduleMissing(err) || scheduleExists(writeErr) {
			return errors.Join(writeErr, err)
		}
		if b.calls == scheduleOperationCalls {
			return errors.Join(writeErr, errScheduleRecoveryBudget, err)
		}
		// A late first create and its retry contend for the same unique ID and frozen intent.
	}
}

func (r *scheduleRecovery) describe(ctx context.Context) (err error) {
	b, cancel := r.operation(ctx, "describe")
	defer cancel()
	defer func() { b.finish(err) }()
	_, err = r.lookup(b)
	if scheduleMissing(err) {
		b.outcome = "not_found_compatible"
		return nil
	}
	if err == nil {
		b.outcome = "acknowledged"
	}
	return err
}

func (r *scheduleRecovery) update(ctx context.Context) (err error) {
	b, cancel := r.operation(ctx, "update")
	defer cancel()
	defer func() { b.finish(err) }()
	witness := "omes-update-" + uuid.NewString()
	b.logger = b.logger.With("mutation_id", witness)
	for {
		prepared := false
		err = b.call(func(ctx context.Context) error {
			return r.api.GetHandle(ctx, r.options.ID).Update(ctx, client.ScheduleUpdateOptions{DoUpdate: func(in client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
				if err := r.check(&in.Description); err != nil {
					return nil, err
				}
				schedule := in.Description.Schedule
				if schedule.State == nil {
					return nil, errScheduleIntentMismatch
				}
				state := *schedule.State
				// A fresh witness distinguishes this update's effect from an identical old payload.
				state.Note = witness
				schedule.State = &state
				action := *schedule.Action.(*client.ScheduleWorkflowAction)
				expected := r.options.Action.(*client.ScheduleWorkflowAction)
				action.Args = expected.Args
				schedule.Action = &action
				prepared = true
				return &client.ScheduleUpdate{Schedule: &schedule}, nil
			}})
		})
		if err == nil {
			b.outcome = "acknowledged"
			return nil
		}
		if scheduleMissing(err) {
			b.outcome = "not_found_compatible"
			return nil
		}
		if !transientScheduleError(err) {
			return err
		}
		// Before DoUpdate, the SDK has only described the schedule.
		if !prepared {
			continue
		}
		writeErr := err
		for {
			var d *client.ScheduleDescription
			d, err = r.lookup(b)
			if err != nil {
				return errors.Join(writeErr, errScheduleUpdateUnknown, err)
			}
			if d.Schedule.State != nil && d.Schedule.State.Note == witness {
				b.outcome = "observed_update_witness"
				return nil
			}
			// The write may still be in flight. Probe within this budget; never replay it.
			if b.calls == scheduleOperationCalls {
				return errors.Join(writeErr, errScheduleUpdateUnknown, errScheduleRecoveryBudget)
			}
		}
	}
}

func (r *scheduleRecovery) delete(ctx context.Context) (err error) {
	b, cancel := r.operation(ctx, "delete")
	defer cancel()
	defer func() { b.finish(err) }()
	for {
		_, err = r.lookup(b)
		if scheduleMissing(err) {
			b.outcome = "observed_absence"
			return nil
		}
		if err != nil {
			return err
		}
		err = b.call(func(ctx context.Context) error { return r.api.GetHandle(ctx, r.options.ID).Delete(ctx) })
		if err == nil {
			b.outcome = "acknowledged"
			return nil
		}
		if scheduleMissing(err) {
			b.outcome = "not_found_compatible"
			return nil
		}
		if !transientScheduleError(err) {
			return err
		}
	}
}
