package loadgen

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"slices"
	"strconv"
	"sync"
	"time"
)

// DistValueType constrains the types that can be used as distribution values.
type DistValueType interface {
	int64 | time.Duration | float32 | int32
}

type Distribution[T DistValueType] interface {
	// Sample returns a random value from the distribution using the provided random number generator.
	Sample(rng *rand.Rand) (T, bool)
	// GetType returns the distribution type identifier.
	GetType() string
	// Validate returns an error if the distribution is invalid.
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
//
//   - "mixture" - Mixture of weighted nested distributions.
//     Example: {"type": "mixture", "components": [{"weight": 3, "distribution": {"type": "normal", "mean": "500", "stdDev": "100", "min": "0", "max": "1000"}}, {"weight": 7, "distribution": {"type": "uniform", "min": "100", "max": "1000"}}]}
type DistributionField[T DistValueType] struct {
	Distribution[T]
	// DistType is NOT marshaled as the custom MarshalJSON implementation flattens the type
	// alongside the internal distribution's type
	DistType string `json:"-"`
}

// NOTE: each distribution's marshaller is responsible for setting the "type" field.
func (df DistributionField[T]) MarshalJSON() ([]byte, error) {
	if df.Distribution == nil {
		return json.Marshal(nil)
	}
	return json.Marshal(df.Distribution)
}

func (df *DistributionField[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		df.Distribution = nil
		df.DistType = ""
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
	df.DistType = typeInfo.Type

	// unmarshal the distribution data based on the type
	var err error
	switch typeInfo.Type {
	case "fixed":
		var dist FixedDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	case "discrete":
		var dist DiscreteDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	case "uniform":
		var dist UniformDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	case "zipf":
		var dist ZipfDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	case "normal":
		var dist NormalDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	case "mixture":
		var dist MixtureDistribution[T]
		err = json.Unmarshal(data, &dist)
		df.Distribution = &dist
	default:
		return fmt.Errorf("unknown distribution type: %s", typeInfo.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to unmarshal %s distribution: %w", typeInfo.Type, err)
	}

	if err := df.Distribution.Validate(); err != nil {
		return fmt.Errorf("invalid %s distribution: %w", typeInfo.Type, err)
	}

	return nil
}

type FixedDistribution[T DistValueType] struct {
	Value T `json:"value"`
}

func NewFixedDistribution[T DistValueType](value T) DistributionField[T] {
	return DistributionField[T]{
		Distribution: &FixedDistribution[T]{Value: value},
		DistType:     "fixed",
	}
}

func (d *FixedDistribution[T]) GetType() string {
	return "fixed"
}

func (d *FixedDistribution[T]) Validate() error {
	// Fixed distributions are always valid
	return nil
}

func (d *FixedDistribution[T]) Sample(_ *rand.Rand) (T, bool) {
	return d.Value, true
}

func (d *FixedDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":  d.GetType(),
		"value": renderToJSON(d.Value),
	})
}

func (d *FixedDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	value, err := parseFromJSON[T](rawMap, "value")
	if err != nil {
		return err
	}

	d.Value = value
	return nil
}

type DiscreteDistribution[T DistValueType] struct {
	Weights   map[T]int         `json:"weights"`
	cacheOnce sync.Once         `json:"-"`
	cache     *discreteCache[T] `json:"-"`
}

// discreteCache holds pre-computed values for DiscreteDistribution sampling
type discreteCache[T DistValueType] struct {
	keys        []T
	weights     []int
	totalWeight int
}

func NewDiscreteDistribution[T DistValueType](weights map[T]int) DistributionField[T] {
	return DistributionField[T]{
		Distribution: &DiscreteDistribution[T]{Weights: weights},
		DistType:     "discrete",
	}
}

func (d *DiscreteDistribution[T]) GetType() string {
	return "discrete"
}

func (d *DiscreteDistribution[T]) Validate() error {
	if len(d.Weights) == 0 {
		return fmt.Errorf("weights map cannot be empty")
	}
	for value, weight := range d.Weights {
		if weight <= 0 {
			return fmt.Errorf("weight value must be positive, got %d for value %v", weight, value)
		}
	}
	return nil
}

func (d *DiscreteDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	var zero T
	if len(d.Weights) == 0 {
		return zero, false
	}
	if len(d.Weights) == 1 {
		for v := range d.Weights {
			return v, true
		}
	}

	// Build the sampling cache exactly once; Sample is called concurrently from
	// multiple iteration goroutines that share a distribution instance.
	d.cacheOnce.Do(func() {
		keys := make([]T, 0, len(d.Weights))
		for k := range d.Weights {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		totalWeight := 0
		weights := make([]int, len(keys))
		for i, k := range keys {
			weights[i] = d.Weights[k]
			totalWeight += weights[i]
		}

		d.cache = &discreteCache[T]{
			keys:        keys,
			weights:     weights,
			totalWeight: totalWeight,
		}
	})

	// Perform weighted sampling
	r := rng.Intn(d.cache.totalWeight)
	cumulativeWeight := 0
	for i, w := range d.cache.weights {
		cumulativeWeight += w
		if r < cumulativeWeight {
			return d.cache.keys[i], true
		}
	}
	return zero, false
}

func (d *DiscreteDistribution[T]) MarshalJSON() ([]byte, error) {
	weights := make(map[string]int, len(d.Weights))
	for value, weight := range d.Weights {
		weights[renderToJSON(value)] = weight
	}

	return json.Marshal(map[string]any{
		"type":    d.GetType(),
		"weights": weights,
	})
}

func (d *DiscreteDistribution[T]) UnmarshalJSON(data []byte) error {
	var rawMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &rawMap); err != nil {
		return err
	}

	var weightsMap map[string]int
	if err := json.Unmarshal(rawMap["weights"], &weightsMap); err != nil {
		return fmt.Errorf("failed to parse 'weights' as map[string]int: %w", err)
	}

	d.Weights = make(map[T]int, len(weightsMap))
	for valueStr, weight := range weightsMap {
		value, err := stringToValue[T](valueStr)
		if err != nil {
			return err
		}
		d.Weights[value] = weight
	}

	return nil
}

type UniformDistribution[T DistValueType] struct {
	Min T `json:"min"`
	Max T `json:"max"`
}

func (d *UniformDistribution[T]) GetType() string {
	return "uniform"
}

func (d *UniformDistribution[T]) Validate() error {
	if d.Max < d.Min {
		return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.Max, d.Min)
	}
	return nil
}

