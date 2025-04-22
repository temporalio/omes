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

// distValueType constrains the types that can be used as distribution values.
type distValueType interface{ int64 | time.Duration }

type distribution[T distValueType] interface {
	// Sample returns a random value from the distribution.
	Sample() (T, bool)
	// GetType returns the distribution type identifier.
	GetType() string
	// Validate checks if the distribution is valid.
	Validate() error
}

// DistributionField is a generic wrapper for any distribution that can be serialized to/from JSON.
//
// Supported distribution types:
//
//   - "discrete" - Discrete distribution with weighted values.
//     Example: {"type": "discrete", "weights": {"100": 1, "200": 2, "300": 3}}
//
//   - "uniform" - Uniform distribution.
//     Example: {"type": "uniform", "min": "100", "max": "1000"}
//
//   - "zipf" - Zipf distribution.
//     Example: {"type": "zipf", "s": 2.0, "v": 1.0, "n": 100}
//
//   - "normal" - Normal distribution.
//     Example: {"type": "normal", "mean": "500", "stdDev": "100", "min": "0", "max": "1000"}
type DistributionField[T distValueType] struct {
	distribution[T]
	distType string
}

// NOTE: each distribution's marshaller is responsible for setting the "type" field.
func (df DistributionField[T]) MarshalJSON() ([]byte, error) {
	if df.distribution == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(df.distribution)
}

func (df *DistributionField[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		df.distribution = nil
		df.distType = ""
		return nil
	}

	// parse the type first
	var typeInfo struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &typeInfo); err != nil {
		return fmt.Errorf("failed to determine distribution type: %w", err)
	}
	if typeInfo.Type == "" {
		return fmt.Errorf("missing distribution type")
	}
	df.distType = typeInfo.Type

	// unmarshal the distribution data based on the type
	var err error
	switch typeInfo.Type {
	case "discrete":
		var dist discreteDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.distribution = dist
	case "uniform":
		var dist uniformDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.distribution = dist
	case "zipf":
		var dist zipfDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.distribution = dist
	case "normal":
		var dist normalDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.distribution = dist
	default:
		return fmt.Errorf("unknown distribution type: %s", typeInfo.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal %s distribution: %w", typeInfo.Type, err)
	}

	if err := df.distribution.Validate(); err != nil {
		return fmt.Errorf("invalid %s distribution: %w", typeInfo.Type, err)
	}

	return nil
}

type discreteDistribution[T distValueType] struct {
	weights map[T]int
}

func (d discreteDistribution[T]) GetType() string {
	return "discrete"
}

func (d discreteDistribution[T]) Validate() error {
	if len(d.weights) == 0 {
		return fmt.Errorf("weights map cannot be empty")
	}
	for value, weight := range d.weights {
		if weight <= 0 {
			return fmt.Errorf("weight value must be positive, got %d for value %v", weight, value)
		}
	}
	return nil
}

func (d discreteDistribution[T]) Sample() (T, bool) {
	var zero T
	if len(d.weights) == 0 {
		return zero, false
	}
	if len(d.weights) == 1 {
		for v := range d.weights {
			return v, true
		}
	}

	totalWeight := 0
	keys := make([]T, 0, len(d.weights))
	weights := make([]int, 0, len(d.weights))
	for k, w := range d.weights {
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

func (d discreteDistribution[T]) MarshalJSON() ([]byte, error) {
	weights := make(map[string]int, len(d.weights))
	for value, weight := range d.weights {
		weights[renderToJSON(value)] = weight
	}

	return json.Marshal(map[string]interface{}{
		"type":    d.GetType(),
		"weights": weights,
	})
}

func (d *discreteDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	var weightsMap map[string]int
	if err := json.Unmarshal(rawMap["weights"], &weightsMap); err != nil {
		return fmt.Errorf("failed to parse 'weights' as map[string]int: %w", err)
	}

	d.weights = make(map[T]int, len(weightsMap))
	for valueStr, weight := range weightsMap {
		value, err := stringToValue[T](valueStr)
		if err != nil {
			return err
		}
		d.weights[value] = weight
	}

	return nil
}

type uniformDistribution[T distValueType] struct {
	min T
	max T
}

func (d uniformDistribution[T]) GetType() string {
	return "uniform"
}

func (d uniformDistribution[T]) Validate() error {
	if int64(d.max) < int64(d.min) {
		return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
	}
	return nil
}

func (d uniformDistribution[T]) Sample() (T, bool) {
	minVal, maxVal := int64(d.min), int64(d.max)
	if maxVal <= minVal {
		return d.min, true
	}
	result := minVal + rand.Int63n(maxVal-minVal+1)
	return T(result), true
}

func (d uniformDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type": d.GetType(),
		"min":  renderToJSON(d.min),
		"max":  renderToJSON(d.max),
	})
}

func (d *uniformDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	min, err := parseFromJSON[T](rawMap, "min")
	if err != nil {
		return err
	}

	max, err := parseFromJSON[T](rawMap, "max")
	if err != nil {
		return err
	}

	d.min = min
	d.max = max

	return nil
}

type zipfDistribution[T distValueType] struct {
	s float64
	v float64
	n uint64
}

