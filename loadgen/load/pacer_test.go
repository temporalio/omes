package load

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPacerLongPhase(t *testing.T) {
	duration := time.Duration(math.MaxInt64)
	level := 1 / duration.Seconds()
	p, err := NewPacer([]Phase{{Name: "hold", Duration: duration, From: level, To: level}})
	require.NoError(t, err)

	wait, done := p.Next(0, 0)
	require.Equal(t, duration, wait)
	require.False(t, done)
}

func TestPacerRate(t *testing.T) {
	p, err := NewPacer([]Phase{
		{Name: "hold", Duration: time.Second, From: 2, To: 2},
		{Name: "up", Duration: 2 * time.Second, From: 0, To: 4},
		{Name: "down", Duration: time.Second, From: 4, To: 0},
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		elapsed time.Duration
		want    float64
	}{{0, 2}, {time.Second, 0}, {2 * time.Second, 2}, {3 * time.Second, 4}, {3500 * time.Millisecond, 2}, {4 * time.Second, 0}} {
		require.Equal(t, tc.want, p.Rate(tc.elapsed))
	}
}

func TestPacerNextRamp(t *testing.T) {
	p, err := NewPacer([]Phase{{Name: "up", Duration: 4 * time.Second, From: 0, To: 8}})
	require.NoError(t, err)

	wait, done := p.Next(0, 0)
	require.Equal(t, time.Second, wait)
	require.False(t, done)

	wait, done = p.Next(time.Second, 1)
	require.Equal(t, 414213563*time.Nanosecond, wait)
	require.False(t, done)

	wait, done = p.Next(4*time.Second, 2)
	require.Equal(t, time.Duration(0), wait)
	require.True(t, done)
}

func TestPacerZeroAndMissedStarts(t *testing.T) {
	p, err := NewPacer([]Phase{
		{Name: "base", Duration: time.Second},
		{Name: "hold", Duration: 2 * time.Second, From: 2, To: 2},
	})
	require.NoError(t, err)

	wait, done := p.Next(0, 0)
	require.Equal(t, 1500*time.Millisecond, wait)
	require.False(t, done)
	require.Equal(t, float64(0), p.Rate(500*time.Millisecond))

	wait, done = p.Next(2250*time.Millisecond, 1)
	require.Equal(t, 250*time.Millisecond, wait)
	require.False(t, done)

	wait, done = p.Next(3*time.Second, 1)
	require.Equal(t, time.Duration(0), wait)
	require.True(t, done)
}

func TestPacerRejectsInvalidPhases(t *testing.T) {
	for _, phases := range [][]Phase{
		nil,
		{{Name: "bad", Duration: -time.Second}},
		{{Name: "bad", Duration: time.Second, From: -1}},
	} {
		_, err := NewPacer(phases)
		require.Error(t, err)
	}
}
