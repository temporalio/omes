package kitchensink

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
)

const (
	// zipfS skews a zipf draw so the most common value takes a large share without starving the tail.
	zipfS = 1.1
	// defaultMaxItems is the KeywordList length bound when PoolGenerator.max_items is 0.
	defaultMaxItems = 3
)

// AttributeNamePattern matches the search attribute names a LoadInput can define.
var AttributeNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// generateValue generates a value to write for the attribute: a string for Keyword and Text, an int64 for Int, a
// float64 for Double, a bool for Bool, a time.Time for Datetime, and a []string of distinct values for KeywordList.
// def must have passed validateDefinition; callers validate once rather than on every draw.
func generateValue(def *SearchAttributeDefinition, r *rand.Rand, now time.Time) (any, error) {
	g := def.GetGenerator()
	switch def.GetType() {
	case enums.INDEXED_VALUE_TYPE_KEYWORD, enums.INDEXED_VALUE_TYPE_TEXT:
		return poolValue(def.GetName(), g.GetPool(), r), nil
	case enums.INDEXED_VALUE_TYPE_KEYWORD_LIST:
		pool := g.GetPool()
		maxItems := int(pool.GetMaxItems())
		if maxItems == 0 {
			maxItems = defaultMaxItems
		}
		n := 1 + r.IntN(maxItems)
		items := make([]string, 0, n)
		seen := map[string]bool{}
		// Bounded so a zipf pool that keeps drawing its head can't spin; the list just comes out shorter.
		for range n * 4 {
			if len(items) == n {
				break
			}
			v := poolValue(def.GetName(), pool, r)
			if !seen[v] {
				seen[v] = true
				items = append(items, v)
			}
		}
		return items, nil
	case enums.INDEXED_VALUE_TYPE_INT:
		lo, hi := g.GetIntRange().GetMin(), g.GetIntRange().GetMax()
		return lo + r.Int64N(hi-lo+1), nil
	case enums.INDEXED_VALUE_TYPE_DOUBLE:
		lo, hi := g.GetDoubleRange().GetMin(), g.GetDoubleRange().GetMax()
		return lo + r.Float64()*(hi-lo), nil
	case enums.INDEXED_VALUE_TYPE_BOOL:
		return r.Float64() < g.GetBool().GetTrueFraction(), nil
	case enums.INDEXED_VALUE_TYPE_DATETIME:
		lo, hi := int64(g.GetTimeOffset().GetMinOffset().AsDuration()), int64(g.GetTimeOffset().GetMaxOffset().AsDuration())
		return now.Add(time.Duration(lo + r.Int64N(hi-lo+1))).UTC(), nil
	default:
		return nil, fmt.Errorf("search attribute %q has unsupported type %s", def.GetName(), def.GetType())
	}
}

// QueryLiteral generates a value for the attribute and formats it as a visibility query literal. A KeywordList
// renders a single element, since a list attribute compared with = matches any of its elements.
func QueryLiteral(def *SearchAttributeDefinition, r *rand.Rand, now time.Time) (string, error) {
	if err := validateDefinition(def); err != nil {
		return "", err
	}
	v, err := generateValue(def, r, now)
	if err != nil {
		return "", err
	}
	switch v := v.(type) {
	case string:
		return quote(v), nil
	case []string:
		return quote(v[0]), nil
	case int64:
		return strconv.FormatInt(v, 10), nil
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(v), nil
	case time.Time:
		return quote(v.Format(time.RFC3339)), nil
	default:
		return "", fmt.Errorf("search attribute %q generated unexpected %T", def.GetName(), v)
	}
}

