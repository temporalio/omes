package loadgen

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/journal"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type testJournal struct {
	mu         sync.Mutex
	states     map[string]journal.State
	loseCommit bool
}

func (b *testJournal) Apply(_ context.Context, lane string, c journal.Command) (journal.State, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.states == nil {
		b.states = map[string]journal.State{}
	}
	s, ok := b.states[lane]
	if !ok {
		s = journal.New(lane)
	}
	next, err := s.Apply(c)
	if err != nil {
		return s, err
	}
	b.states[lane] = next
	if c.Kind == "commit" && b.loseCommit {
		b.loseCommit = false
		return journal.State{}, errors.New("lost durable commit reply")
	}
	return next.Clone(), nil
}
func recoveryInfo(backend journal.Client) ScenarioInfo {
	return ScenarioInfo{
		ScenarioName: "main", RunID: "run", ExecutionID: "exec", Logger: zap.NewNop().Sugar(), MetricsHandler: client.MetricsNopHandler,
		Configuration: RunConfiguration{Iterations: 1, MaxConcurrent: 1, DoNotRegisterSearchAttributes: true, ContinueOnIterationFailure: true},
		Recovery:      &RecoveryOptions{Client: backend, Epoch: 1, Contract: []byte("fixed run contract"), Streams: map[string][]string{"main": {"lane"}}, GenerationEnd: time.Now().Add(time.Minute), DrainEnd: time.Now().Add(2 * time.Minute)},
	}
}
func TestJournalDriverLostCommitReplyDoesNotDuplicateWork(t *testing.T) {
	backend := &testJournal{loseCommit: true}
	info := recoveryInfo(backend)
	effects := 0
	executor := &GenericExecutor{Execute: func(ctx context.Context, run *Run) error {
		_, err := run.Journal.Plan(ctx, "effect", time.Time{}, func() ([]byte, error) { return []byte("fixed intent"), nil })
		if err != nil {
			return err
		}
		if err := run.Journal.Issue(ctx, "effect"); err != nil {
			return err
		}
		effects++
		return run.Journal.Observe(ctx, "effect", []byte("terminal result"))
	}}
	require.ErrorIs(t, executor.Run(t.Context(), info), journal.ErrPersistence)
	info.Recovery.Epoch = 2
	require.NoError(t, executor.Run(t.Context(), info))
	require.Equal(t, 1, effects)
	totals, err := info.Recovery.Totals(t.Context(), "main")
	require.NoError(t, err)
	require.Equal(t, uint64(1), totals.Succeeded)
}
func TestJournalDriverKeepsFatalErrorsSeparate(t *testing.T) {
	backend := &testJournal{}
	info := recoveryInfo(backend)
	fatal := errors.New("unsupported target protocol")
	err := (&GenericExecutor{Execute: func(context.Context, *Run) error { return fatal }}).Run(t.Context(), info)
	require.ErrorIs(t, err, fatal)
	require.False(t, OnlyIterationFailures(err, "main"))
	require.False(t, OnlyIterationFailures(errors.Join(&IterationFailuresError{Stream: "main", Failed: 1}, fatal), "main"))
	require.False(t, OnlyIterationFailures(&IterationFailuresError{Stream: "operators", Failed: 1}, "main"))
}
