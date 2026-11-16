package journal

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Client acknowledges a command only after its resulting state is durable.
type Client interface {
	Apply(context.Context, string, Command) (State, error)
}
type Session struct {
	mu     sync.Mutex
	client Client
	lane   string
	epoch  uint64
	state  State
}

func Claim(ctx context.Context, c Client, lane string, epoch uint64) (*Session, error) {
	s, err := c.Apply(ctx, lane, Command{Kind: "claim", Epoch: epoch})
	if err != nil {
		return nil, err
	}
	return &Session{client: c, lane: lane, epoch: epoch, state: s}, nil
}
func (s *Session) State() State { s.mu.Lock(); defer s.mu.Unlock(); return s.state.Clone() }
func (s *Session) apply(ctx context.Context, c Command) error {
	c.Epoch = s.epoch
	if s.state.Pending != nil {
		c.Ordinal = s.state.Pending.Ordinal
	}
	next, err := s.client.Apply(ctx, s.lane, c)
	if err == nil {
		s.state = next
	}
	if err != nil {
		return errors.Join(ErrPersistence, err)
	}
	return nil
}
func (s *Session) Begin(ctx context.Context, number int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apply(ctx, Command{Kind: "begin", Ordinal: s.state.Next, Number: number})
}
func (s *Session) Attempt(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Pending == nil {
		return ErrConflict
	}
	return s.apply(ctx, Command{Kind: "attempt", Call: s.state.Pending.Attempts + 1})
}
func (s *Session) Plan(ctx context.Context, key string, deadline time.Time, generate func() ([]byte, error)) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.Pending == nil {
		return Item{}, ErrConflict
	}
	if item, ok := s.state.Pending.Items[key]; ok {
		return item, nil
	}
	data, err := generate()
	if err != nil {
		return Item{}, err
	}
	if err = s.apply(ctx, Command{Kind: "plan", Key: key, Data: data, Deadline: deadline}); err != nil {
		return Item{}, err
	}
	return s.state.Pending.Items[key], nil
}
func (s *Session) Issue(ctx context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.state.Pending.Items[key]
	return s.apply(ctx, Command{Kind: "issue", Key: key, Call: item.Calls + 1})
}
func (s *Session) Observe(ctx context.Context, key string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apply(ctx, Command{Kind: "observe", Key: key, Data: data})
}
func (s *Session) Commit(ctx context.Context, outcome Outcome, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apply(ctx, Command{Kind: "commit", Outcome: outcome, Reason: reason})
}

func (s *Session) Value(ctx context.Context, key string, generate func() ([]byte, error)) ([]byte, error) {
	item, err := s.Plan(ctx, key, time.Time{}, generate)
	if err != nil {
		return nil, err
	}
	if !item.Done {
		if err = s.Observe(ctx, key, []byte("frozen")); err != nil {
			return nil, err
		}
	}
	return item.Plan, nil
}
func (s *Session) ObserveStatus(ctx context.Context, key string, data []byte, unresolved bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := Command{Kind: "observe", Key: key, Data: data}
	if unresolved {
		c.Outcome = Unresolved
	}
	return s.apply(ctx, c)
}

func (s *Session) Configure(ctx context.Context, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.apply(ctx, Command{Kind: "configure", Data: data})
}
