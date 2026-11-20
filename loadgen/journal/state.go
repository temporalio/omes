// Package journal defines bounded, durable iteration recovery state.
package journal

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

const Version = 1
const MaxStateBytes = 1 << 20
const MaxItems = 1024

var (
	ErrStale       = errors.New("stale journal owner")
	ErrConflict    = errors.New("journal identity conflict")
	ErrStopped     = errors.New("journal admission stopped")
	ErrPersistence = errors.New("journal persistence failed")
	ErrUnresolved  = errors.New("target effect remains unresolved")
	ErrTerminal    = errors.New("known terminal iteration failure")
)

type Outcome string

const (
	Succeeded  Outcome = "succeeded"
	Failed     Outcome = "failed"
	Excluded   Outcome = "excluded_by_stop"
	Unresolved Outcome = "unresolved"
)

type Counts struct{ Allocated, Succeeded, Failed, Excluded, Unresolved uint64 }

func (c Counts) Settled() uint64 { return c.Succeeded + c.Failed + c.Excluded + c.Unresolved }

type Item struct {
	Plan        []byte
	Deadline    time.Time
	Calls       uint32
	Observation []byte
	Done        bool
	Unresolved  bool
}
type Iteration struct {
	Ordinal  uint64
	Number   int
	Attempts uint32
	Items    map[string]Item
}
type Commit struct {
	Ordinal uint64
	Outcome Outcome
	Reason  string
}
type State struct {
	Version  int
	Contract []byte
	LaneID   string
	Epoch    uint64
	Next     uint64
	Pending  *Iteration
	Last     *Commit
	Counts   Counts
	Stopping bool
	Sealed   bool
}

func New(lane string) State { return State{Version: Version, LaneID: lane, Next: 1} }
func (s State) Complete() bool {
	return len(s.Contract) > 0 && s.Sealed && s.Pending == nil && s.Counts.Unresolved == 0 && s.Counts.Allocated == s.Counts.Settled()
}

type Command struct {
	Kind           string
	Epoch, Ordinal uint64
	Number         int
	Key            string
	Data           []byte
	Deadline       time.Time
	Call           uint32
	Outcome        Outcome
	Reason         string
}

