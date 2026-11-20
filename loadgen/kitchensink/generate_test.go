package kitchensink

import (
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/types/known/durationpb"
)

const draws = 5000

// TestGenerateValueIsStable checks that a fixed seed yields a fixed value for each type.
func TestGenerateValueIsStable(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		def  *SearchAttributeDefinition
		want any
	}{
		{"keyword zipf", keywordDef("VisTenant", 50, PoolDistribution_POOL_DISTRIBUTION_ZIPF), "VisTenant-1"},
		{"keyword uniform", keywordDef("VisTenant", 50, PoolDistribution_POOL_DISTRIBUTION_UNIFORM), "VisTenant-38"},
		{"keyword list", &SearchAttributeDefinition{Name: "VisTags", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST,
			Generator: poolGen(20, PoolDistribution_POOL_DISTRIBUTION_ZIPF, 3)},
			[]string{"VisTags-2", "VisTags-3", "VisTags-0"}},
		{"int", intDef("VisPriority", 1, 100), int64(77)},
		{"double", doubleDef("VisScore", 0.5, 2.5), 1.8529113193356501},
		{"bool", boolDef("VisPinned", 0.5), false},
		{"datetime", datetimeDef("VisDue", -2*time.Hour, time.Hour),
			time.Date(2026, time.May, 1, 12, 18, 29, 231309143, time.UTC)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := generateValue(tt.def, newRand(), now)
			require.NoError(t, err)
			require.Equal(t, tt.want, v)
		})
	}
}

// TestKeywordValuesStayInPool checks that Keyword values come only from the attribute's pool, and that a uniform pool
// small enough to exhaust is drawn in full.
func TestKeywordValuesStayInPool(t *testing.T) {
	for _, dist := range []PoolDistribution{PoolDistribution_POOL_DISTRIBUTION_UNIFORM,
		PoolDistribution_POOL_DISTRIBUTION_ZIPF} {
		t.Run(dist.String(), func(t *testing.T) {
			def := keywordDef("K", 7, dist)
			pool := poolOf("K", 7)
			seen := map[string]int{}
			r := newRand()
			for range draws {
				v, err := generateValue(def, r, time.Time{})
				require.NoError(t, err)
				require.Contains(t, pool, v)
				seen[v.(string)]++
			}
			if dist == PoolDistribution_POOL_DISTRIBUTION_UNIFORM {
				require.Len(t, seen, 7)
			} else {
				require.Greater(t, seen["K-0"], seen["K-6"], "zipf should favor the head of the pool")
			}
		})
	}
}

// TestKeywordCardinalityOne checks that a pool of one always yields its single value.
func TestKeywordCardinalityOne(t *testing.T) {
	def := &SearchAttributeDefinition{Name: "K", Type: enums.INDEXED_VALUE_TYPE_TEXT,
		Generator: poolGen(1, PoolDistribution_POOL_DISTRIBUTION_ZIPF, 0)}
	r := newRand()
	for range 100 {
		v, err := generateValue(def, r, time.Time{})
		require.NoError(t, err)
		require.Equal(t, "K-0", v)
	}
}

// TestKeywordListValues checks that a KeywordList holds 1 to max_items distinct values from its pool.
func TestKeywordListValues(t *testing.T) {
	def := &SearchAttributeDefinition{Name: "L", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST,
		Generator: poolGen(4, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 3)}
	pool := poolOf("L", 4)
	lengths := map[int]bool{}
	r := newRand()
	for range draws {
		v, err := generateValue(def, r, time.Time{})
		require.NoError(t, err)
		items := v.([]string)
		require.NotEmpty(t, items)
		require.LessOrEqual(t, len(items), 3)
		lengths[len(items)] = true
		seen := map[string]bool{}
		for _, item := range items {
			require.Contains(t, pool, item)
			require.False(t, seen[item], "duplicate item %q in %v", item, items)
			seen[item] = true
		}
	}
	require.Len(t, lengths, 3)
}

// TestKeywordListDefaultsToThreeItems checks that max_items 0 allows up to three items.
func TestKeywordListDefaultsToThreeItems(t *testing.T) {
	def := &SearchAttributeDefinition{Name: "L", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST,
		Generator: poolGen(10, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 0)}
	longest := 0
	r := newRand()
	for range draws {
		v, err := generateValue(def, r, time.Time{})
		require.NoError(t, err)
		longest = max(longest, len(v.([]string)))
	}
	require.Equal(t, 3, longest)
}

