package loadgen

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
)

type (
	Weight                   int
	DistType                 interface{ ~int | ~string | time.Duration }
	Distribution[T DistType] map[T]Weight
)

func (d Distribution[T]) MarshalJSON() ([]byte, error) {
	res := map[string]string{}
	for value, weight := range d {
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

func (d *Distribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]string
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	result := make(Distribution[T])
	for valueStr, weightStr := range rawMap {
		weightVal, err := strconv.Atoi(weightStr)
		if err != nil {
			return fmt.Errorf("failed to parse weight '%s': %w", weightStr, err)
		}

		var value T
		switch any(value).(type) {
		case string:
			value = any(valueStr).(T)
		case int:
			intVal, err := strconv.Atoi(valueStr)
			if err != nil {
				return fmt.Errorf("failed to parse int value '%s': %w", valueStr, err)
			}
			value = any(intVal).(T)
		case time.Duration:
			durVal, err := time.ParseDuration(valueStr)
			if err != nil {
				return fmt.Errorf("failed to parse duration value '%s': %w", valueStr, err)
			}
			value = any(durVal).(T)
		default:
			return fmt.Errorf("unsupported type %T for Distribution", value)
		}
		result[value] = Weight(weightVal)
	}

	*d = result
	return nil
}

// Sample returns a random value from the distribution.
func (d Distribution[T]) Sample() (T, bool) {
	var zero T
	if len(d) == 0 {
		return zero, false
	}
	if len(d) == 1 {
		for v := range d {
			return v, true
		}
	}

	totalWeight := 0
	keys := make([]T, 0, len(d))
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
	return zero, false
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
