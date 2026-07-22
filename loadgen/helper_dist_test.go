package loadgen

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributionField(t *testing.T) {
	const testSeed = int64(12345) // Fixed seed for deterministic testing

	t.Run("Discrete Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1":10,"5":20,"10":70}}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				Distribution: &DiscreteDistribution[int64]{
					Weights: map[int64]int{
						1:  10,
						5:  20,
						10: 70,
					},
				},
				DistType: "discrete",
			}
			assert.EqualExportedValues(t, expected, df)

			// Test with fixed seed for exact deterministic results
			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(10), value)

			// Test that same seed produces same result
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with different specific seeds for exact results
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, int64(1), value3)
		})

		t.Run("Float32", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1.5":10,"5.25":20,"10.75":70}}`
			var df DistributionField[float32]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[float32]{
				Distribution: &DiscreteDistribution[float32]{
					Weights: map[float32]int{
						1.5:   10,
						5.25:  20,
						10.75: 70,
					},
				},
				DistType: "discrete",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, float32(10.75), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, float32(1.5), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1s":10,"5s":20,"10s":70}}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				Distribution: &DiscreteDistribution[time.Duration]{
					Weights: map[time.Duration]int{
						1 * time.Second:  10,
						5 * time.Second:  20,
						10 * time.Second: 70,
					},
				},
				DistType: "discrete",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, 10*time.Second, value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			checkJsonRoundtrip(t, df)
		})
	})

	t.Run("Uniform Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":1,"max":100}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				Distribution: &UniformDistribution[int64]{
					Min: 1,
					Max: 100,
				},
				DistType: "uniform",
			}
			assert.EqualExportedValues(t, expected, df)

			// Test exact values with specific seeds
			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(99), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, int64(76), value3)
		})

		t.Run("Float32", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":1.5,"max":99.9}`
			var df DistributionField[float32]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[float32]{
				Distribution: &UniformDistribution[float32]{
					Min: 1.5,
					Max: 99.9,
				},
				DistType: "uniform",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, float32(85.01509), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, float32(38.205994), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":"1s","max":"1m"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				Distribution: &UniformDistribution[time.Duration]{
					Min: 1 * time.Second,
					Max: 1 * time.Minute,
				},
				DistType: "uniform",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, time.Duration(21344346453), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			checkJsonRoundtrip(t, df)
		})
	})

	t.Run("Zipf Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":2.0,"v":1.0,"n":100}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)
			require.NotNil(t, df)

			expected := DistributionField[int64]{
				Distribution: &ZipfDistribution[int64]{
					S: 2.0,
					V: 1.0,
					N: 100,
				},
				DistType: "zipf",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(0), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, int64(1), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Float32", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":2.0,"v":1.0,"n":100}`
			var df DistributionField[float32]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)
			require.NotNil(t, df)

			expected := DistributionField[float32]{
				Distribution: &ZipfDistribution[float32]{
					S: 2.0,
					V: 1.0,
					N: 100,
				},
				DistType: "zipf",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, float32(0), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, float32(1), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":2.0,"v":1.0,"n":100}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				Distribution: &ZipfDistribution[time.Duration]{
					S: 2.0,
					V: 1.0,
					N: 100,
				},
				DistType: "zipf",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, time.Duration(0), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, time.Duration(1), value3)

			checkJsonRoundtrip(t, df)
		})
	})

	t.Run("Normal Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":50,"stdDev":10,"min":30,"max":70}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				Distribution: &NormalDistribution[int64]{
					Mean:   50,
					StdDev: 10,
					Min:    30,
					Max:    70,
				},
				DistType: "normal",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(48), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, int64(65), value3)
		})

		t.Run("Float32", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":50.5,"stdDev":10.2,"min":30.1,"max":70.9}`
			var df DistributionField[float32]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[float32]{
				Distribution: &NormalDistribution[float32]{
					Mean:   50.5,
					StdDev: 10.2,
					Min:    30.1,
					Max:    70.9,
				},
				DistType: "normal",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, float32(47.62513), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values
			rng3 := rand.New(rand.NewSource(42))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, float32(66.34703), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":"20s","stdDev":"5s","min":"1s","max":"60s"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				Distribution: &NormalDistribution[time.Duration]{
					Mean:   20 * time.Second,
					StdDev: 5 * time.Second,
					Min:    1 * time.Second,
					Max:    60 * time.Second,
				},
				DistType: "normal",
			}
			assert.EqualExportedValues(t, expected, df)

			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, time.Duration(18590749775), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			checkJsonRoundtrip(t, df)
		})
	})

	t.Run("Fixed Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"fixed","value":"42"}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				Distribution: &FixedDistribution[int64]{
					Value: 42,
				},
				DistType: "fixed",
			}
			assert.Equal(t, expected.DistType, df.DistType)

			// Test all samples return the same value regardless of seed.
			for i := range 10 {
				rng := rand.New(rand.NewSource(testSeed + int64(i)))
				v, ok := df.Sample(rng)
				assert.True(t, ok)
				assert.Equal(t, int64(42), v)
			}
		})

		// no need to test other dist types ...
	})

	t.Run("Mixture Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":3,"distribution":{"type":"uniform","min":"0","max":"100"}},{"weight":7,"distribution":{"type":"uniform","min":"1000","max":"2000"}}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			// Test exact values with specific seeds
			rng := rand.New(rand.NewSource(testSeed))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(1234), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(testSeed))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			// Test with other specific seeds for different values, including a
			// draw that picks the low-weight component.
			rng3 := rand.New(rand.NewSource(1))
			value3, ok3 := df.Sample(rng3)
			assert.True(t, ok3)
			assert.Equal(t, int64(35), value3)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Nested mixture", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":1,"distribution":{"type":"mixture","components":[{"weight":1,"distribution":{"type":"uniform","min":"0","max":"100"}},{"weight":1,"distribution":{"type":"uniform","min":"1000","max":"2000"}}]}},{"weight":1,"distribution":{"type":"fixed","value":"9999"}}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			rng := rand.New(rand.NewSource(0))
			value, ok := df.Sample(rng)
			assert.True(t, ok)
			assert.Equal(t, int64(63), value)

			// Verify deterministic behavior
			rng2 := rand.New(rand.NewSource(0))
			value2, ok2 := df.Sample(rng2)
			assert.True(t, ok2)
			assert.Equal(t, value, value2)

			checkJsonRoundtrip(t, df)
		})

		t.Run("Single component is valid and always samples that component", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":5,"distribution":{"type":"fixed","value":"777"}}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			for i := range 10 {
				rng := rand.New(rand.NewSource(testSeed + int64(i)))
				v, ok := df.Sample(rng)
				assert.True(t, ok)
				assert.Equal(t, int64(777), v)
			}

			checkJsonRoundtrip(t, df)
		})

		t.Run("Empty components is invalid", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "components cannot be empty")
		})

		t.Run("Non-positive weight is invalid", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":0,"distribution":{"type":"fixed","value":"1"}}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "component 0 weight must be positive")
		})

		t.Run("Null component distribution is invalid", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":1,"distribution":null}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "component 0 is missing a distribution")
		})

		t.Run("Missing component distribution is invalid", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":1}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "component 0 is missing a distribution")
		})

		t.Run("Invalid nested distribution is invalid", func(t *testing.T) {
			jsonData := `{"type":"mixture","components":[{"weight":1,"distribution":{"type":"normal","mean":"5","stdDev":"0","min":"0","max":"10"}}]}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "standard deviation must be positive")
		})
	})

	t.Run("Error Cases", func(t *testing.T) {
		t.Run("Unknown distribution type", func(t *testing.T) {
			jsonData := `{"type":"unknown"}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "unknown distribution type: unknown")
		})

		t.Run("Missing type", func(t *testing.T) {
			jsonData := `{"mean":50,"stdDev":10}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing distribution type")
		})

		t.Run("Nil distribution", func(t *testing.T) {
			var df DistributionField[int64]
			marshaled, err := json.Marshal(df)
			assert.NoError(t, err)
			assert.Equal(t, "null", string(marshaled))

			var parsedDF DistributionField[int64]
			err = json.Unmarshal(marshaled, &parsedDF)
			assert.NoError(t, err)
			assert.Nil(t, parsedDF.Distribution)
		})
	})
}

func checkJsonRoundtrip[T DistValueType](t *testing.T, df DistributionField[T]) {
	t.Helper()

	marshaled, err := json.Marshal(df)
	require.NoError(t, err)

	var unmarshalled DistributionField[T]
	err = json.Unmarshal(marshaled, &unmarshalled)
	require.NoError(t, err)

	// The distribution structs expose no exported fields (and some hold
	// sampling caches), so EqualExportedValues would pass vacuously. Compare
	// the re-marshaled JSON instead: a faithful roundtrip must reproduce it.
	remarshaled, err := json.Marshal(unmarshalled)
	require.NoError(t, err)
	assert.JSONEq(t, string(marshaled), string(remarshaled))
}
