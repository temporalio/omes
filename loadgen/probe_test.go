package loadgen

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// shortenProbeBackoff keeps the retrying tests fast without changing the number
// of attempts, which is what they are about.
func shortenProbeBackoff(t *testing.T) {
	t.Helper()
	original := featureProbeBackoff
	featureProbeBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { featureProbeBackoff = original })
}

func TestRetryableProbeError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		retryable bool
	}{
		// A refusal describes the caller, and says the same thing every time.
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "no"), retryable: false},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "no"), retryable: false},
		{name: "unimplemented", err: status.Error(codes.Unimplemented, "no"), retryable: false},
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "no"), retryable: false},
		{name: "not found", err: status.Error(codes.NotFound, "no"), retryable: false},
		{name: "failed precondition", err: status.Error(codes.FailedPrecondition, "no"), retryable: false},

		// These describe the server's condition, which can change.
		{name: "unavailable", err: status.Error(codes.Unavailable, "later"), retryable: true},
		{name: "deadline exceeded", err: status.Error(codes.DeadlineExceeded, "slow"), retryable: true},
		{name: "resource exhausted", err: status.Error(codes.ResourceExhausted, "busy"), retryable: true},
		{name: "internal", err: status.Error(codes.Internal, "oops"), retryable: true},

		// Not a gRPC status at all: nothing says it is permanent, so retry.
		{name: "plain error", err: errors.New("boom"), retryable: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := retryableProbeError(tc.err); got != tc.retryable {
				t.Errorf("retryable: want %v, got %v", tc.retryable, got)
			}
		})
	}
}

func TestProbeSucceedsWithoutRetrying(t *testing.T) {
	attempts := 0
	got, err := probe(t.Context(), "thing", func(context.Context) (string, error) {
		attempts++
		return "answer", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "answer" {
		t.Errorf("want the value the call returned, got %q", got)
	}
	if attempts != 1 {
		t.Errorf("a call that succeeds should be made once, made %d", attempts)
	}
}

func TestProbeRetriesUntilItSucceeds(t *testing.T) {
	shortenProbeBackoff(t)

	attempts := 0
	got, err := probe(t.Context(), "thing", func(context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", status.Error(codes.Unavailable, "not yet")
		}
		return "answer", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "answer" || attempts != 3 {
		t.Errorf("want the value once available after 3 attempts, got %q after %d", got, attempts)
	}
}

// A probe that never answers has to end, and end the run: guessing at a
// capability would silently change the load.
func TestProbeGivesUpAfterExhaustingBackoff(t *testing.T) {
	shortenProbeBackoff(t)

	attempts := 0
	_, err := probe(t.Context(), "server capabilities", func(context.Context) (string, error) {
		attempts++
		return "", status.Error(codes.Unavailable, "down")
	})
	if err == nil {
		t.Fatal("expected a probe that never answers to fail")
	}
	if want := len(featureProbeBackoff) + 1; attempts != want {
		t.Errorf("want %d attempts, made %d", want, attempts)
	}
	if !strings.Contains(err.Error(), "server capabilities") {
		t.Errorf("error should name what was being probed, got: %v", err)
	}
}

func TestProbeDoesNotRetryAPermanentRejection(t *testing.T) {
	shortenProbeBackoff(t)

	attempts := 0
	_, err := probe(t.Context(), "thing", func(context.Context) (string, error) {
		attempts++
		return "", status.Error(codes.PermissionDenied, "not allowed")
	})
	if err == nil {
		t.Fatal("expected a refused probe to fail")
	}
	if attempts != 1 {
		t.Errorf("a refusal should not be retried; made %d attempts", attempts)
	}
}

// Each attempt is bounded, so a server that answers slowly rather than not at all
// cannot hang the start of the run.
func TestProbeBoundsEachAttempt(t *testing.T) {
	deadline, ok := probeAttemptDeadline(t)
	if !ok {
		t.Fatal("each attempt should carry a deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > featureProbeTimeout {
		t.Errorf("attempt deadline should be within the probe timeout, got %v remaining", remaining)
	}
}

func probeAttemptDeadline(t *testing.T) (time.Time, bool) {
	t.Helper()
	var deadline time.Time
	var ok bool
	_, err := probe(t.Context(), "thing", func(ctx context.Context) (string, error) {
		deadline, ok = ctx.Deadline()
		return "", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return deadline, ok
}

// Cancelling the run stops the probe rather than working through the backoff, and
// the reason stays recognizable.
func TestProbeStopsWhenTheRunIsCancelled(t *testing.T) {
	original := featureProbeBackoff
	featureProbeBackoff = []time.Duration{time.Hour}
	t.Cleanup(func() { featureProbeBackoff = original })

	ctx, cancel := context.WithCancel(t.Context())
	attempts := 0
	done := make(chan error, 1)
	go func() {
		_, err := probe(ctx, "thing", func(context.Context) (string, error) {
			attempts++
			cancel()
			return "", status.Error(codes.Unavailable, "down")
		})
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("want a cancelled context to surface, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("probe kept waiting after the run was cancelled")
	}
	if attempts != 1 {
		t.Errorf("want 1 attempt before cancellation, made %d", attempts)
	}
}
