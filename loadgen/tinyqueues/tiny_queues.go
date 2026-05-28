// Package tinyqueues holds the shared signal/input types for the tiny_queues
// scenario. Both the omes scenario driver (which sends signals) and the Go
// worker workflow (which receives them) import this package so the on-the-
// wire payload shapes stay in lockstep.
//
// This is a port of bench-go's TinyQueueParameters / TinyQueueMessage —
// only fields meaningful to the first-cut omes scenario are retained.
package tinyqueues

// ResourceEventSignalName is the signal name used to deliver TinyQueueMessage
// events. Matches bench-go to keep behaviour parity.
const ResourceEventSignalName = "resource_event"

// WorkflowName is the name used when registering and starting the tiny
// queues workflow. Workers in other languages must register under this same
// name once they are ported.
const WorkflowName = "tinyQueueWorkflow"

// TinyQueueInput is the workflow input. Mirrors a subset of bench-go's
// TinyQueueParameters meaningful to the omes scenario.
type TinyQueueInput struct {
	// MaximumQueueLifetimeSeconds bounds how long the workflow stays open
	// waiting for more signals before it shuts down (or continues-as-new
	// if it still has unprocessed events).
	MaximumQueueLifetimeSeconds int
	// MaxProcessTimePerMessageSeconds is plumbed through for parity but is
	// currently unused in the simplified processor (see worker TODO).
	MaxProcessTimePerMessageSeconds int
	// ContinueAsNewState carries unprocessed events forward when the
	// workflow continues-as-new. Nil on initial start.
	ContinueAsNewState *TinyQueueState
}

// TinyQueueMessage is the per-signal payload. EventID is the logical
// position of the event in the per-workflow sequence (1-indexed).
type TinyQueueMessage struct {
	EventID                         int
	MaxProcessTimePerMessageSeconds int
}

// TinyQueueState is the persisted processor state across continue-as-new.
type TinyQueueState struct {
	PreviousEventID int
	Items           map[int]TinyQueueMessage
}

// QueryableState is returned by the "current_state" query handler.
type QueryableState struct {
	ProcessedEvents int
}