func (d zipfDistribution[T]) GetType() string {
	return "zipf"
}

func (d zipfDistribution[T]) Validate() error {
	if d.s <= 1.0 {
		return fmt.Errorf("zipf distribution requires s > 1.0, got %f", d.s)
	}
	if d.v < 1.0 {
		return fmt.Errorf("zipf distribution requires v ≥ 1.0, got %f", d.v)
	}
	if d.n < 1 {
		return fmt.Errorf("zipf distribution requires n ≥ 1, got %d", d.n)
	}
	return nil
}

func (d zipfDistribution[T]) Sample() (T, bool) {
	zipf := rand.NewZipf(rand.New(rand.NewSource(time.Now().UnixNano())), d.s, d.v, d.n)
	return T(zipf.Uint64()), true
}

func (d zipfDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type": d.GetType(),
		"s":    d.s,
		"v":    d.v,
		"n":    d.n,
	})
}

func (d *zipfDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	s, err := parseFromJSON[float64](rawMap, "s")
	if err != nil {
		return err
	}

	v, err := parseFromJSON[float64](rawMap, "v")
	if err != nil {
		return err
	}

	n, err := parseFromJSON[uint64](rawMap, "n")
	if err != nil {
		return err
	}

	d.s = s
	d.v = v
	d.n = n

	return nil
}

type normalDistribution[T distValueType] struct {
	mean   T
	stdDev T
	min    T
	max    T
}

func (d normalDistribution[T]) GetType() string {
	return "normal"
}

func (d normalDistribution[T]) Validate() error {
	if int64(d.stdDev) <= 0 {
		return fmt.Errorf("standard deviation must be positive, got %v", d.stdDev)
	}

	if int64(d.max) < int64(d.min) {
		return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
	}

	return nil
}

func (d normalDistribution[T]) Sample() (T, bool) {
	mean, stdDev := int64(d.mean), int64(d.stdDev)
	minVal, maxVal := int64(d.min), int64(d.max)

	result := mean + int64(float64(stdDev)*rand.NormFloat64())
	if result < minVal {
		result = minVal
	}
	if result > maxVal {
		result = maxVal
	}
	return T(result), true
}

func (d normalDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   d.GetType(),
		"mean":   renderToJSON(d.mean),
		"stdDev": renderToJSON(d.stdDev),
		"min":    renderToJSON(d.min),
		"max":    renderToJSON(d.max),
	})
}

func (d *normalDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	mean, err := parseFromJSON[T](rawMap, "mean")
	if err != nil {
		return err
	}

	stdDev, err := parseFromJSON[T](rawMap, "stdDev")
	if err != nil {
		return err
	}

	minVal, err := parseFromJSON[T](rawMap, "min")
	if err != nil {
		return err
	}

	maxVal, err := parseFromJSON[T](rawMap, "max")
	if err != nil {
		return err
	}

	d.mean = mean
	d.stdDev = stdDev
	d.min = minVal
	d.max = maxVal

	return nil
}

func parseFromJSON[T any](rawMap map[string]json.RawMessage, fieldName string) (T, error) {
	var zero T
	rawMsg, ok := rawMap[fieldName]
	if !ok {
		return zero, fmt.Errorf("missing required field '%s'", fieldName)
	}
	if len(rawMsg) == 0 {
		return zero, fmt.Errorf("field '%s' cannot be empty", fieldName)
	}

	switch any(zero).(type) {
	case int64:
		var numValue int64
		if err := json.Unmarshal(rawMsg, &numValue); err == nil {
			return any(numValue).(T), nil
		}

		var strValue string
		if err := json.Unmarshal(rawMsg, &strValue); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as int64 or string: %w", fieldName, err)
		}

		intVal, err := strconv.ParseInt(strValue, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse int64 value '%s': %w", strValue, err)
		}
		return any(intVal).(T), nil

	case time.Duration:
		var strValue string
		if err := json.Unmarshal(rawMsg, &strValue); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as duration or number: %w", fieldName, err)
		}

		durVal, err := time.ParseDuration(strValue)
		if err != nil {
			return zero, fmt.Errorf("failed to parse duration value '%s': %w", strValue, err)
		}
		return any(durVal).(T), nil

	case float64:
		var value float64
		if err := json.Unmarshal(rawMsg, &value); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as float64: %w", fieldName, err)
		}
		return any(value).(T), nil

	case uint64:
		var value uint64
		if err := json.Unmarshal(rawMsg, &value); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as uint64: %w", fieldName, err)
		}
		return any(value).(T), nil

	default:
		return zero, fmt.Errorf("unsupported type %T for field '%s'", zero, fieldName)
	}
}

func renderToJSON[T distValueType](value T) string {
	switch v := any(value).(type) {
	case time.Duration:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func stringToValue[T distValueType](valueStr string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case int64:
		intVal, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse int64 value '%s': %w", valueStr, err)
		}
		return T(intVal), nil
	case time.Duration:
		durVal, err := time.ParseDuration(valueStr)
		if err != nil {
			return zero, fmt.Errorf("failed to parse duration value '%s': %w", valueStr, err)
		}
		return T(durVal), nil
	default:
		return zero, fmt.Errorf("unsupported type %T for Distribution", zero)
	}
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
