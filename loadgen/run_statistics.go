package loadgen

import (
	"context"
	"errors"
	"slices"
	"sync"
	"unicode/utf8"
)

const (
	// statisticsMaxFailures bounds the failures kept for reporting; the failed count is not capped.
	statisticsMaxFailures = 100
	// statisticsMaxErrorBytes bounds each kept error message, so memory stays bounded however
	// long the errors are.
	statisticsMaxErrorBytes = 1024
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
// The zero value is ready to use. Counts stay unavailable (Iterations is nil) until
// a generic executor runs, distinguishing unavailable counts from zero iterations.
type RunStatistics struct {
	mu    sync.Mutex
	state RunCounts
}

// RunCounts is a point-in-time copy of what a run counted: iterations, failures and timeout.
type RunCounts struct {
	Iterations *IterationCounts
	Failures   []IterationFailure
	TimedOut   bool
}

func (s *RunStatistics) Snapshot() RunCounts {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.clone()
}

// clone copies the snapshot so callers can't alias the collector's counts or failures.
func (s RunCounts) clone() RunCounts {
	s.Failures = slices.Clone(s.Failures)
	if s.Iterations != nil {
		counts := *s.Iterations
		s.Iterations = &counts
	}
	return s
}

// runStarted records that a generic executor counts this run's iterations; until then counts are unavailable.
func (s *RunStatistics) runStarted() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
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
		if len(s.state.Failures) < statisticsMaxFailures {
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
	if len(message) <= statisticsMaxErrorBytes {
		return message
	}
	cut := statisticsMaxErrorBytes
	for cut > 0 && !utf8.RuneStart(message[cut]) {
		cut--
	}
	return message[:cut] + "… (truncated)"
}
