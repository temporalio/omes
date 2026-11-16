package loadgen

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/temporalio/omes/loadgen/journal"
)

// RecoveryOptions identifies journals created for this load before its driver starts.
type RecoveryOptions struct {
	Stop       <-chan struct{}
	StopErrors <-chan error

	Contract                []byte
	Client                  journal.Client
	Epoch                   uint64
	Streams                 map[string][]string
	GenerationEnd, DrainEnd time.Time
}

// IterationFailuresError contains only settled failures, without operational errors.
type IterationFailuresError struct {
	Stream string
	Failed uint64
}

func (e *IterationFailuresError) Error() string {
	return fmt.Sprintf("%s completed with %d failed iterations", e.Stream, e.Failed)
}

// OnlyIterationFailures excludes joined infrastructure, observation and component errors.
func OnlyIterationFailures(err error, stream string) bool {
	if err == nil {
		return false
	}
	if failure, ok := err.(*IterationFailuresError); ok {
		return failure.Stream == stream
	}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		found := false
		for _, child := range joined.Unwrap() {
			if child == nil {
				continue
			}
			found = true
			if !OnlyIterationFailures(child, stream) {
				return false
			}
		}
		return found
	}
	if wrapped, ok := err.(interface{ Unwrap() error }); ok {
		return OnlyIterationFailures(wrapped.Unwrap(), stream)
	}
	return false
}

func (r *RecoveryOptions) Totals(ctx context.Context, stream string) (journal.Counts, error) {
	var total journal.Counts
	lanes := r.Streams[stream]
	if len(lanes) == 0 {
		return total, errors.New("required journal stream is absent")
	}
	for _, lane := range lanes {
		s, err := r.Client.Apply(ctx, lane, journal.Command{Kind: "claim", Epoch: r.Epoch})
		if err != nil {
			return total, err
		}
		total.Allocated += s.Counts.Allocated
		total.Succeeded += s.Counts.Succeeded
		total.Failed += s.Counts.Failed
		total.Excluded += s.Counts.Excluded
		total.Unresolved += s.Counts.Unresolved
	}
	return total, nil
}

func (g *genericRun) runJournaled(ctx context.Context) error {
	recovery := g.info.Recovery
	lanes := recovery.Streams[g.info.ScenarioName]
	if recovery.Client == nil || recovery.Epoch == 0 || len(lanes) != g.config.MaxConcurrent || recovery.GenerationEnd.IsZero() || !recovery.DrainEnd.After(recovery.GenerationEnd) || g.config.StartFromIteration != 0 || !g.config.DoNotRegisterSearchAttributes {
		return errors.New("invalid journaled run contract")
	}
	ctx, cancel := context.WithDeadline(ctx, recovery.DrainEnd)
	defer cancel()
	sessions := make([]*journal.Session, len(lanes))
	for i, lane := range lanes {
		s, err := journal.Claim(ctx, recovery.Client, lane, recovery.Epoch)
		if err != nil {
			return err
		}
		if err := s.Configure(ctx, recovery.Contract); err != nil {
			return err
		}
		sessions[i] = s
	}
	if !g.config.DoNotRegisterSearchAttributes {
		if err := g.info.RegisterDefaultSearchAttributes(ctx); err != nil {
			return err
		}
	}
	var ticks <-chan time.Time
	if g.config.MaxIterationsPerSecond > 0 {
		ticker := time.NewTicker(time.Duration(float64(time.Second) / g.config.MaxIterationsPerSecond))
		defer ticker.Stop()
		ticks = ticker.C
	}
	results := make(chan error, len(lanes))
	var workers sync.WaitGroup
	for i, session := range sessions {
		workers.Add(1)
		go func(slot int, s *journal.Session) {
			defer workers.Done()
			err := g.runJournalLane(ctx, s, slot, len(lanes), ticks)
			if err != nil {
				cancel()
			}
			results <- err
		}(i, session)
	}
	workers.Wait()
	close(results)
	var fatal error
	for err := range results {
		fatal = errors.Join(fatal, err)
	}
	if fatal != nil {
		return fatal
	}
	select {
	case err := <-recovery.StopErrors:
		if err != nil {
			return err
		}
	default:
	}
	totals, err := recovery.Totals(ctx, g.info.ScenarioName)
	if err != nil {
		return err
	}
	if totals.Unresolved > 0 {
		return journal.ErrUnresolved
	}
	if totals.Failed > 0 {
		return &IterationFailuresError{Stream: g.info.ScenarioName, Failed: totals.Failed}
	}
	return nil
}

