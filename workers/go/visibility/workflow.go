// Package visibility implements the visibilityStressWorker workflow for the
// visibility stress test scenario.
//
// This workflow is intentionally simple: it receives a list of CSA update
// instructions, executes them with sleeps in between, then completes (or
// intentionally fails/times out). The executor controls all the complexity;
// the workflow just follows instructions.
package visibility

import (
	"fmt"
	"strings"
	"time"

	vstypes "github.com/temporalio/omes/loadgen/visibility"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// VisibilityStressWorkerWorkflow processes CSA update instructions from its input,
// then reaches a terminal state (completed, failed, or timed out).
//
// Each CSA update group triggers one UpsertTypedSearchAttributes call, with a
// configurable delay between updates. After all updates:
//   - If ShouldTimeout: sleeps for 24h (execution timeout kills it → TimedOut status)
//   - If ShouldFail: returns an ApplicationError → Failed status
//   - Otherwise: returns nil → Completed status
func VisibilityStressWorkerWorkflow(ctx workflow.Context, input *vstypes.VisibilityWorkerInput) error {
	if input == nil {
		return nil
	}

	// Execute each CSA update group sequentially with delays between them.
	for i, group := range input.CSAUpdates {
		typedAttrs, err := buildTypedSearchAttributes(group.Attributes)
		if err != nil {
			return fmt.Errorf("failed to build typed attributes for update %d: %w", i, err)
		}
		if err := workflow.UpsertTypedSearchAttributes(ctx, typedAttrs...); err != nil {
			return fmt.Errorf("failed to upsert search attributes (update %d): %w", i, err)
		}
		// Sleep between updates (but not after the last one).
		if i < len(input.CSAUpdates)-1 && input.Delay > 0 {
			_ = workflow.Sleep(ctx, input.Delay)
		}
	}

	// Intentional timeout: sleep for 24h, which exceeds the execution timeout
	// set by the executor. The server will terminate the workflow with TimedOut status.
	if input.ShouldTimeout {
		_ = workflow.Sleep(ctx, 24*time.Hour)
	}

	// Intentional failure: return an application error to produce a Failed status.
	if input.ShouldFail {
		return temporal.NewApplicationError("intentional failure", "VS_INTENTIONAL")
	}

	return nil
}

// buildTypedSearchAttributes converts a raw map[string]any (from JSON deserialization)
// into typed search attribute updates that the Temporal SDK requires.
//
// The type of each CSA is inferred from its name prefix:
//   - VS_Int_*      → int64 (JSON float64 → int64 cast)
//   - VS_Keyword_*  → string
//   - VS_Bool_*     → bool
//   - VS_Double_*   → float64
//   - VS_Text_*     → string (uses KeyString in Go SDK for Text SA type)
//   - VS_Datetime_* → time.Time (parsed from RFC3339 string)
func buildTypedSearchAttributes(attrs map[string]any) ([]temporal.SearchAttributeUpdate, error) {
	updates := make([]temporal.SearchAttributeUpdate, 0, len(attrs))
	for name, val := range attrs {
		update, err := buildOneAttribute(name, val)
		if err != nil {
			return nil, fmt.Errorf("CSA %q: %w", name, err)
		}
		updates = append(updates, update)
	}
	return updates, nil
}

// buildOneAttribute creates a single typed search attribute update by dispatching
// on the CSA name prefix.
func buildOneAttribute(name string, val any) (temporal.SearchAttributeUpdate, error) {
	switch {
	case strings.HasPrefix(name, "VS_Int_"):
		// JSON numbers arrive as float64; cast to int64 for the Int SA type.
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 (JSON number) for Int CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyInt64(name).ValueSet(int64(f)), nil

	case strings.HasPrefix(name, "VS_Keyword_"):
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for Keyword CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyKeyword(name).ValueSet(s), nil

	case strings.HasPrefix(name, "VS_Bool_"):
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool for Bool CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyBool(name).ValueSet(b), nil

	case strings.HasPrefix(name, "VS_Double_"):
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 for Double CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyFloat64(name).ValueSet(f), nil

	case strings.HasPrefix(name, "VS_Text_"):
		// Go SDK uses NewSearchAttributeKeyString for Text SA type.
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for Text CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyString(name).ValueSet(s), nil

	case strings.HasPrefix(name, "VS_Datetime_"):
		// Datetime values are serialized as RFC3339 strings in JSON.
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string (RFC3339) for Datetime CSA, got %T", val)
		}
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Datetime CSA value %q: %w", s, err)
		}
		return temporal.NewSearchAttributeKeyTime(name).ValueSet(t), nil

	default:
		return nil, fmt.Errorf("unrecognized CSA prefix in %q", name)
	}
}
