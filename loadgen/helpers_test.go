package loadgen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDistributionField(t *testing.T) {
	t.Run("Discrete Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1":10,"5":20,"10":70}}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: discreteDistribution[int64]{
					weights: map[int64]int{
						1:  10,
						5:  20,
						10: 70,
					},
				},
				distType: "discrete",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.Contains(t, []int64{1, 5, 10}, value)

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"discrete","weights":{"1s":10,"5s":20,"10s":70}}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: discreteDistribution[time.Duration]{
					weights: map[time.Duration]int{
						1 * time.Second:  10,
						5 * time.Second:  20,
						10 * time.Second: 70,
					},
				},
				distType: "discrete",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.Contains(t, []time.Duration{1 * time.Second, 5 * time.Second, 10 * time.Second}, value)

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Uniform Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":1,"max":100,"steps":10}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: uniformDistribution[int64]{
					min: 1,
					max: 100,
				},
				distType: "uniform",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, int64(1))
			assert.LessOrEqual(t, value, int64(100))

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"uniform","min":"1s","max":"1m"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: uniformDistribution[time.Duration]{
					min: 1 * time.Second,
					max: 1 * time.Minute,
				},
				distType: "uniform",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, 1*time.Second)
			assert.LessOrEqual(t, value, 1*time.Minute)

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Zipf Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":1.5,"v":2.0,"n":100}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)
			require.NotNil(t, df)

			expected := DistributionField[int64]{
				distribution: zipfDistribution[int64]{
					s: 1.5,
					v: 2.0,
					n: 100,
				},
				distType: "zipf",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, int64(0))
			assert.LessOrEqual(t, value, int64(100))

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"zipf","s":1.5,"v":2.0,"n":50}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: zipfDistribution[time.Duration]{
					s: 1.5,
					v: 2.0,
					n: 50,
				},
				distType: "zipf",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, time.Duration(0))

			checkJsonRoundtrip(t, err, df)
		})
	})

	t.Run("Normal Distribution", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":50,"stdDev":1,"min":30,"max":70}`
			var df DistributionField[int64]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[int64]{
				distribution: normalDistribution[int64]{
					mean:   50,
					stdDev: 10,
					min:    30,
					max:    70,
				},
				distType: "normal",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, int64(30))
			assert.LessOrEqual(t, value, int64(70))

			checkJsonRoundtrip(t, err, df)
		})

		t.Run("Duration", func(t *testing.T) {
			jsonData := `{"type":"normal","mean":"20s","stdDev":"1s","min":"1s","max":"60s"}`
			var df DistributionField[time.Duration]

			err := json.Unmarshal([]byte(jsonData), &df)
			require.NoError(t, err)

			expected := DistributionField[time.Duration]{
				distribution: normalDistribution[time.Duration]{
					mean:   20 * time.Second,
					stdDev: 5000.0,
					min:    1 * time.Second,
					max:    60 * time.Second,
				},
				distType: "normal",
			}
			assert.EqualExportedValues(t, expected, df)

			value, ok := df.Sample()
			assert.True(t, ok)
			assert.GreaterOrEqual(t, value, 1*time.Second)
			assert.LessOrEqual(t, value, 60*time.Second)

			checkJsonRoundtrip(t, err, df)
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