func (g *genericRun) runJournalLane(ctx context.Context, s *journal.Session, slot, width int, ticks <-chan time.Time) error {
	for {
		state := s.State()
		if state.Pending == nil {
			if state.Stopping {
				return nil
			}
			select {
			case <-g.info.Recovery.Stop:
				return nil
			default:
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if !time.Now().Before(g.info.Recovery.GenerationEnd) {
				return nil
			}
			select {
			case <-g.executor.StopNewIterations:
				return nil
			default:
			}
			if state.Next > uint64((math.MaxInt-slot-1)/width)+1 {
				return errors.New("journal iteration number overflow")
			}
			number := slot + int(state.Next-1)*width + 1
			if g.config.Iterations > 0 && number > g.config.Iterations {
				return nil
			}
			if ticks != nil {
				timer := time.NewTimer(time.Until(g.info.Recovery.GenerationEnd))
				select {
				case <-ticks:
					timer.Stop()
				case <-timer.C:
					return nil
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-g.info.Recovery.Stop:
					timer.Stop()
					return nil
				case <-g.executor.StopNewIterations:
					timer.Stop()
					return nil
				}
			}
			if err := s.Begin(ctx, number); err != nil {
				if errors.Is(err, journal.ErrStopped) {
					return nil
				}
				return err
			}
		}
		if err := g.executeJournalIteration(ctx, s); err != nil {
			return err
		}
	}
}

func (g *genericRun) executeJournalIteration(ctx context.Context, s *journal.Session) error {
	pending := s.State().Pending
	run := g.info.NewRun(pending.Number)
	run.Journal = s
	resumed := pending.Attempts > 0
	started := time.Now()
	var executeErr error
	for {
		if !resumed {
			if err := s.Attempt(ctx); err != nil {
				return err
			}
		}
		resumed = false
		executeErr = g.executor.Execute(ctx, run)
		if errors.Is(executeErr, journal.ErrPersistence) {
			return executeErr
		}
		if errors.Is(executeErr, journal.ErrUnresolved) || errors.Is(executeErr, journal.ErrConflict) {
			break
		}
		if executeErr != nil && g.config.HandleExecuteError != nil {
			executeErr = g.config.HandleExecuteError(ctx, run, executeErr)
		}
		if executeErr == nil || ctx.Err() != nil || int(s.State().Pending.Attempts) >= g.config.MaxIterationAttempts {
			break
		}
		select {
		case <-time.After(BaseIterationRetryBackoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(executeErr, journal.ErrPersistence) || (executeErr != nil && len(s.State().Pending.Items) == 0) {
		return executeErr
	}
	outcome := journal.Succeeded
	if executeErr != nil {
		outcome = journal.Failed
	}
	for _, item := range s.State().Pending.Items {
		if !item.Done || item.Unresolved {
			outcome = journal.Unresolved
			break
		}
	}
	reason := ""
	if executeErr != nil {
		reason = truncate(executeErr.Error())
	}
	if err := s.Commit(ctx, outcome, reason); err != nil {
		return err
	}
	if outcome == journal.Unresolved {
		return errors.Join(journal.ErrUnresolved, executeErr)
	}
	if outcome == journal.Failed {
		if g.config.OnIterationFailure != nil {
			g.config.OnIterationFailure(ctx, run, executeErr)
		}
		if !g.config.ContinueOnIterationFailure {
			return executeErr
		}
	} else {
		run.Duration = time.Since(started)
		if g.config.OnCompletion != nil {
			g.config.OnCompletion(ctx, run)
		}
	}
	return nil
}
