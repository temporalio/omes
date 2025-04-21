package loadgen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDistribution(t *testing.T) {
	t.Run("String distribution", func(t *testing.T) {
		jsonData := `{"small":10, "medium":20, "large":70}`
		var dist Distribution[string]

		err := json.Unmarshal([]byte(jsonData), &dist)
		assert.NoError(t, err)
		assert.Equal(t, Weight(10), dist["small"])
		assert.Equal(t, Weight(20), dist["medium"])
		assert.Equal(t, Weight(70), dist["large"])
		assert.Len(t, dist, 3)

		value, ok := dist.Sample()
		assert.True(t, ok)
		assert.Contains(t, []string{"small", "medium", "large"}, value)

		marshaled, err := json.Marshal(dist)
		assert.NoError(t, err)

		var parsedDist Distribution[string]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Int distribution", func(t *testing.T) {
		jsonData := `{"1":25, "5":25, "10":50}`
		var dist Distribution[int]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.NoError(t, err)
		assert.Equal(t, Weight(25), dist[1])
		assert.Equal(t, Weight(25), dist[5])
		assert.Equal(t, Weight(50), dist[10])
		assert.Len(t, dist, 3)

		value, ok := dist.Sample()
		assert.True(t, ok)
		assert.Contains(t, []int{1, 5, 10}, value)

		marshaled, err := json.Marshal(dist)
		assert.NoError(t, err)

		var parsedDist Distribution[int]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Duration distribution", func(t *testing.T) {
		jsonData := `{"1s":20, "5s":30, "10s":50}`
		var dist Distribution[time.Duration]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.NoError(t, err)
		assert.Equal(t, Weight(20), dist[1*time.Second])
		assert.Equal(t, Weight(30), dist[5*time.Second])
		assert.Equal(t, Weight(50), dist[10*time.Second])
		assert.Len(t, dist, 3)

		value, ok := dist.Sample()
		assert.True(t, ok)
		assert.True(t, value >= 1*time.Second && value <= 10*time.Second)

		marshaled, err := json.Marshal(dist)
		assert.NoError(t, err)

		var parsedDist Distribution[time.Duration]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Invalid weight", func(t *testing.T) {
		jsonData := `{"small":"invalid", "medium":20}`
		var dist Distribution[string]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal string into Go value of type int")
	})

	t.Run("Invalid int value", func(t *testing.T) {
		jsonData := `{"invalid":10, "5":20}`
		var dist Distribution[int]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse int value 'invalid'")
	})

	t.Run("Invalid time.Duration value", func(t *testing.T) {
		jsonData := `{"10": 10}`
		var dist Distribution[time.Duration]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing unit in duration")
	})
}
