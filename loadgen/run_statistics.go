package loadgen

import (
	"context"
	"errors"
	"slices"
	"sync"
)

// IterationCounts counts each logical iteration once, regardless of retries.
type IterationCounts struct {
	Started   int64 `json:"started"`
	Succeeded int64 `json:"succeeded"`
	Failed    int64 `json:"failed"`
}

// IterationFailure describes a terminal iteration failure after retries.
type IterationFailure struct {
	Iteration int    `json:"iteration"`
	Attempt   int    `json:"attempt"`
	Error     string `json:"error"`
}

// RunStatistics collects generic executor outcomes independently of scenario callbacks.
// Its zero value is ready for use. Runs that do not use a generic executor leave
// Iterations nil, distinguishing unavailable counts from zero iterations.
type RunStatistics struct {
	mu            sync.Mutex
	configuration RunConfiguration
	iterations    *IterationCounts
	failures      []IterationFailure
	timedOut      bool
}

// RunStatisticsSnapshot contains a copy of the collected counts and failure details.
type RunStatisticsSnapshot struct {
	Configuration RunConfiguration
	Iterations    *IterationCounts
	Failures      []IterationFailure
	TimedOut      bool
}

func (s *RunStatistics) Snapshot() RunStatisticsSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	snapshot := RunStatisticsSnapshot{Configuration: s.configuration, Failures: slices.Clone(s.failures), TimedOut: s.timedOut}
	if s.iterations != nil {
		counts := *s.iterations
		snapshot.Iterations = &counts
	}
	return snapshot
}

func (s *RunStatistics) configure(configuration RunConfiguration) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configuration = configuration
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
