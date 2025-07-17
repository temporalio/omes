package loadgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"
)

// distValueType constrains the types that can be used as distribution values.
type distValueType interface {
	int64 | time.Duration | float32
}

type distribution[T distValueType] interface {
	// Sample returns a random value from the distribution using the provided random number generator.
	Sample(rng *rand.Rand) (T, bool)
	// GetType returns the distribution type identifier.
	GetType() string
	// Validate checks if the distribution is valid.
	Validate() error
}

// DistributionField is a generic wrapper for any distribution that can be serialized to/from JSON.
//
// Supported distribution types:
//
//   - "fixed" - Fixed distribution that always returns the same value.
//     Example: {"type": "fixed", "value": "500"}
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
	case "fixed":
		var dist fixedDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.distribution = dist
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

type fixedDistribution[T distValueType] struct {
	value T
}

func NewFixedDistribution[T distValueType](value T) DistributionField[T] {
	return DistributionField[T]{
		distribution: fixedDistribution[T]{value: value},
		distType:     "fixed",
	}
}

func (d fixedDistribution[T]) GetType() string {
	return "fixed"
}

func (d fixedDistribution[T]) Validate() error {
	// Fixed distributions are always valid
	return nil
}

func (d fixedDistribution[T]) Sample(_ *rand.Rand) (T, bool) {
	return d.value, true
}

func (d fixedDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":  d.GetType(),
		"value": renderToJSON(d.value),
	})
}

func (d *fixedDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	value, err := parseFromJSON[T](rawMap, "value")
	if err != nil {
		return err
	}

	d.value = value
	return nil
}

type discreteDistribution[T distValueType] struct {
	weights map[T]int
}

func NewDiscreteDistribution[T distValueType](weights map[T]int) DistributionField[T] {
	return DistributionField[T]{
		distribution: discreteDistribution[T]{weights: weights},
		distType:     "discrete",
	}
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

func (d discreteDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	var zero T
	if len(d.weights) == 0 {
		return zero, false
	}
	if len(d.weights) == 1 {
		for v := range d.weights {
			return v, true
		}
	}

	keys := make([]T, 0, len(d.weights))
	for k := range d.weights {
		keys = append(keys, k)
	}
	switch any(keys[0]).(type) {
	case int64:
		sort.Slice(keys, func(i, j int) bool {
			return any(keys[i]).(int64) < any(keys[j]).(int64)
		})
	case float32:
		sort.Slice(keys, func(i, j int) bool {
			return any(keys[i]).(float32) < any(keys[j]).(float32)
		})
	case time.Duration:
		sort.Slice(keys, func(i, j int) bool {
			return any(keys[i]).(time.Duration) < any(keys[j]).(time.Duration)
		})
	}

	totalWeight := 0
	weights := make([]int, len(keys))
	for i, k := range keys {
		weights[i] = d.weights[k]
		totalWeight += weights[i]
	}

	r := rng.Intn(totalWeight)
	cumulativeWeight := 0
	for i, w := range weights {
		cumulativeWeight += w
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
	switch any(d.min).(type) {
	case int64:
		if any(d.max).(int64) < any(d.min).(int64) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	case float32:
		if any(d.max).(float32) < any(d.min).(float32) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	case time.Duration:
		if any(d.max).(time.Duration) < any(d.min).(time.Duration) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	}
	return nil
}

func (d uniformDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	switch any(d.min).(type) {
	case int64:
		minVal, maxVal := any(d.min).(int64), any(d.max).(int64)
		if maxVal <= minVal {
			return d.min, true
		}
		result := minVal + rng.Int63n(maxVal-minVal+1)
		return T(result), true
	case float32:
		minVal, maxVal := any(d.min).(float32), any(d.max).(float32)
		if maxVal <= minVal {
			return d.min, true
		}
		result := minVal + rng.Float32()*(maxVal-minVal)
		return any(result).(T), true
	case time.Duration:
		minVal, maxVal := any(d.min).(time.Duration), any(d.max).(time.Duration)
		if maxVal <= minVal {
			return d.min, true
		}
		result := time.Duration(int64(minVal) + rng.Int63n(int64(maxVal)-int64(minVal)+1))
		return T(result), true
	default:
		return d.min, true
	}
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

func (d zipfDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	zipf := rand.NewZipf(rng, d.s, d.v, d.n)
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
	switch any(d.stdDev).(type) {
	case int64:
		if any(d.stdDev).(int64) <= 0 {
			return fmt.Errorf("standard deviation must be positive, got %v", d.stdDev)
		}
		if any(d.max).(int64) < any(d.min).(int64) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	case float32:
		if any(d.stdDev).(float32) <= 0 {
			return fmt.Errorf("standard deviation must be positive, got %v", d.stdDev)
		}
		if any(d.max).(float32) < any(d.min).(float32) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	case time.Duration:
		if any(d.stdDev).(time.Duration) <= 0 {
			return fmt.Errorf("standard deviation must be positive, got %v", d.stdDev)
		}
		if any(d.max).(time.Duration) < any(d.min).(time.Duration) {
			return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.max, d.min)
		}
	}
	return nil
}

func (d normalDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	switch any(d.mean).(type) {
	case int64:
		mean, stdDev := any(d.mean).(int64), any(d.stdDev).(int64)
		minVal, maxVal := any(d.min).(int64), any(d.max).(int64)

		result := mean + int64(float64(stdDev)*rng.NormFloat64())
		if result < minVal {
			result = minVal
		}
		if result > maxVal {
			result = maxVal
		}
		return T(result), true
	case float32:
		mean, stdDev := any(d.mean).(float32), any(d.stdDev).(float32)
		minVal, maxVal := any(d.min).(float32), any(d.max).(float32)

		result := mean + float32(float64(stdDev)*rng.NormFloat64())
		if result < minVal {
			result = minVal
		}
		if result > maxVal {
			result = maxVal
		}
		return any(result).(T), true
	case time.Duration:
		mean, stdDev := any(d.mean).(time.Duration), any(d.stdDev).(time.Duration)
		minVal, maxVal := any(d.min).(time.Duration), any(d.max).(time.Duration)

		result := time.Duration(int64(mean) + int64(float64(stdDev)*rng.NormFloat64()))
		if result < minVal {
			result = minVal
		}
		if result > maxVal {
			result = maxVal
		}
		return T(result), true
	default:
		return d.mean, false
	}
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

	case float32:
		var numValue float32
		if err := json.Unmarshal(rawMsg, &numValue); err == nil {
			return any(numValue).(T), nil
		}

		var strValue string
		if err := json.Unmarshal(rawMsg, &strValue); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as float32 or string: %w", fieldName, err)
		}

		floatVal, err := strconv.ParseFloat(strValue, 32)
		if err != nil {
			return zero, fmt.Errorf("failed to parse float32 value '%s': %w", strValue, err)
		}
		return any(float32(floatVal)).(T), nil

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
	case float32:
		floatVal, err := strconv.ParseFloat(valueStr, 32)
		if err != nil {
			return zero, fmt.Errorf("failed to parse float32 value '%s': %w", valueStr, err)
		}
		return any(float32(floatVal)).(T), nil
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