// QueryLiterals draws n distinct values from the attribute's pool and formats them as query literals joined with
// commas, for an IN list. The draw is uniform whatever the attribute's distribution, so it always ends even when n
// is close to the pool's size. n must be in [1, cardinality].
func QueryLiterals(def *SearchAttributeDefinition, r *rand.Rand, n int) (string, error) {
	if err := validateDefinition(def); err != nil {
		return "", err
	}
	pool := def.GetGenerator().GetPool()
	if pool == nil {
		return "", fmt.Errorf("search attribute %q has no value pool for an IN list", def.GetName())
	}
	card := int(pool.GetCardinality())
	if n < 1 || n > card {
		return "", fmt.Errorf("search attribute %q: IN list size must be in [1, cardinality %d], got %d",
			def.GetName(), card, n)
	}
	// A partial Fisher-Yates shuffle of [0, cardinality), tracking only the swapped slots.
	swapped := map[int]int{}
	slot := func(i int) int {
		if v, ok := swapped[i]; ok {
			return v
		}
		return i
	}
	literals := make([]string, n)
	for i := range n {
		j := i + r.IntN(card-i)
		vi, vj := slot(i), slot(j)
		swapped[i], swapped[j] = vj, vi
		literals[i] = quote(poolName(def.GetName(), uint64(vj)))
	}
	return strings.Join(literals, ", "), nil
}

// validateDefinition checks that the attribute has a valid name and a supported type, and that its generator matches
// the type and has usable parameters. Every error names the attribute.
func validateDefinition(def *SearchAttributeDefinition) error {
	name := def.GetName()
	if name == "" {
		return errors.New("search attribute has no name")
	}
	// The name has to fit a read template placeholder, and pool values built from it are quoted without escaping.
	if !AttributeNamePattern.MatchString(name) {
		return fmt.Errorf("search attribute %q: name must match %s", name, AttributeNamePattern)
	}
	g := def.GetGenerator().GetVariant()
	if g == nil {
		return fmt.Errorf("search attribute %q: generator is required", name)
	}
	mismatch := func(want string) error {
		return fmt.Errorf("search attribute %q: type %s needs a %s generator, got %T", name, def.GetType(), want, g)
	}
	switch def.GetType() {
	case enums.INDEXED_VALUE_TYPE_KEYWORD, enums.INDEXED_VALUE_TYPE_TEXT, enums.INDEXED_VALUE_TYPE_KEYWORD_LIST:
		pool, ok := g.(*ValueGenerator_Pool)
		if !ok {
			return mismatch("pool")
		}
		card := pool.Pool.GetCardinality()
		if card == 0 {
			return fmt.Errorf("search attribute %q: pool cardinality must be positive", name)
		}
		if d := pool.Pool.GetDistribution(); d != PoolDistribution_POOL_DISTRIBUTION_UNIFORM &&
			d != PoolDistribution_POOL_DISTRIBUTION_ZIPF {
			return fmt.Errorf("search attribute %q: unsupported pool distribution %d", name, int32(d))
		}
		if def.GetType() != enums.INDEXED_VALUE_TYPE_KEYWORD_LIST {
			if pool.Pool.GetMaxItems() != 0 {
				return fmt.Errorf("search attribute %q: max_items applies only to KeywordList", name)
			}
			return nil
		}
		maxItems := pool.Pool.GetMaxItems()
		if maxItems == 0 {
			maxItems = defaultMaxItems
		}
		if maxItems > card {
			return fmt.Errorf("search attribute %q: max_items %d exceeds pool cardinality %d", name, maxItems, card)
		}
	case enums.INDEXED_VALUE_TYPE_INT:
		ir, ok := g.(*ValueGenerator_IntRange)
		if !ok {
			return mismatch("int_range")
		}
		lo, hi := ir.IntRange.GetMin(), ir.IntRange.GetMax()
		if lo > hi {
			return fmt.Errorf("search attribute %q: int_range min %d exceeds max %d", name, lo, hi)
		}
		if hi-lo < 0 || hi-lo == math.MaxInt64 {
			return fmt.Errorf("search attribute %q: int_range [%d, %d] is too wide", name, lo, hi)
		}
	case enums.INDEXED_VALUE_TYPE_DOUBLE:
		dr, ok := g.(*ValueGenerator_DoubleRange)
		if !ok {
			return mismatch("double_range")
		}
		for _, v := range []float64{dr.DoubleRange.GetMin(), dr.DoubleRange.GetMax()} {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return fmt.Errorf("search attribute %q: double_range bounds must be finite, got %v", name, v)
			}
		}
		if dr.DoubleRange.GetMin() > dr.DoubleRange.GetMax() {
			return fmt.Errorf("search attribute %q: double_range min %v exceeds max %v", name,
				dr.DoubleRange.GetMin(), dr.DoubleRange.GetMax())
		}
	case enums.INDEXED_VALUE_TYPE_BOOL:
		b, ok := g.(*ValueGenerator_Bool)
		if !ok {
			return mismatch("bool")
		}
		if f := b.Bool.GetTrueFraction(); !(f >= 0 && f <= 1) {
			return fmt.Errorf("search attribute %q: true_fraction must be in [0, 1], got %v", name, f)
		}
	case enums.INDEXED_VALUE_TYPE_DATETIME:
		to, ok := g.(*ValueGenerator_TimeOffset)
		if !ok {
			return mismatch("time_offset")
		}
		lo, hi := to.TimeOffset.GetMinOffset().AsDuration(), to.TimeOffset.GetMaxOffset().AsDuration()
		if lo > hi {
			return fmt.Errorf("search attribute %q: min_offset %s exceeds max_offset %s", name, lo, hi)
		}
		if hi-lo < 0 || hi-lo == math.MaxInt64 {
			return fmt.Errorf("search attribute %q: time_offset range [%s, %s] is too wide", name, lo, hi)
		}
	default:
		return fmt.Errorf("search attribute %q has unsupported type %s", name, def.GetType())
	}
	return nil
}

