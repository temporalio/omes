package journal

import (
	"crypto/sha256"
	"encoding/binary"

	"go.temporal.io/sdk/workflow"
)

// Digest covers every persisted field using ordered, length-delimited encoding.
func (s State) Digest() [32]byte {
	data := make([]byte, 0, 1024)
	number := func(value uint64) { data = binary.AppendUvarint(data, value) }
	blob := func(value []byte) { number(uint64(len(value))); data = append(data, value...) }
	text := func(value string) { blob([]byte(value)) }
	flag := func(value bool) {
		if value {
			number(1)
		} else {
			number(0)
		}
	}
	number(uint64(s.Version))
	text(s.LaneID)
	blob(s.Contract)
	number(s.Epoch)
	number(s.Next)
	number(s.Counts.Allocated)
	number(s.Counts.Succeeded)
	number(s.Counts.Failed)
	number(s.Counts.Excluded)
	number(s.Counts.Unresolved)
	flag(s.Stopping)
	flag(s.Sealed)
	flag(s.Last != nil)
	if s.Last != nil {
		number(s.Last.Ordinal)
		text(string(s.Last.Outcome))
		text(s.Last.Reason)
	}
	flag(s.Pending != nil)
	if s.Pending != nil {
		number(s.Pending.Ordinal)
		number(uint64(s.Pending.Number))
		number(uint64(s.Pending.Attempts))
		number(uint64(len(s.Pending.Items)))
		for _, key := range workflow.DeterministicKeys(s.Pending.Items) {
			item := s.Pending.Items[key]
			text(key)
			blob(item.Plan)
			blob(item.Observation)
			number(uint64(item.Deadline.Unix()))
			number(uint64(item.Deadline.Nanosecond()))
			number(uint64(item.Calls))
			flag(item.Done)
			flag(item.Unresolved)
		}
	}
	return sha256.Sum256(data)
}

// encodedSizeBound includes worst-case JSON escaping and base64 expansion.
func (s State) encodedSizeBound() int {
	size := 1024 + 6*len(s.LaneID) + 2*len(s.Contract)
	if s.Last != nil {
		size += 128 + 6*(len(s.Last.Outcome)+len(s.Last.Reason))
	}
	if s.Pending != nil {
		for _, key := range workflow.DeterministicKeys(s.Pending.Items) {
			item := s.Pending.Items[key]
			size += 512 + 6*len(key) + 2*(len(item.Plan)+len(item.Observation))
		}
	}
	return size
}
