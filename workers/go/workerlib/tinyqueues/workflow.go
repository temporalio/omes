// Package tinyqueues implements the worker-side workflow for the tiny_queues
// scenario. It is a port of bench-go's workflows/tinyqueues workflow:
// the workflow receives "resource event" signals carrying EventIDs that may
// arrive out of order, buffers them by EventID, processes any contiguous
// prefix in order, and (in bench-go) optionally continues-as-new with any
// remaining unprocessed events when its lifetime timer fires.
//
// For the first cut of the omes port we keep this Go-only and intentionally
// skip the local-activity "process event" step (kept as a TODO). Workers in
// other languages (Java/Python/TS) are out of scope for this first cut.
package tinyqueues

import (
	"time"

	tq "github.com/temporalio/omes/loadgen/tinyqueues"
	"go.temporal.io/sdk/log"
	"go.temporal.io/sdk/workflow"
)

// TinyQueueWorkflow buffers signal-delivered events and processes them in
// EventID order. It mirrors bench-go's TinyQueuesWorkflow semantics:
//
//   - Sets up a "current_state" query that returns the highest processed
//     EventID seen so far.
//   - Pumps signals into a state map keyed by EventID.
//   - After each signal, drains any contiguous prefix starting from
//     PreviousEventID+1.
//   - When the lifetime timer fires, drains remaining signals and either
//     completes (no leftover) or continues-as-new with leftover state.
//
// TODO(re-263): bench-go runs a (local) activity per processed event with
// per-message StartToCloseTimeout = MaxProcessTimePerMessageSeconds+30s. We
// elide that for the first cut to keep the worker self-contained; revisit
// if signal-processing throughput vs. activity scheduling is part of what
// we want to measure here.
func TinyQueueWorkflow(ctx workflow.Context, input tq.TinyQueueInput) error {
	logger := workflow.GetLogger(ctx)

	state := &tq.TinyQueueState{Items: map[int]tq.TinyQueueMessage{}}
	if input.ContinueAsNewState != nil {
		state.PreviousEventID = input.ContinueAsNewState.PreviousEventID
		if len(input.ContinueAsNewState.Items) > 0 {
			state.Items = input.ContinueAsNewState.Items
		}
	}

	if err := workflow.SetQueryHandler(ctx, "current_state", func() (tq.QueryableState, error) {
		return tq.QueryableState{ProcessedEvents: state.PreviousEventID}, nil
	}); err != nil {
		logger.Warn("failed to register current_state query handler", "error", err)
		return err
	}

	eventCh := workflow.GetSignalChannel(ctx, tq.ResourceEventSignalName)

	timerDuration := time.Hour
	if input.MaximumQueueLifetimeSeconds > 0 {
		timerDuration = time.Duration(input.MaximumQueueLifetimeSeconds) * time.Second
	}

	closeWorkflow := false
	selector := workflow.NewSelector(ctx)
	selector.AddFuture(workflow.NewTimer(ctx, timerDuration), func(_ workflow.Future) {
		closeWorkflow = true
	})
	selector.AddReceive(eventCh, func(c workflow.ReceiveChannel, _ bool) {
		var msg tq.TinyQueueMessage
		c.Receive(ctx, &msg)
		addEvent(state, msg, logger)
		processInOrder(state, logger)
	})

	for !closeWorkflow || selector.HasPending() {
		selector.Select(ctx)
	}

	// Drain any signals that arrived after the timer fired but before we
	// observed it, then process whatever is now contiguous.
	var msg tq.TinyQueueMessage
	for eventCh.ReceiveAsync(&msg) {
		addEvent(state, msg, logger)
	}
	processInOrder(state, logger)

	if len(state.Items) > 0 {
		// Carry remaining unprocessed events forward into a new run so a
		// late-arriving lower EventID can still unblock the queue.
		return workflow.NewContinueAsNewError(ctx, TinyQueueWorkflow, tq.TinyQueueInput{
			MaximumQueueLifetimeSeconds:     input.MaximumQueueLifetimeSeconds,
			MaxProcessTimePerMessageSeconds: input.MaxProcessTimePerMessageSeconds,
			ContinueAsNewState:              state,
		})
	}
	return nil
}

func addEvent(s *tq.TinyQueueState, msg tq.TinyQueueMessage, logger log.Logger) {
	if msg.EventID <= s.PreviousEventID {
		logger.Debug("dedupe already processed event", "lastEventID", s.PreviousEventID, "eventID", msg.EventID)
		return
	}
	s.Items[msg.EventID] = msg
}

func processInOrder(s *tq.TinyQueueState, logger log.Logger) {
	nextID := s.PreviousEventID + 1
	for {
		_, ok := s.Items[nextID]
		if !ok {
			break
		}
		// TODO(re-263): bench-go executes a "tinyqueues-process-event"
		// (local) activity here with a timeout derived from
		// MaxProcessTimePerMessageSeconds. Omit for the first cut.
		logger.Debug("processed event", "eventID", nextID)
		delete(s.Items, nextID)
		nextID++
	}
	s.PreviousEventID = nextID - 1
}