// typedSearchAttributeUpdate sets v, a value from generateValue, under the SDK key for the attribute's type.
func typedSearchAttributeUpdate(def *SearchAttributeDefinition, v any) (temporal.SearchAttributeUpdate, error) {
	name := def.GetName()
	switch v := v.(type) {
	case string:
		if def.GetType() == enums.INDEXED_VALUE_TYPE_TEXT {
			return temporal.NewSearchAttributeKeyString(name).ValueSet(v), nil
		}
		return temporal.NewSearchAttributeKeyKeyword(name).ValueSet(v), nil
	case []string:
		return temporal.NewSearchAttributeKeyKeywordList(name).ValueSet(v), nil
	case int64:
		return temporal.NewSearchAttributeKeyInt64(name).ValueSet(v), nil
	case float64:
		return temporal.NewSearchAttributeKeyFloat64(name).ValueSet(v), nil
	case bool:
		return temporal.NewSearchAttributeKeyBool(name).ValueSet(v), nil
	case time.Time:
		return temporal.NewSearchAttributeKeyTime(name).ValueSet(v), nil
	default:
		return nil, fmt.Errorf("search attribute %q: no SDK key for generated %T", name, v)
	}
}

// searchAttributePayload encodes v the way the SDK encodes a typed search attribute. The kitchenSink worker hands the
// payload to the server as is, and the type metadata has the server check it against the registered type.
func searchAttributePayload(def *SearchAttributeDefinition, v any) (*common.Payload, error) {
	p, err := converter.GetDefaultDataConverter().ToPayload(v)
	if err != nil {
		return nil, fmt.Errorf("encode search attribute %q: %w", def.GetName(), err)
	}
	p.Metadata["type"] = []byte(def.GetType().String())
	return p, nil
}

// poolValue draws one of the pool's cardinality values.
func poolValue(attr string, pool *PoolGenerator, r *rand.Rand) string {
	card := uint64(pool.GetCardinality())
	var i uint64
	switch {
	case card == 1:
	case pool.GetDistribution() == PoolDistribution_POOL_DISTRIBUTION_ZIPF:
		i = rand.NewZipf(r, zipfS, 1, card-1).Uint64()
	default:
		i = r.Uint64N(card)
	}
	return poolName(attr, i)
}

// poolName is the attribute's i'th pool value, named after the attribute so values stay distinct across attributes.
func poolName(attr string, i uint64) string {
	return fmt.Sprintf("%s-%d", attr, i)
}

func quote(s string) string {
	return "'" + s + "'"
}
