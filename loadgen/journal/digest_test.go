package journal

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDigestSurvivesTransportAndOrdersItems(t *testing.T) {
	state := New("lane")
	state.Epoch = 3
	state.Contract = []byte("contract")
	state.Counts = Counts{Allocated: 9, Succeeded: 4, Failed: 2, Excluded: 1, Unresolved: 1}
	state.Next = 10
	state.Stopping = true
	state.Last = &Commit{Ordinal: 9, Outcome: Failed, Reason: "original failure"}
	state.Pending = &Iteration{Ordinal: 10, Number: 19, Attempts: 2, Items: map[string]Item{"z": {Plan: []byte("z"), Calls: 2, Deadline: time.Now()}, "a": {Plan: []byte("a"), Observation: []byte("unknown"), Done: true, Unresolved: true}}}
	raw, err := json.Marshal(state)
	require.NoError(t, err)
	var decoded State
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, state.Digest(), decoded.Digest())
	require.LessOrEqual(t, len(raw), state.encodedSizeBound())
	reversed := state.Clone()
	reversed.Pending.Items = map[string]Item{"a": state.Pending.Items["a"], "z": state.Pending.Items["z"]}
	require.Equal(t, state.Digest(), reversed.Digest())
	mutations := []func(*State){func(s *State) { s.Epoch++ }, func(s *State) { s.Contract[0] = 'X' }, func(s *State) { s.Counts.Failed++ }, func(s *State) { s.Last.Outcome = Succeeded }, func(s *State) { s.Pending.Attempts++ }, func(s *State) { i := s.Pending.Items["z"]; i.Calls++; s.Pending.Items["z"] = i }, func(s *State) { i := s.Pending.Items["a"]; i.Unresolved = false; s.Pending.Items["a"] = i }, func(s *State) { s.Stopping = false }}
	for _, mutate := range mutations {
		other := state.Clone()
		mutate(&other)
		require.NotEqual(t, state.Digest(), other.Digest())
	}
}
func TestOversizePlanIsRejectedWithoutChangingIntent(t *testing.T) {
	state := apply(t, New("lane"), Command{Kind: "claim", Epoch: 1})
	state = apply(t, state, Command{Kind: "begin", Epoch: 1, Ordinal: 1, Number: 1})
	previous := state.Clone()
	_, err := state.Apply(Command{Kind: "plan", Epoch: 1, Ordinal: 1, Key: "oversize", Data: make([]byte, MaxStateBytes)})
	require.Error(t, err)
	require.Equal(t, previous, state)
}
