package visibility

import "time"

// VisibilityWorkerInput is the input for the visibilityStressWorker workflow.
// Shared between the scenario (which builds inputs) and the worker (which executes them).
type VisibilityWorkerInput struct {
	// Each element is one UpsertSearchAttributes call.
	// May be empty if updatesPerWF rounds to 0.
	CSAUpdates []CSAUpdateGroup `json:"csaUpdates"`
	// Duration to sleep between each CSA update.
	Delay time.Duration `json:"delay"`
	// If true, workflow returns an ApplicationError after completing CSA updates.
	ShouldFail bool `json:"shouldFail"`
	// If true, workflow sleeps for 24h after CSA updates (execution timeout will kill it).
	ShouldTimeout bool `json:"shouldTimeout"`
	// TODO: Add memo update support
	// MemoUpdates   []MemoUpdate `json:"memoUpdates"`
	// MemoSizeBytes int          `json:"memoSizeBytes"`
}

// CSAUpdateGroup represents one UpsertSearchAttributes call.
// Keys are CSA names (e.g., "VS_Int_01"), values are the values to set.
// JSON deserializes numbers as float64; the workflow's type dispatch handles casting.
type CSAUpdateGroup struct {
	Attributes map[string]any `json:"attributes"`
}
