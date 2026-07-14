package loadgen

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidateMemoConfig(t *testing.T) {
	t.Run("empty string returns nil config and nil error", func(t *testing.T) {
		config, err := ParseAndValidateMemoConfig("")
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("valid config", func(t *testing.T) {
		config, err := ParseAndValidateMemoConfig(
			`{"probability":0.5,"size":{"type":"fixed","value":"4096"}}`)
		require.NoError(t, err)
		require.NotNil(t, config)
		assert.Equal(t, 0.5, config.Probability)
		require.NotNil(t, config.Size)
	})

	t.Run("probability out of range is rejected", func(t *testing.T) {
		_, err := ParseAndValidateMemoConfig(`{"probability":1.5,"size":{"type":"fixed","value":"1"}}`)
		require.Error(t, err)
	})

	t.Run("size required when probability > 0", func(t *testing.T) {
		_, err := ParseAndValidateMemoConfig(`{"probability":0.5}`)
		require.Error(t, err)
	})

	t.Run("invalid JSON returns an error", func(t *testing.T) {
		_, err := ParseAndValidateMemoConfig("{not valid json")
		require.Error(t, err)
	})
}

func TestMemoConfigSampleMemoBytes(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	t.Run("nil config returns nil", func(t *testing.T) {
		var config *MemoConfig
		assert.Nil(t, config.SampleMemoBytes(rng))
	})

	t.Run("zero probability never selects", func(t *testing.T) {
		dist := NewFixedDistribution[int64](100)
		config := &MemoConfig{Probability: 0, Size: &dist}
		assert.Nil(t, config.SampleMemoBytes(rng))
	})

	t.Run("probability 1 always returns a blob of the sampled size", func(t *testing.T) {
		dist := NewFixedDistribution[int64](512)
		config := &MemoConfig{Probability: 1, Size: &dist}
		blob := config.SampleMemoBytes(rand.New(rand.NewSource(7)))
		require.NotNil(t, blob)
		assert.Len(t, blob, 512)
	})

	t.Run("size is clamped to MaxMemoBytes", func(t *testing.T) {
		dist := NewFixedDistribution[int64](10 * MaxMemoBytes)
		config := &MemoConfig{Probability: 1, Size: &dist}
		blob := config.SampleMemoBytes(rand.New(rand.NewSource(7)))
		assert.Len(t, blob, MaxMemoBytes)
	})

	t.Run("probability is honored over many rolls", func(t *testing.T) {
		dist := NewFixedDistribution[int64](8)
		config := &MemoConfig{Probability: 0.3, Size: &dist}
		rng := rand.New(rand.NewSource(1))
		const iterations = 10000
		hits := 0
		for i := 0; i < iterations; i++ {
			if config.SampleMemoBytes(rng) != nil {
				hits++
			}
		}
		assert.InDelta(t, 0.3, float64(hits)/iterations, 0.05)
	})
}
