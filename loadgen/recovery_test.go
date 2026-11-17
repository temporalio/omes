package loadgen

import (
	"context"
	"errors"
	"fmt"
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
	require.True(t, OnlyIterationFailuresFor(errors.Join(&IterationFailuresError{Stream: "main", Failed: 1}, &IterationFailuresError{Stream: "operators", Failed: 1}), "main", "operators"))
	require.False(t, OnlyIterationFailuresFor(errors.Join(&IterationFailuresError{Stream: "operators", Failed: 1}, fatal), "main", "operators"))
}

func TestJournalDriverStopUnblocksRateWait(t *testing.T) {
	backend := &testJournal{}
	info := recoveryInfo(backend)
	info.Configuration.MaxIterationsPerSecond = 0.001
	stop := make(chan struct{})
	info.Recovery.Stop = stop
	result := make(chan error, 1)
	go func() {
		result <- (&GenericExecutor{Execute: func(context.Context, *Run) error { return errors.New("must not admit an iteration") }}).Run(t.Context(), info)
	}()
	close(stop)
	select {
	case err := <-result:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("stop waited for the rate-limit interval")
	}
	require.Zero(t, backend.states["lane"].Counts.Allocated)
}
func TestJournalDriverStopDrainsAndPreservesFailures(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(fmt.Sprint(failed), func(t *testing.T) {
			backend := &testJournal{}
			info := recoveryInfo(backend)
			info.Configuration.Iterations = 2
			stop := make(chan struct{})
			info.Recovery.Stop = stop
			started := make(chan struct{})
			finish := make(chan struct{})
			result := make(chan error, 1)
			executor := GenericExecutor{Execute: func(ctx context.Context, run *Run) error {
				if _, err := run.Journal.Plan(ctx, "effect", time.Time{}, func() ([]byte, error) { return []byte("admitted intent"), nil }); err != nil {
					return err
				}
				if err := run.Journal.Issue(ctx, "effect"); err != nil {
					return err
				}
				close(started)
				<-finish
				if err := run.Journal.Observe(ctx, "effect", []byte("terminal outcome")); err != nil {
					return err
				}
				if failed {
					return errors.New("original target failed")
				}
				return nil
			}}
			go func() { result <- executor.Run(t.Context(), info) }()
			<-started
			close(stop)
			close(finish)
			err := <-result
			require.Equal(t, uint64(1), backend.states["lane"].Counts.Allocated)
			if failed {
				require.True(t, OnlyIterationFailures(err, "main"))
				require.Equal(t, uint64(1), backend.states["lane"].Counts.Failed)
			} else {
				require.NoError(t, err)
				require.Equal(t, uint64(1), backend.states["lane"].Counts.Succeeded)
			}
		})
	}
}

func TestJournalDriverDoesNotRetrySettledFailureOrFinalUnknown(t *testing.T) {
	for _, unknown := range []bool{false, true} {
		t.Run(fmt.Sprint(unknown), func(t *testing.T) {
			backend := &testJournal{}
			info := recoveryInfo(backend)
			info.Configuration.MaxIterationAttempts = 3
			calls := 0
			executor := &GenericExecutor{Execute: func(ctx context.Context, run *Run) error {
				calls++
				if _, err := run.Journal.Plan(ctx, "effect", time.Time{}, func() ([]byte, error) { return []byte("intent"), nil }); err != nil {
					return err
				}
				if unknown {
					return journal.ErrUnresolved
				}
				if err := run.Journal.Observe(ctx, "effect", []byte("known target failure")); err != nil {
					return err
				}
				return errors.Join(journal.ErrTerminal, errors.New("target failed"))
			}}
			err := executor.Run(t.Context(), info)
			require.Equal(t, 1, calls)
			if unknown {
				require.ErrorIs(t, err, journal.ErrUnresolved)
				info.Recovery.Epoch = 2
				require.ErrorIs(t, executor.Run(t.Context(), info), journal.ErrUnresolved)
				require.Equal(t, 1, calls)
				require.Equal(t, uint64(1), backend.states["lane"].Counts.Unresolved)
			} else {
				require.True(t, OnlyIterationFailures(err, "main"))
				require.Equal(t, journal.Counts{Allocated: 1, Failed: 1}, backend.states["lane"].Counts)
			}
		})
	}
}
