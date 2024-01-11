package scenarios_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/temporalio/omes/scenarios"
)

func TestExponentialSample(t *testing.T) {
	t.Parallel()
	n := 10
	g := rand.New(rand.NewSource(6174))
	counts := make([]int, n)
	for i := 0; i < 10000; i++ {
		counts[scenarios.ExponentialSample(n, 1.0, g.Float64())]++
	}
	assert.Equal(t, []int{6407, 2293, 813, 295, 120, 48, 16, 5, 1, 2}, counts,
		"Counts should be roughly proportional to the exponential distribution")
}
