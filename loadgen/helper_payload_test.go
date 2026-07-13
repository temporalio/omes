package loadgen

import (
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAndValidatePayloadConfig(t *testing.T) {
	t.Run("empty string returns nil config and nil error", func(t *testing.T) {
		config, err := ParseAndValidatePayloadConfig("")
		require.NoError(t, err)
		assert.Nil(t, config)
	})

	t.Run("valid config with discrete size", func(t *testing.T) {
		config, err := ParseAndValidatePayloadConfig(
			`{"size":{"type":"discrete","weights":{"100":1,"200":3}}}`)
		require.NoError(t, err)
		require.NotNil(t, config)
		require.NotNil(t, config.Size)
	})

	t.Run("invalid JSON returns an error", func(t *testing.T) {
		config, err := ParseAndValidatePayloadConfig("{not valid json")
		require.Error(t, err)
		assert.Nil(t, config)
	})
}

func TestPayloadConfigSamplePayloadSize(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	t.Run("nil config returns fallback", func(t *testing.T) {
		var config *PayloadConfig
		assert.Equal(t, 256, config.SamplePayloadSize(rng, 256))
	})

	t.Run("config with nil Size returns fallback", func(t *testing.T) {
		config := &PayloadConfig{}
		assert.Equal(t, 256, config.SamplePayloadSize(rng, 256))
	})

	t.Run("fixed distribution returns the fixed value", func(t *testing.T) {
		dist := NewFixedDistribution[int64](1024)
		config := &PayloadConfig{Size: &dist}
		assert.Equal(t, 1024, config.SamplePayloadSize(rng, 256))
	})

	t.Run("values above int32 max are clamped (no negative-length panic)", func(t *testing.T) {
		dist := NewFixedDistribution[int64](int64(math.MaxInt32) + 5000)
		config := &PayloadConfig{Size: &dist}
		assert.Equal(t, math.MaxInt32, config.SamplePayloadSize(rng, 256))
	})

	t.Run("discrete distribution respects weights over many samples", func(t *testing.T) {
		dist := NewDiscreteDistribution(map[int64]int{100: 70, 200: 25, 300: 5})
		config := &PayloadConfig{Size: &dist}

		const iterations = 10000
		counts := map[int]int{}
		rng := rand.New(rand.NewSource(42))
		for i := 0; i < iterations; i++ {
			counts[config.SamplePayloadSize(rng, 0)]++
		}

		// Only the configured keys should ever appear.
		for k := range counts {
			assert.Contains(t, []int{100, 200, 300}, k)
		}

		// Loose bounds on the proportions to guard against gross weighting bugs.
		assert.InDelta(t, 0.70, float64(counts[100])/iterations, 0.1)
		assert.InDelta(t, 0.25, float64(counts[200])/iterations, 0.1)
		assert.InDelta(t, 0.05, float64(counts[300])/iterations, 0.05)
	})
}