func (d *UniformDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	switch any(d.Min).(type) {
	case int64:
		minVal, maxVal := any(d.Min).(int64), any(d.Max).(int64)
		if maxVal <= minVal {
			return d.Min, true
		}
		result := minVal + rng.Int63n(maxVal-minVal+1)
		return T(result), true
	case int32:
		minVal, maxVal := any(d.Min).(int32), any(d.Max).(int32)
		if maxVal <= minVal {
			return d.Min, true
		}
		result := minVal + rng.Int31n(maxVal-minVal+1)
		return T(result), true
	case float32:
		minVal, maxVal := any(d.Min).(float32), any(d.Max).(float32)
		if maxVal <= minVal {
			return d.Min, true
		}
		result := minVal + rng.Float32()*(maxVal-minVal)
		return T(result), true
	case time.Duration:
		minVal, maxVal := any(d.Min).(time.Duration), any(d.Max).(time.Duration)
		if maxVal <= minVal {
			return d.Min, true
		}
		result := time.Duration(int64(minVal) + rng.Int63n(int64(maxVal)-int64(minVal)+1))
		return T(result), true
	default:
		return d.Min, true
	}
}

func (d *UniformDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": d.GetType(),
		"min":  renderToJSON(d.Min),
		"max":  renderToJSON(d.Max),
	})
}

func (d *UniformDistribution[T]) UnmarshalJSON(data []byte) error {
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

	d.Min = min
	d.Max = max

	return nil
}

type ZipfDistribution[T DistValueType] struct {
	S float64 `json:"s"`
	V float64 `json:"v"`
	N uint64  `json:"n"`
}

func (d *ZipfDistribution[T]) GetType() string {
	return "zipf"
}

func (d *ZipfDistribution[T]) Validate() error {
	if d.S <= 1.0 {
		return fmt.Errorf("zipf distribution requires s > 1.0, got %f", d.S)
	}
	if d.V < 1.0 {
		return fmt.Errorf("zipf distribution requires v ≥ 1.0, got %f", d.V)
	}
	if d.N < 1 {
		return fmt.Errorf("zipf distribution requires n ≥ 1, got %d", d.N)
	}
	return nil
}

func (d *ZipfDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	zipf := rand.NewZipf(rng, d.S, d.V, d.N)
	return T(zipf.Uint64()), true
}

func (d *ZipfDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type": d.GetType(),
		"s":    d.S,
		"v":    d.V,
		"n":    d.N,
	})
}

func (d *ZipfDistribution[T]) UnmarshalJSON(data []byte) error {
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

	d.S = s
	d.V = v
	d.N = n

	return nil
}

type NormalDistribution[T DistValueType] struct {
	Mean   T `json:"mean"`
	StdDev T `json:"stdDev"`
	Min    T `json:"min"`
	Max    T `json:"max"`
}

func (d *NormalDistribution[T]) GetType() string {
	return "normal"
}

func (d *NormalDistribution[T]) Validate() error {
	if d.StdDev <= 0 {
		return fmt.Errorf("standard deviation must be positive, got %v", d.StdDev)
	}
	if d.Max < d.Min {
		return fmt.Errorf("max value (%v) cannot be less than min value (%v)", d.Max, d.Min)
	}
	return nil
}

