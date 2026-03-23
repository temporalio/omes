// Package visibility provides shared types for the visibility stress test scenario.
//
// These types are imported by both the scenario code (which builds workflow inputs)
// and the worker code (which executes the workflow). They live in loadgen/ rather than
// workers/go/ because the scenario (in the main Go module) cannot import from the
// workers/go module (separate Go module).
package visibility

import "time"

// VisibilityWorkerInput is the input for the visibilityStressWorker workflow.
//
// The executor builds one of these per workflow, encoding all the instructions for
// what the workflow should do: which CSAs to update, how long to sleep between updates,
// and whether to intentionally fail or timeout.
type VisibilityWorkerInput struct {
	// Each element is one UpsertSearchAttributes call.
	// May be empty if updatesPerWF rounds to 0 for this workflow.
	CSAUpdates []CSAUpdateGroup `json:"csaUpdates"`

	// Duration to sleep between consecutive CSA updates within the workflow.
	Delay time.Duration `json:"delay"`

	// If true, the workflow returns an ApplicationError after completing all CSA updates.
	// This produces a "Failed" terminal status in the visibility store.
	ShouldFail bool `json:"shouldFail"`

	// If true, the workflow sleeps for 24h after completing CSA updates.
	// The executor sets a short execution timeout, so the workflow will be killed,
	// producing a "TimedOut" terminal status in the visibility store.
	ShouldTimeout bool `json:"shouldTimeout"`

	// TODO: Add memo update support
	// MemoUpdates   []MemoUpdate `json:"memoUpdates"`
	// MemoSizeBytes int          `json:"memoSizeBytes"`
}

// CSAUpdateGroup represents one UpsertSearchAttributes call.
//
// Keys are CSA names (e.g., "VS_Int_01"), values are the values to set.
// The workflow dispatches on the name prefix (VS_Int_, VS_Keyword_, etc.) to
// determine the typed search attribute key.
//
// Note: JSON deserializes all numbers as float64, so the workflow must cast
// float64 → int64 for Int CSAs, and parse string → time.Time for Datetime CSAs.
type CSAUpdateGroup struct {
	Attributes map[string]any `json:"attributes"`
}