// Apply returns a new state; rejected commands never change the prior state.
func (s State) Apply(c Command) (State, error) {
	next := s.Clone()
	var err error
	if err = next.apply(c); err != nil {
		return s, err
	}
	if next.encodedSizeBound() > MaxStateBytes {
		return s, errors.New("journal state exceeds payload budget")
	}
	return next, nil
}
func (s *State) apply(c Command) error {
	if s.Version != Version || s.LaneID == "" {
		return errors.New("unsupported or missing journal identity")
	}
	if c.Kind == "claim" {
		if c.Epoch == 0 || c.Epoch < s.Epoch {
			return ErrStale
		}
		if s.Sealed {
			return ErrStopped
		}
		s.Epoch = c.Epoch
		return nil
	}
	if c.Kind == "stop" {
		s.Stopping = true
		return nil
	}
	if c.Kind == "seal" {
		s.Stopping = true
		s.Sealed = true
		return nil
	}
	if c.Epoch != s.Epoch || c.Epoch == 0 {
		return ErrStale
	}
	if s.Sealed {
		return ErrStopped
	}
	if c.Kind == "configure" {
		if len(c.Data) == 0 {
			return ErrConflict
		}
		if len(s.Contract) > 0 {
			if !bytes.Equal(s.Contract, c.Data) {
				return ErrConflict
			}
			return nil
		}
		if s.Counts.Allocated > 0 {
			return ErrConflict
		}
		s.Contract = append([]byte(nil), c.Data...)
		return nil
	}
	if c.Kind == "begin" {
		if s.Pending != nil {
			if s.Pending.Ordinal == c.Ordinal && s.Pending.Number == c.Number {
				return nil
			}
			return ErrConflict
		}
		if s.Stopping {
			return ErrStopped
		}
		if c.Ordinal != s.Next || c.Number < 1 {
			return ErrConflict
		}
		s.Pending = &Iteration{Ordinal: c.Ordinal, Number: c.Number, Items: map[string]Item{}}
		s.Counts.Allocated++
		return nil
	}
	if c.Kind == "commit" && s.Last != nil && *s.Last == (Commit{c.Ordinal, c.Outcome, c.Reason}) {
		return nil
	}
	if s.Pending == nil || c.Ordinal != s.Pending.Ordinal {
		return ErrConflict
	}
	if c.Kind == "attempt" {
		if c.Call == s.Pending.Attempts {
			return nil
		}
		if c.Call != s.Pending.Attempts+1 {
			return ErrConflict
		}
		s.Pending.Attempts = c.Call
		return nil
	}
	if c.Kind == "commit" {
		switch c.Outcome {
		case Succeeded:
			for _, key := range workflow.DeterministicKeys(s.Pending.Items) {
				v := s.Pending.Items[key]
				if !v.Done || v.Unresolved {
					return ErrUnresolved
				}
			}
			s.Counts.Succeeded++
		case Failed:
			s.Counts.Failed++
		case Excluded:
			for _, key := range workflow.DeterministicKeys(s.Pending.Items) {
				v := s.Pending.Items[key]
				if v.Calls > 0 {
					return ErrUnresolved
				}
			}
			s.Counts.Excluded++
		case Unresolved:
			s.Counts.Unresolved++
		default:
			return errors.New("invalid journal outcome")
		}
		if c.Outcome != Unresolved && c.Outcome != Excluded {
			for _, key := range workflow.DeterministicKeys(s.Pending.Items) {
				v := s.Pending.Items[key]
				if !v.Done || v.Unresolved {
					return ErrUnresolved
				}
			}
		}
		s.Last = &Commit{c.Ordinal, c.Outcome, c.Reason}
		s.Pending = nil
		s.Next++
		return nil
	}
	if c.Key == "" {
		return errors.New("journal item key is required")
	}
	item, exists := s.Pending.Items[c.Key]
	switch c.Kind {
	case "plan":
		if exists {
			if !bytes.Equal(item.Plan, c.Data) || !item.Deadline.Equal(c.Deadline) {
				return ErrConflict
			}
			return nil
		}
		if len(s.Pending.Items) >= MaxItems {
			return errors.New("too many journal items")
		}
		item = Item{Plan: append([]byte(nil), c.Data...), Deadline: c.Deadline}
	case "issue":
		if !exists || item.Done {
			return ErrConflict
		}
		if c.Call == item.Calls {
			return nil
		}
		if c.Call != item.Calls+1 {
			return ErrConflict
		}
		item.Calls = c.Call
	case "observe":
		if !exists {
			return ErrConflict
		}
		if item.Done {
			if bytes.Equal(item.Observation, c.Data) && item.Unresolved == (c.Outcome == Unresolved) {
				return nil
			}
			return ErrConflict
		}
		item.Done = true
		item.Unresolved = c.Outcome == Unresolved
		item.Observation = append([]byte(nil), c.Data...)
	default:
		return fmt.Errorf("unsupported journal command %q", c.Kind)
	}
	s.Pending.Items[c.Key] = item
	return nil
}

func (s State) Clone() State {
	out := s
	out.Contract = append([]byte(nil), s.Contract...)
	if s.Pending != nil {
		p := *s.Pending
		p.Items = make(map[string]Item, len(s.Pending.Items))
		for _, k := range workflow.DeterministicKeys(s.Pending.Items) {
			v := s.Pending.Items[k]
			v.Plan = append([]byte(nil), v.Plan...)
			v.Observation = append([]byte(nil), v.Observation...)
			p.Items[k] = v
		}
		out.Pending = &p
	}
	if s.Last != nil {
		last := *s.Last
		out.Last = &last
	}
	return out
}
