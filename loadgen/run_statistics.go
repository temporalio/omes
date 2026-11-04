package loadgen

import (
	"context"
	"errors"
	"slices"
	"sync"
	"unicode/utf8"
)

const (
	// maxIterationFailures bounds the failures kept for reporting; the failed count is not capped.
	maxIterationFailures = 100
	// maxIterationErrorBytes bounds each kept error message, so memory stays bounded however
	// long the errors are.
	maxIterationErrorBytes = 1024
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
	mu    sync.Mutex
	state RunStatisticsSnapshot
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
	snapshot := s.state
	snapshot.Failures = slices.Clone(s.state.Failures)
	if s.state.Iterations != nil {
		counts := *s.state.Iterations
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
	s.state.Configuration = configuration
	if s.state.Iterations == nil {
		s.state.Iterations = &IterationCounts{}
	}
}

func (s *RunStatistics) iterationStarted() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.Iterations.Started++
}

func (s *RunStatistics) iterationFinished(iteration, attempt int, err error, stopping bool) {
	if s == nil || stopping {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err == nil {
		s.state.Iterations.Succeeded++
	} else {
		s.state.Iterations.Failed++
		if len(s.state.Failures) < maxIterationFailures {
			s.state.Failures = append(s.state.Failures, IterationFailure{Iteration: iteration, Attempt: attempt, Error: truncate(err.Error())})
		}
	}
}

func (s *RunStatistics) runFinished(err error) {
	if s == nil || !errors.Is(err, context.DeadlineExceeded) {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.state.TimedOut = true
}

func truncate(message string) string {
	if len(message) <= maxIterationErrorBytes {
		return message
	}
	cut := maxIterationErrorBytes
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut] + "… (truncated)"
}
