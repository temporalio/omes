package visibility

import (
	"fmt"
	"strings"
	"time"

	vstypes "github.com/temporalio/omes/loadgen/visibility"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// VisibilityStressWorkerWorkflow executes CSA updates then completes (or fails/times out).
func VisibilityStressWorkerWorkflow(ctx workflow.Context, input *vstypes.VisibilityWorkerInput) error {
	if input == nil {
		return nil
	}

	for i, group := range input.CSAUpdates {
		typedAttrs, err := buildTypedSearchAttributes(group.Attributes)
		if err != nil {
			return fmt.Errorf("failed to build typed attributes for update %d: %w", i, err)
		}
		if err := workflow.UpsertTypedSearchAttributes(ctx, typedAttrs...); err != nil {
			return fmt.Errorf("failed to upsert search attributes (update %d): %w", i, err)
		}
		if i < len(input.CSAUpdates)-1 && input.Delay > 0 {
			_ = workflow.Sleep(ctx, input.Delay)
		}
	}

	if input.ShouldTimeout {
		// Sleep for 24h; the execution timeout (set by the executor) will kill this.
		_ = workflow.Sleep(ctx, 24*time.Hour)
	}

	if input.ShouldFail {
		return temporal.NewApplicationError("intentional failure", "VS_INTENTIONAL")
	}

	return nil
}

// buildTypedSearchAttributes converts a map[string]any to typed search attribute updates,
// dispatching on the CSA name prefix.
//
// JSON deserializes all numbers as float64, so VS_Int values arrive as float64 and must
// be cast to int64. VS_Datetime values arrive as strings (RFC3339) and must be parsed.
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

func buildOneAttribute(name string, val any) (temporal.SearchAttributeUpdate, error) {
	switch {
	case strings.HasPrefix(name, "VS_Int_"):
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
		s, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for Text CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyString(name).ValueSet(s), nil

	case strings.HasPrefix(name, "VS_Datetime_"):
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