// TestNumericValuesStayInRange checks that Int and Double values fall within [min, max] and Int reaches both ends.
func TestNumericValuesStayInRange(t *testing.T) {
	r := newRand()
	ints := intDef("I", -2, 3)
	seen := map[int64]bool{}
	for range draws {
		v, err := generateValue(ints, r, time.Time{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, v.(int64), int64(-2))
		require.LessOrEqual(t, v.(int64), int64(3))
		seen[v.(int64)] = true
	}
	require.Len(t, seen, 6)

	doubles := doubleDef("D", -2, 3)
	for range draws {
		v, err := generateValue(doubles, r, time.Time{})
		require.NoError(t, err)
		require.GreaterOrEqual(t, v.(float64), -2.0)
		require.LessOrEqual(t, v.(float64), 3.0)
	}
}

// TestBoolTrueFraction checks that true_fraction 0 and 1 pin the value and 0.1 yields mostly false.
func TestBoolTrueFraction(t *testing.T) {
	r := newRand()
	count := func(frac float64) int {
		def := boolDef("B", frac)
		n := 0
		for range draws {
			v, err := generateValue(def, r, time.Time{})
			require.NoError(t, err)
			if v.(bool) {
				n++
			}
		}
		return n
	}
	require.Zero(t, count(0))
	require.Equal(t, draws, count(1))
	require.InDelta(t, draws/10, count(0.1), draws/50)
}

// TestDatetimeValuesStayInOffsetRange checks that Datetime values fall within now plus [min_offset, max_offset] in UTC.
func TestDatetimeValuesStayInOffsetRange(t *testing.T) {
	def := datetimeDef("T", -2*time.Hour, time.Hour)
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("x", 3600))
	r := newRand()
	for range draws {
		v, err := generateValue(def, r, now)
		require.NoError(t, err)
		got := v.(time.Time)
		require.False(t, got.Before(now.Add(-2*time.Hour)), got)
		require.False(t, got.After(now.Add(time.Hour)), got)
		require.Equal(t, time.UTC, got.Location())
	}
}

