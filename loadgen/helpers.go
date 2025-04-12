package loadgen

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type (
	Weight     int
	Discrete   interface{ ~int | ~string }
	continious interface{ int64 | time.Duration }

	DiscreteDistributionDist[D Discrete] map[D]Weight
	ContiniousDistribution[C continious] map[C]Weight
)

func (d DiscreteDistributionDist[C]) MarshalJSON() ([]byte, error) {
	res := map[string]string{}
	for value, weight := range d {
		res[fmt.Sprintf("%v", value)] = fmt.Sprintf("%d", weight)
	}
	return json.Marshal(res)
}

func (d *DiscreteDistributionDist[D]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]string
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	result := make(DiscreteDistributionDist[D])
	for valueStr, weightStr := range rawMap {
		weightVal, err := strconv.Atoi(weightStr)
		if err != nil {
			return fmt.Errorf("failed to parse weight '%s': %w", weightStr, err)
		}

		var value D
		switch any(value).(type) {
		case string:
			value = any(valueStr).(D)
		case int:
			intVal, err := strconv.Atoi(valueStr)
			if err != nil {
				return fmt.Errorf("failed to parse int value '%s': %w", valueStr, err)
			}
			value = any(intVal).(D)
		default:
			return fmt.Errorf("unsupported type %T for DiscreteDistribution", value)
		}
		result[value] = Weight(weightVal)
	}

	*d = result
	return nil
}

// Sample returns a random, Discrete value from the distribution.
func (d DiscreteDistributionDist[K]) Sample() (K, bool) {
	var res K
	if len(d) == 0 {
		return res, false
	}

	totalWeight := 0
	keys := make([]K, 0, len(d))
	weights := make([]Weight, 0, len(d))
	for k, w := range d {
		keys = append(keys, k)
		weights = append(weights, w)
		totalWeight += int(w)
	}

	r := rand.Intn(totalWeight)
	cumulativeWeight := 0
	for i, w := range weights {
		cumulativeWeight += int(w)
		if r < cumulativeWeight {
			return keys[i], true
		}
	}
	return res, false
}

func (c ContiniousDistribution[C]) MarshalJSON() ([]byte, error) {
	res := map[string]string{}
	for value, weight := range c {
		var valueStr string
		if v, ok := any(value).(fmt.Stringer); ok {
			valueStr = v.String()
		} else {
			valueStr = fmt.Sprintf("%v", value)
		}
		res[valueStr] = fmt.Sprintf("%d", weight)
	}
	return json.Marshal(res)
}

func (c *ContiniousDistribution[C]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]string
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	result := make(ContiniousDistribution[C])
	for valueStr, weightStr := range rawMap {
		weightVal, err := strconv.Atoi(weightStr)
		if err != nil {
			return fmt.Errorf("failed to parse weight '%s': %w", weightStr, err)
		}

		var value C
		switch any(value).(type) {
		case int64:
			intVal, err := strconv.ParseInt(valueStr, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse int value '%s': %w", valueStr, err)
			}
			value = C(intVal)
		case time.Duration:
			durVal, err := time.ParseDuration(valueStr)
			if err != nil {
				return fmt.Errorf("failed to parse duration value '%s': %w", valueStr, err)
			}
			value = C(durVal)
		default:
			return fmt.Errorf("unsupported type %T for ContiniousDistribution", value)
		}
		result[value] = Weight(weightVal)
	}

	*c = result
	return nil
}

// Sample returns a random, continuous number from the distribution.
func (c ContiniousDistribution[C]) Sample() (C, bool) {
	if len(c) == 0 {
		return 0, false
	}
	if len(c) == 1 {
		for v := range c {
			return v, true
		}
	}

	keys := make([]C, 0, len(c))
	cdf := make([]int, len(c))
	totalWeight := 0
	for k := range c {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	for i, k := range keys {
		totalWeight += int(c[k])
		cdf[i] = totalWeight
	}

	r := rand.Intn(totalWeight)
	idx := sort.SearchInts(cdf, r)
	if idx == 0 {
		idx = 1
	}
	v1, v2 := keys[idx-1], keys[idx]
	return C(int64(v1) + rand.Int63n(int64(v2)-int64(v1)+1)), true
}

// VisibilityCountIsEventually ensures that some visibility query count matches the provided
// expected number within the provided time limit.
func VisibilityCountIsEventually(
	ctx context.Context,
	client client.Client,
	request *workflowservice.CountWorkflowExecutionsRequest,
	expectedCount int,
	waitAtMost time.Duration,
) error {
	deadline := time.Now().Add(waitAtMost)
	for {
		visibilityCount, err := client.CountWorkflow(ctx, request)
		if err != nil {
			return fmt.Errorf("failed to count workflows in visibility: %w", err)
		}
		if visibilityCount.Count == int64(expectedCount) {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("expected %d workflows in visibility, got %d after waiting %v",
				expectedCount, visibilityCount.Count, waitAtMost)
		}
		time.Sleep(5 * time.Second)
	}
}