func (d *NormalDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	result := min(d.Max, max(d.Min, d.Mean+T(float64(d.StdDev)*rng.NormFloat64())))
	return result, true
}

func (d *NormalDistribution[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{
		"type":   d.GetType(),
		"mean":   renderToJSON(d.Mean),
		"stdDev": renderToJSON(d.StdDev),
		"min":    renderToJSON(d.Min),
		"max":    renderToJSON(d.Max),
	})
}

func (d *NormalDistribution[T]) UnmarshalJSON(data []byte) error {
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

	d.Mean = mean
	d.StdDev = stdDev
	d.Min = minVal
	d.Max = maxVal

	return nil
}

type MixtureComponent[T DistValueType] struct {
	Weight       int                  `json:"weight"`
	Distribution DistributionField[T] `json:"distribution"`
}

type MixtureDistribution[T DistValueType] struct {
	Components []MixtureComponent[T] `json:"components"`

	// totalWeight is the sum of component weights, cached on first Sample.
	// Sample is called concurrently from multiple iteration goroutines that
	// share a distribution instance, so it is computed exactly once.
	totalWeightOnce sync.Once `json:"-"`
	totalWeight     int       `json:"-"`
}

func (d *MixtureDistribution[T]) GetType() string {
	return "mixture"
}

func (d *MixtureDistribution[T]) Validate() error {
	if len(d.Components) == 0 {
		return fmt.Errorf("components cannot be empty")
	}
	for i, component := range d.Components {
		if component.Weight <= 0 {
			return fmt.Errorf("component %d weight must be positive, got %d", i, component.Weight)
		}
		if component.Distribution.Distribution == nil {
			return fmt.Errorf("component %d is missing a distribution", i)
		}
		if err := component.Distribution.Validate(); err != nil {
			return fmt.Errorf("invalid component %d distribution: %w", i, err)
		}
	}
	return nil
}

func (d *MixtureDistribution[T]) Sample(rng *rand.Rand) (T, bool) {
	var zero T
	if len(d.Components) == 0 {
		return zero, false
	}

	d.totalWeightOnce.Do(func() {
		for _, component := range d.Components {
			d.totalWeight += component.Weight
		}
	})
	if d.totalWeight <= 0 {
		return zero, false
	}

	// Perform weighted sampling to pick a component, then sample from it.
	r := rng.Intn(d.totalWeight)
	cumulativeWeight := 0
	for _, component := range d.Components {
		cumulativeWeight += component.Weight
		if r < cumulativeWeight {
			return component.Distribution.Sample(rng)
		}
	}
	return zero, false
}

func (d *MixtureDistribution[T]) MarshalJSON() ([]byte, error) {
	components := make([]map[string]any, len(d.Components))
	for i, component := range d.Components {
		components[i] = map[string]any{
			"weight":       component.Weight,
			"distribution": component.Distribution,
		}
	}

	return json.Marshal(map[string]any{
		"type":       d.GetType(),
		"components": components,
	})
}

func (d *MixtureDistribution[T]) UnmarshalJSON(data []byte) error {
	var raw struct {
		Components []struct {
			Weight       int                  `json:"weight"`
			Distribution DistributionField[T] `json:"distribution"`
		} `json:"components"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	d.Components = make([]MixtureComponent[T], len(raw.Components))
	for i, component := range raw.Components {
		d.Components[i] = MixtureComponent[T]{
			Weight:       component.Weight,
			Distribution: component.Distribution,
		}
	}

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
	case int32:
		var numValue int32
		if err := json.Unmarshal(rawMsg, &numValue); err == nil {
			return any(numValue).(T), nil
		}

		var strValue string
		if err := json.Unmarshal(rawMsg, &strValue); err != nil {
			return zero, fmt.Errorf("failed to parse '%s' as int32 or string: %w", fieldName, err)
		}

		intVal, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("failed to parse int32 value '%s': %w", strValue, err)
		}
		return any(int32(intVal)).(T), nil

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

func renderToJSON[T DistValueType](value T) string {
	switch v := any(value).(type) {
	case time.Duration:
		return v.String()
	default:
		return fmt.Sprintf("%v", value)
	}
}

func stringToValue[T DistValueType](valueStr string) (T, error) {
	var zero T
	switch any(zero).(type) {
	case int64:
		intVal, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return zero, fmt.Errorf("failed to parse int64 value '%s': %w", valueStr, err)
		}
		return any(intVal).(T), nil
	case int32:
		intVal, err := strconv.ParseInt(valueStr, 10, 32)
		if err != nil {
			return zero, fmt.Errorf("failed to parse int32 value '%s': %w", valueStr, err)
		}
		return any(int32(intVal)).(T), nil
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
