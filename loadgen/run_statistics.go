package loadgen

import (
	"context"
	"errors"
	"slices"
	"sync"
)

// RunStatistics collects generic executor outcomes independently of scenario callbacks.
// Its zero value is ready for use. Runs that do not use a generic executor leave
// Iterations nil, distinguishing unavailable counts from zero iterations.
type RunStatistics struct {
	mu            sync.Mutex
	iterations    *IterationCounts
	failures      []IterationFailure
	timedOut      bool
}

func (s *RunStatistics) Snapshot() RunCounts {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := RunCounts{Failures: slices.Clone(s.failures), TimedOut: s.timedOut}
	if s.iterations != nil {
		counts := *s.iterations
		snapshot.Iterations = &counts
	}
	return snapshot
}

func (s *RunStatistics) runStarted() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.iterations == nil {
		s.iterations = &IterationCounts{}
	}
}

func (s *RunStatistics) iterationStarted() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.iterations.Started++
}

func (s *RunStatistics) iterationFinished(iteration, attempt int, err error, stopping bool) {
	if s == nil || stopping {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.iterations.Succeeded++
	} else {
		s.iterations.Failed++
		if len(s.failures) < 100 {
			s.failures = append(s.failures, IterationFailure{Iteration: iteration, Attempt: attempt, Error: err.Error()})
		}
	}
}

func (s *RunStatistics) runFinished(err error) {
	if s == nil || !errors.Is(err, context.DeadlineExceeded) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.timedOut = true
}
