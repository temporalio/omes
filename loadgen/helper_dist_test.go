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
				distribution: &discreteDistribution[int64]{
					weights: map[int64]int{
						1:  10,
						5:  20,
						10: 70,
					},
				},
				distType: "discrete",
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
				distribution: &discreteDistribution[float32]{
					weights: map[float32]int{
						1.5:   10,
						5.25:  20,
						10.75: 70,
					},
				},
				distType: "discrete",
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

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1s":10,"5s":20,"10s":70}}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: &discreteDistribution[time.Duration]{
					weights: map[time.Duration]int{
						1 * time.Second:  10,
						5 * time.Second:  20,
						10 * time.Second: 70,
					},
				},
				distType: "discrete",
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

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Uniform Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":1,"max":100}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: &uniformDistribution[int64]{
					min: 1,
					max: 100,
				},
				distType: "uniform",
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
				distribution: &uniformDistribution[float32]{
					min: 1.5,
					max: 99.9,
				},
				distType: "uniform",
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

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":"1s","max":"1m"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: &uniformDistribution[time.Duration]{
					min: 1 * time.Second,
					max: 1 * time.Minute,
				},
				distType: "uniform",
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

			checkJsonRoundtrip(t, err, df)
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
				distribution: &zipfDistribution[int64]{
					s: 2.0,
					v: 1.0,
					n: 100,
				},
				distType: "zipf",
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

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Float32", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":2.0,"v":1.0,"n":100}`
			var df DistributionField[float32]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)
			require.NotNil(t, df)

			expected := DistributionField[float32]{
				distribution: &zipfDistribution[float32]{
					s: 2.0,
					v: 1.0,
					n: 100,
				},
				distType: "zipf",
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

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":2.0,"v":1.0,"n":100}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: &zipfDistribution[time.Duration]{
					s: 2.0,
					v: 1.0,
					n: 100,
				},
				distType: "zipf",
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

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Normal Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":50,"stdDev":10,"min":30,"max":70}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: &normalDistribution[int64]{
					mean:   50,
					stdDev: 10,
					min:    30,
					max:    70,
				},
				distType: "normal",
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
				distribution: &normalDistribution[float32]{
					mean:   50.5,
					stdDev: 10.2,
					min:    30.1,
					max:    70.9,
				},
				distType: "normal",
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

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":"20s","stdDev":"5s","min":"1s","max":"60s"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: &normalDistribution[time.Duration]{
					mean:   20 * time.Second,
					stdDev: 5 * time.Second,
					min:    1 * time.Second,
					max:    60 * time.Second,
				},
				distType: "normal",
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

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Fixed Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"fixed","value":"42"}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: &fixedDistribution[int64]{
					value: 42,
				},
				distType: "fixed",
			}
			assert.Equal(t, expected.distType, df.distType)

			// Test all samples return the same value regardless of seed.
			for i := 0; i < 10; i++ {
				rng := rand.New(rand.NewSource(testSeed + int64(i)))
				v, ok := df.Sample(rng)
				assert.True(t, ok)
				assert.Equal(t, int64(42), v)
			}
		})

		// no need to test other dist types ...
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
			assert.Nil(t, parsedDF.distribution)
		})
	})
}

func checkJsonRoundtrip[T distValueType](t *testing.T, err error, df DistributionField[T]) {
	t.Helper()

	marshaled, err := json.Marshal(df)
	assert.NoError(t, err)

	var unmarshalled DistributionField[T]
	err = json.Unmarshal(marshaled, &unmarshalled)
	assert.NoError(t, err)

	assert.EqualExportedValues(t, df, unmarshalled)
}
