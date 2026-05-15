package devserver

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// LogMonitor tails one or more dev-server log files and applies a set of
// substring matchers per file. Each matcher is either positive (MustMatch:
// fail if 0 hits) or negative (MustNotMatch: fail on first hit, with the
// matching lines captured for the error message).
type LogMonitor struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLogMonitor returns a LogMonitor whose lifetime is bound to t.
func NewLogMonitor(t *testing.T) *LogMonitor {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	m := &LogMonitor{
		ctx:    ctx,
		cancel: cancel,
	}
	t.Cleanup(m.Stop)
	return m
}

// Watcher tails one file. Attach matchers with MustMatch / MustNotMatch.
type Watcher struct {
	name     string
	path     string
	matchers []*matcher
}

type matcher struct {
	pattern   string
	mustExist bool
	matches   atomic.Int64
	// captured records the first few matching lines for MustNotMatch error
	// messages. MustMatch doesn't need them — the counter is enough.
	mu       sync.Mutex
	captured []string
}

const capturedSampleLimit = 5

// Watch starts tailing path. Returns a Watcher handle for attaching matchers.
// Existing content is skipped so only output produced after Watch is checked.
func (m *LogMonitor) Watch(t *testing.T, name, path string) *Watcher {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		require.NoError(t, err)
	}

	w := &Watcher{name: name, path: path}

	m.wg.Go(func() {
		defer f.Close()

		reader := bufio.NewReader(f)
		for {
			line, err := reader.ReadString('\n')
			if err == nil {
				w.checkLine(strings.TrimSpace(line))
				continue
			}
			if err != io.EOF {
				// Read failure is itself a finding — synthesize a captured
				// match on a sentinel matcher so AssertAll surfaces it.
				w.matchers = append(w.matchers, &matcher{
					pattern:   fmt.Sprintf("log read error: %v", err),
					mustExist: false,
				})
				w.matchers[len(w.matchers)-1].matches.Add(1)
				return
			}

			select {
			case <-m.ctx.Done():
				return
			case <-time.After(200 * time.Millisecond):
			}
		}
	})

	return w
}

// MustMatch registers a substring that must appear at least once in this
// file before AssertAll is called. Chainable.
func (w *Watcher) MustMatch(pattern string) *Watcher {
	w.matchers = append(w.matchers, &matcher{pattern: pattern, mustExist: true})
	return w
}

// MustNotMatch registers a substring that must never appear in this file.
// Matching lines are captured (up to a small sample) and surfaced in the
// AssertAll error message. Chainable.
func (w *Watcher) MustNotMatch(pattern string) *Watcher {
	w.matchers = append(w.matchers, &matcher{pattern: pattern, mustExist: false})
	return w
}

func (w *Watcher) checkLine(line string) {
	if line == "" {
		return
	}
	for _, m := range w.matchers {
		if !strings.Contains(line, m.pattern) {
			continue
		}
		m.matches.Add(1)
		if !m.mustExist {
			m.mu.Lock()
			if len(m.captured) < capturedSampleLimit {
				m.captured = append(m.captured, line)
			}
			m.mu.Unlock()
		}
	}
}

// AssertAll stops the monitor and fails the test if any registered matcher
// failed its expectation.
func (m *LogMonitor) AssertAll(t *testing.T, watchers ...*Watcher) {
	t.Helper()
	m.Stop()

	var problems []string
	for _, w := range watchers {
		for _, mt := range w.matchers {
			n := mt.matches.Load()
			switch {
			case mt.mustExist && n == 0:
				problems = append(problems, fmt.Sprintf("[%s] expected at least one log line containing %q, got none", w.name, mt.pattern))
			case !mt.mustExist && n > 0:
				mt.mu.Lock()
				sample := strings.Join(mt.captured, "\n  ")
				mt.mu.Unlock()
				problems = append(problems, fmt.Sprintf("[%s] unexpected %d log line(s) containing %q. Sample:\n  %s", w.name, n, mt.pattern, sample))
			}
		}
	}
	require.Empty(t, problems, "log monitor assertions failed:\n%s", strings.Join(problems, "\n"))
}

// Stop ends all watchers. Safe to call multiple times.
func (m *LogMonitor) Stop() {
	m.cancel()
	m.wg.Wait()
}
