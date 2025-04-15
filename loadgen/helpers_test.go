package loadgen

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDiscreteDistribution(t *testing.T) {
	t.Run("String distribution", func(t *testing.T) {
		jsonData := `{"small":"10", "medium":"20", "large":"70"}`
		var dist DiscreteDistributionDist[string]

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

		var parsedDist DiscreteDistributionDist[string]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Int distribution", func(t *testing.T) {
		jsonData := `{"1":"25", "5":"25", "10":"50"}`
		var dist DiscreteDistributionDist[int]

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

		var parsedDist DiscreteDistributionDist[int]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Invalid weight", func(t *testing.T) {
		jsonData := `{"small":"invalid", "medium":"20"}`
		var dist DiscreteDistributionDist[string]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse weight 'invalid'")
	})

	// Test error case - invalid int value
	t.Run("Invalid int value", func(t *testing.T) {
		jsonData := `{"invalid":"10", "5":"20"}`
		var dist DiscreteDistributionDist[int]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse int value 'invalid'")
	})
}

func TestContiniousDistribution(t *testing.T) {
	t.Run("Int distribution", func(t *testing.T) {
		jsonData := `{"100":"30", "200":"30", "300":"40"}`
		var dist ContiniousDistribution[int64]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.NoError(t, err)
		assert.Equal(t, Weight(30), dist[100])
		assert.Equal(t, Weight(30), dist[200])
		assert.Equal(t, Weight(40), dist[300])
		assert.Len(t, dist, 3)

		value, ok := dist.Sample()
		assert.True(t, ok)
		assert.True(t, value >= 100 && value <= 300)

		marshaled, err := json.Marshal(dist)
		assert.NoError(t, err)

		var parsedDist ContiniousDistribution[int64]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Duration distribution", func(t *testing.T) {
		jsonData := `{"1s":"20","5s":"30","10s":"50"}`
		var dist ContiniousDistribution[time.Duration]

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

		var parsedDist ContiniousDistribution[time.Duration]
		err = json.Unmarshal(marshaled, &parsedDist)
		assert.NoError(t, err)
		assert.Equal(t, dist, parsedDist)
	})

	t.Run("Invalid weight", func(t *testing.T) {
		jsonData := `{"100":"thirty", "200":"70"}`
		var dist ContiniousDistribution[int64]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse weight 'thirty'")
	})

	t.Run("Invalid value", func(t *testing.T) {
		jsonData := `{"invalid":"30", "200":"70"}`
		var dist ContiniousDistribution[int64]

		err := json.Unmarshal([]byte(jsonData), &dist)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "strconv.ParseInt")
	})
}