// TestQueryLiteral checks how each type formats as a visibility query literal.
func TestQueryLiteral(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.FixedZone("x", 3600))
	tests := []struct {
		def  *SearchAttributeDefinition
		want string
	}{
		{keywordDef("K", 1, PoolDistribution_POOL_DISTRIBUTION_UNIFORM), "'K-0'"},
		{&SearchAttributeDefinition{Name: "L", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST,
			Generator: poolGen(1, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 1)}, "'L-0'"},
		{intDef("I", 7, 7), "7"},
		{doubleDef("D", 0.5, 0.5), "0.5"},
		{boolDef("B", 1), "true"},
		{datetimeDef("T", 0, 0), "'2026-05-01T11:00:00Z'"},
	}
	for _, tt := range tests {
		t.Run(tt.def.Type.String(), func(t *testing.T) {
			got, err := QueryLiteral(tt.def, newRand(), now)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestQueryLiterals checks that an IN list holds n distinct quoted values from the attribute's pool, even when n is the
// whole pool of a zipf attribute.
func TestQueryLiterals(t *testing.T) {
	def := keywordDef("K", 8, PoolDistribution_POOL_DISTRIBUTION_ZIPF)
	r := newRand()
	for _, n := range []int{1, 3, 8} {
		lits, err := QueryLiterals(def, r, n)
		require.NoError(t, err)
		got := strings.Split(lits, ", ")
		require.Len(t, got, n)
		var values []string
		for _, lit := range got {
			v, ok := strings.CutPrefix(lit, "'")
			require.True(t, ok, lit)
			v, ok = strings.CutSuffix(v, "'")
			require.True(t, ok, lit)
			values = append(values, v)
		}
		require.Subset(t, poolOf("K", 8), values)
		slices.Sort(values)
		require.Len(t, slices.Compact(values), n, "values repeat: %v", got)
	}
}

// TestQueryLiteralsRejectsBadCount checks that n outside [1, cardinality] is an error.
func TestQueryLiteralsRejectsBadCount(t *testing.T) {
	def := keywordDef("K", 3, PoolDistribution_POOL_DISTRIBUTION_UNIFORM)
	for _, n := range []int{0, 4} {
		_, err := QueryLiterals(def, newRand(), n)
		require.ErrorContains(t, err, "K")
	}
}

// TestQueryLiteralRejectsInvalidDefinition checks that QueryLiteral validates its definition before drawing, naming
// the attribute.
func TestQueryLiteralRejectsInvalidDefinition(t *testing.T) {
	_, err := QueryLiteral(intDef("Bad", 5, 4), newRand(), time.Time{})
	require.ErrorContains(t, err, "Bad")
}

// TestValidateDefinitionRejectsMismatchedGenerator checks that an Int attribute with a pool generator is refused by
// name.
func TestValidateDefinitionRejectsMismatchedGenerator(t *testing.T) {
	def := &SearchAttributeDefinition{Name: "VisPriority", Type: enums.INDEXED_VALUE_TYPE_INT,
		Generator: poolGen(5, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 0)}
	require.ErrorContains(t, validateDefinition(def), "VisPriority")
}

// TestValidateDefinition checks that valid definitions pass and each invalid one is refused naming the attribute.
func TestValidateDefinition(t *testing.T) {
	bad := map[string]*SearchAttributeDefinition{
		"keyword with int range": {Name: "Bad", Type: enums.INDEXED_VALUE_TYPE_KEYWORD,
			Generator: intDef("", 1, 2).Generator},
		"missing generator": {Name: "Bad", Type: enums.INDEXED_VALUE_TYPE_INT},
		"zero cardinality":  keywordDef("Bad", 0, PoolDistribution_POOL_DISTRIBUTION_UNIFORM),
		"max_items on keyword": {Name: "Bad", Type: enums.INDEXED_VALUE_TYPE_KEYWORD,
			Generator: poolGen(5, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 2)},
		"max_items above cardinality": {Name: "Bad", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST,
			Generator: poolGen(2, PoolDistribution_POOL_DISTRIBUTION_UNIFORM, 3)},
		"int min above max":     intDef("Bad", 5, 4),
		"double min above max":  doubleDef("Bad", 5, 4),
		"true fraction above 1": boolDef("Bad", 1.5),
		"true fraction NaN":     boolDef("Bad", math.NaN()),
		"double min NaN":        doubleDef("Bad", math.NaN(), 4),
		"double max NaN":        doubleDef("Bad", 1, math.NaN()),
		"double min -Inf":       doubleDef("Bad", math.Inf(-1), 4),
		"double max +Inf":       doubleDef("Bad", 1, math.Inf(1)),
		"unknown distribution":  keywordDef("Bad", 5, PoolDistribution(7)),
		"offset min above max":  datetimeDef("Bad", time.Hour, 0),
		"name with a quote":     intDef("Bad'", 1, 2),
		"name with a space":     intDef("Bad name", 1, 2),
		"unsupported type": {Name: "Bad", Type: enums.INDEXED_VALUE_TYPE_UNSPECIFIED,
			Generator: intDef("", 1, 2).Generator},
		"missing name": intDef("", 1, 2),
	}
	for name, def := range bad {
		t.Run(name, func(t *testing.T) {
			err := validateDefinition(def)
			require.Error(t, err)
			if def.Name != "" {
				require.ErrorContains(t, err, "Bad")
			}
		})
	}
	for _, def := range []*SearchAttributeDefinition{
		keywordDef("Ok", 1, PoolDistribution_POOL_DISTRIBUTION_ZIPF),
		intDef("Ok", 1, 1),
		boolDef("Ok", 0),
		datetimeDef("Ok", 0, 0),
	} {
		require.NoError(t, validateDefinition(def))
	}
}

// TestTypedSearchAttributeUpdate checks that each generated value type maps to the SDK key for the attribute's type.
func TestTypedSearchAttributeUpdate(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	defs := []*SearchAttributeDefinition{
		keywordDef("K", 3, PoolDistribution_POOL_DISTRIBUTION_UNIFORM),
		{Name: "Tx", Type: enums.INDEXED_VALUE_TYPE_TEXT, Generator: poolGen(3, 0, 0)},
		{Name: "L", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST, Generator: poolGen(3, 0, 2)},
		intDef("I", 1, 5),
		doubleDef("D", 1, 5),
		boolDef("B", 1),
		datetimeDef("T", 0, time.Hour),
	}
	wantKinds := map[string]enums.IndexedValueType{
		"K": enums.INDEXED_VALUE_TYPE_KEYWORD, "Tx": enums.INDEXED_VALUE_TYPE_TEXT,
		"L": enums.INDEXED_VALUE_TYPE_KEYWORD_LIST, "I": enums.INDEXED_VALUE_TYPE_INT,
		"D": enums.INDEXED_VALUE_TYPE_DOUBLE, "B": enums.INDEXED_VALUE_TYPE_BOOL,
		"T": enums.INDEXED_VALUE_TYPE_DATETIME,
	}
	for _, def := range defs {
		t.Run(def.Name, func(t *testing.T) {
			v, err := generateValue(def, newRand(), now)
			require.NoError(t, err)
			u, err := typedSearchAttributeUpdate(def, v)
			require.NoError(t, err)
			keys := temporal.NewSearchAttributes(u).GetUntypedValues()
			require.Len(t, keys, 1)
			for k := range keys {
				require.Equal(t, def.Name, k.GetName())
				require.Equal(t, wantKinds[def.Name], k.GetValueType())
			}
		})
	}
	_, err := typedSearchAttributeUpdate(defs[0], struct{}{})
	require.ErrorContains(t, err, "K")
}

// TestSearchAttributePayloadSetsType checks that an upsert payload carries the IndexedValueType name the SDK writes
// (String(), for example "Bool") as its type metadata.
func TestSearchAttributePayloadSetsType(t *testing.T) {
	def := boolDef("VisPinned", 1)
	p, err := searchAttributePayload(def, true)
	require.NoError(t, err)
	require.Equal(t, "Bool", string(p.Metadata["type"]))
	var got bool
	require.NoError(t, converter.GetDefaultDataConverter().FromPayload(p, &got))
	require.True(t, got)
}

func newRand() *rand.Rand {
	return rand.New(rand.NewPCG(1, 2))
}

func poolOf(name string, n int) []string {
	pool := make([]string, n)
	for i := range pool {
		pool[i] = name + "-" + string(rune('0'+i))
	}
	return pool
}

func poolGen(card uint32, dist PoolDistribution, maxItems uint32) *ValueGenerator {
	return &ValueGenerator{Variant: &ValueGenerator_Pool{Pool: &PoolGenerator{
		Cardinality: card, Distribution: dist, MaxItems: maxItems}}}
}

func keywordDef(name string, card uint32, dist PoolDistribution) *SearchAttributeDefinition {
	return &SearchAttributeDefinition{Name: name, Type: enums.INDEXED_VALUE_TYPE_KEYWORD,
		Generator: poolGen(card, dist, 0)}
}

func intDef(name string, lo, hi int64) *SearchAttributeDefinition {
	return &SearchAttributeDefinition{Name: name, Type: enums.INDEXED_VALUE_TYPE_INT,
		Generator: &ValueGenerator{Variant: &ValueGenerator_IntRange{IntRange: &IntRangeGenerator{Min: lo, Max: hi}}}}
}

func doubleDef(name string, lo, hi float64) *SearchAttributeDefinition {
	return &SearchAttributeDefinition{Name: name, Type: enums.INDEXED_VALUE_TYPE_DOUBLE,
		Generator: &ValueGenerator{Variant: &ValueGenerator_DoubleRange{
			DoubleRange: &DoubleRangeGenerator{Min: lo, Max: hi}}}}
}

func boolDef(name string, frac float64) *SearchAttributeDefinition {
	return &SearchAttributeDefinition{Name: name, Type: enums.INDEXED_VALUE_TYPE_BOOL,
		Generator: &ValueGenerator{Variant: &ValueGenerator_Bool{Bool: &BoolGenerator{TrueFraction: frac}}}}
}

func datetimeDef(name string, lo, hi time.Duration) *SearchAttributeDefinition {
	return &SearchAttributeDefinition{Name: name, Type: enums.INDEXED_VALUE_TYPE_DATETIME,
		Generator: &ValueGenerator{Variant: &ValueGenerator_TimeOffset{TimeOffset: &TimeOffsetGenerator{
			MinOffset: durationpb.New(lo), MaxOffset: durationpb.New(hi)}}}}
}
