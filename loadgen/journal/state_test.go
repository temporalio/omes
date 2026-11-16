package journal

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func apply(t *testing.T, s State, c Command) State {
	t.Helper()
	n, e := s.Apply(c)
	require.NoError(t, e)
	return n
}
func TestRecoveryAndCommitReceipts(t *testing.T) {
	s := New("main/0")
	s = apply(t, s, Command{Kind: "claim", Epoch: 1})
	s = apply(t, s, Command{Kind: "configure", Epoch: 1, Data: []byte("run contract")})
	s = apply(t, s, Command{Kind: "begin", Epoch: 1, Ordinal: 1, Number: 1})
	s = apply(t, s, Command{Kind: "plan", Epoch: 1, Ordinal: 1, Key: "start", Data: []byte("frozen")})
	s = apply(t, s, Command{Kind: "issue", Epoch: 1, Ordinal: 1, Key: "start", Call: 1})
	raw, e := json.Marshal(s)
	require.NoError(t, e)
	var restored State
	require.NoError(t, json.Unmarshal(raw, &restored))
	restored = apply(t, restored, Command{Kind: "claim", Epoch: 2})
	_, e = restored.Apply(Command{Kind: "observe", Epoch: 1, Ordinal: 1, Key: "start"})
	require.ErrorIs(t, e, ErrStale)
	_, e = restored.Apply(Command{Kind: "commit", Epoch: 2, Ordinal: 1, Outcome: Succeeded})
	require.ErrorIs(t, e, ErrUnresolved)
	restored = apply(t, restored, Command{Kind: "observe", Epoch: 2, Ordinal: 1, Key: "start", Data: []byte("verified original result")})
	commit := Command{Kind: "commit", Epoch: 2, Ordinal: 1, Outcome: Failed, Reason: "original failed"}
	restored = apply(t, restored, commit)
	restored = apply(t, restored, commit)
	require.Equal(t, Counts{Allocated: 1, Failed: 1}, restored.Counts)
	restored = apply(t, restored, Command{Kind: "seal"})
	require.True(t, restored.Complete())
}
func TestIncompleteSealAndConflictingPlans(t *testing.T) {
	s := New("scheduler/0")
	s = apply(t, s, Command{Kind: "claim", Epoch: 1})
	s = apply(t, s, Command{Kind: "begin", Epoch: 1, Ordinal: 1, Number: 3})
	s = apply(t, s, Command{Kind: "plan", Epoch: 1, Ordinal: 1, Key: "update", Data: []byte("original")})
	prior := s.Clone()
	_, e := s.Apply(Command{Kind: "plan", Epoch: 1, Ordinal: 1, Key: "update", Data: []byte("foreign")})
	require.ErrorIs(t, e, ErrConflict)
	require.Equal(t, prior, s)
	s = apply(t, s, Command{Kind: "issue", Epoch: 1, Ordinal: 1, Key: "update", Call: 1})
	_, e = s.Apply(Command{Kind: "commit", Epoch: 1, Ordinal: 1, Outcome: Excluded})
	require.ErrorIs(t, e, ErrUnresolved)
	s = apply(t, s, Command{Kind: "seal"})
	require.False(t, s.Complete())
	require.Equal(t, uint64(1), s.Counts.Allocated)
}
func TestStopAllowsRecoveryButNotAdmission(t *testing.T) {
	s := apply(t, New("main/0"), Command{Kind: "claim", Epoch: 1})
	s = apply(t, s, Command{Kind: "stop"})
	_, e := s.Apply(Command{Kind: "begin", Epoch: 1, Ordinal: 1, Number: 1})
	require.True(t, errors.Is(e, ErrStopped))
	s = apply(t, s, Command{Kind: "claim", Epoch: 2})
	require.True(t, s.Stopping)
}
