package loadgen

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestResolveFeatureOptionsNoopWithoutFeatures(t *testing.T) {
	s := &Scenario{Options: func(o *OptionSet) {
		o.Int("count", 0, "")
	}}
	set := s.MustResolveOptions(nil)

	// A nil client would panic if ResolveFeatureOptions tried to use it. Passing
	// nil here proves the no-features short circuit never touches the client.
	if err := ResolveFeatureOptions(context.Background(), nil, "default", set, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestResolveFeatureOptionsNoopWithNilOptions(t *testing.T) {
	// A scenario built without an Options func (common in tests) has a nil
	// OptionSet; this must not panic.
	if err := ResolveFeatureOptions(context.Background(), nil, "default", nil, zap.NewNop().Sugar()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

// Retrying, giving up, and bounding an attempt are covered in probe_test.go,
// which drives probe directly rather than through a WorkflowService stub.
