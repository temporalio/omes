package visibility

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/enums/v1"
)

// TestRender verifies that attribute and ago placeholders render from the value source and other text passes through.
func TestRender(t *testing.T) {
	src := &fixedValues{
		literals: map[string]string{"Tenant": "'Tenant-3'", "Priority": "7"},
		now:      time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
	}
	got, err := render("Tenant = {Tenant} AND Priority >= {Priority} AND CloseTime > {ago:15m} AND StartTime < {ago:0s}",
		src)
	require.NoError(t, err)
	require.Equal(t,
		"Tenant = 'Tenant-3' AND Priority >= 7 AND CloseTime > '2026-01-02T02:49:05Z' AND StartTime < '2026-01-02T03:04:05Z'",
		got)
}

// TestRenderListAndWorkflowIDPlaceholders verifies that IN list and workflow ID placeholders render from the value
// source, and that a range's narrow bounds share one draw.
func TestRenderListAndWorkflowIDPlaceholders(t *testing.T) {
	src := &fixedValues{
		lists:   map[string]string{"Build": "'Build-4', 'Build-1'"},
		narrows: []string{"w-run-exec-12", "w-run-exec-98"},
		broad:   "w-run-",
		ids:     []string{"w-run-exec-3", "w-run-exec-5"},
	}
	got, err := render("WorkflowId >= {wfid:prefix:narrow} AND WorkflowId < {wfid:prefix-end:narrow} "+
		"AND WorkflowId STARTS_WITH {wfid:prefix:broad} AND WorkflowId IN ({wfid:ids:2}) AND Build IN ({Build:in:2})", src)
	require.NoError(t, err)
	require.Equal(t, "WorkflowId >= 'w-run-exec-12' AND WorkflowId < 'w-run-exec-12~' "+
		"AND WorkflowId STARTS_WITH 'w-run-' AND WorkflowId IN ('w-run-exec-3', 'w-run-exec-5') "+
		"AND Build IN ('Build-4', 'Build-1')", got)

	// the next render draws a fresh narrow prefix
	got, err = render("WorkflowId < {wfid:prefix-end:narrow}", src)
	require.NoError(t, err)
	require.Equal(t, "WorkflowId < 'w-run-exec-98~'", got)
}

// TestRenderErrors verifies that malformed placeholders and value source failures surface as errors.
func TestRenderErrors(t *testing.T) {
	forms := "want {Attr}, {Attr:in:N}, {ago:<duration>}, {wfid:prefix:narrow}, {wfid:prefix:broad}, " +
		"{wfid:prefix-end:narrow}, or {wfid:ids:N}"
	tests := []struct {
		query string
		want  string
	}{
		{"A = {Tenant", "unclosed { at offset 4"},
		{"A = {Ten{ant}", "unclosed { at offset 4"},
		{"A = Tenant}", "unmatched } at offset 10"},
		{"A > {ago:soon}", `placeholder {ago:soon}: time: invalid duration "soon"`},
		{"A > {ago:-1h}", "placeholder {ago:-1h}: duration must not be negative"},
		{"A = {}", "placeholder {}: " + forms},
		{"A = {Ten ant}", "placeholder {Ten ant}: " + forms},
		{"A = {Missing}", "placeholder {Missing}: no value for Missing"},
		{"A IN ({Tenant:in:x})", "placeholder {Tenant:in:x}: list length must be an integer in [1, 1000]"},
		{"A IN ({Tenant:in:0})", "placeholder {Tenant:in:0}: list length must be an integer in [1, 1000]"},
		{"A IN ({Tenant:in:1001})", "placeholder {Tenant:in:1001}: list length must be an integer in [1, 1000]"},
		{"A IN ({Ten ant:in:2})", "placeholder {Ten ant:in:2}: " + forms},
		{"A IN ({Missing:in:2})", "placeholder {Missing:in:2}: no list for Missing"},
		{"A >= {wfid:prefix:wide}", "placeholder {wfid:prefix:wide}: " + forms},
		{"A < {wfid:prefix-end:broad}", "placeholder {wfid:prefix-end:broad}: " + forms},
		{"A IN ({wfid:ids:-3})", "placeholder {wfid:ids:-3}: list length must be an integer in [1, 1000]"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			_, err := render(tt.query, &fixedValues{})
			require.ErrorContains(t, err, tt.want)
		})
	}
}

// TestCheckQuery verifies that checkQuery accepts templates whose placeholders name declared attributes and rejects
// empty templates, undeclared attributes, IN lists on the wrong type or larger than the pool, and malformed syntax.
func TestCheckQuery(t *testing.T) {
	defs := testDefs()
	tests := []struct {
		query string
		want  string
	}{
		{"VisTenant = {VisTenant} AND VisPriority > {VisPriority} AND StartTime > {ago:1h}", ""},
		{"VisTenant IN ({VisTenant:in:50}) AND VisTags IN ({VisTags:in:3})", ""},
		{"WorkflowId IN ({wfid:ids:3}) AND WorkflowId STARTS_WITH {wfid:prefix:broad}", ""},
		{"  ", "required"},
		{"A = {VisTenant", "unclosed { at offset 4"},
		{"A = {Nope}", "placeholder {Nope} names an undeclared search attribute"},
		{"A IN ({Nope:in:2})", "placeholder {Nope:in:2}: names an undeclared search attribute"},
		{"A IN ({VisPriority:in:2})", "placeholder {VisPriority:in:2}: IN lists need a Keyword or KeywordList attribute"},
		{"A IN ({VisTenant:in:51})",
			"placeholder {VisTenant:in:51}: asks for 51 distinct values of an attribute with cardinality 50"},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			err := checkQuery(tt.query, defs)
			if tt.want == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.want)
			}
		})
	}
}

// TestRenderWithNewValues verifies that templates render end to end from definitions, leave no placeholder behind,
// and give the same query for the same seed.
func TestRenderWithNewValues(t *testing.T) {
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	query := "VisTenant = {VisTenant} AND VisPriority = {VisPriority} AND VisTags IN ({VisTags:in:3}) " +
		"AND StartTime > {ago:15m} AND WorkflowId STARTS_WITH {wfid:prefix:narrow} AND WorkflowId IN ({wfid:ids:2})"
	render := func() string {
		got, err := render(query, newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), now, testWorkflows(5000)))
		require.NoError(t, err)
		return got
	}
	got := render()
	require.NotContains(t, got, "{")
	require.NotContains(t, got, "}")
	require.Contains(t, got, "StartTime > '2026-01-02T02:49:05Z'")
	require.Equal(t, got, render())
}

// TestValuesUndeclared verifies that the value source refuses an attribute the definitions don't declare.
func TestValuesUndeclared(t *testing.T) {
	src := newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), time.Now(), testWorkflows(1))
	_, err := render("X = {Nope}", src)
	require.ErrorContains(t, err, `placeholder {Nope}: undeclared search attribute "Nope"`)
	_, err = render("X IN ({Nope:in:2})", src)
	require.ErrorContains(t, err, `placeholder {Nope:in:2}: undeclared search attribute "Nope"`)
}

// TestValuesNarrowPrefix verifies that a narrow prefix matches between one and eleven of the workflow IDs started so
// far, and that a run with ten or fewer started falls back to the execution's own prefix.
func TestValuesNarrowPrefix(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for _, started := range []int{0, 10} {
		require.Equal(t, "w-run-exec-",
			newValues(testDefs(), r, time.Now(), testWorkflows(started)).WorkflowIDPrefix(false))
	}
	for _, started := range []int{11, 99, 100, 2345} {
		w := testWorkflows(started)
		src := newValues(testDefs(), r, time.Now(), w)
		for range 200 {
			prefix := src.WorkflowIDPrefix(false)
			matched := 0
			for i := 1; i < started; i++ {
				if strings.HasPrefix(w.ID(i), prefix) {
					matched++
				}
			}
			require.True(t, matched >= 1 && matched <= 11, "started %d: prefix %q matches %d", started, prefix,
				matched)
		}
	}
}

// TestValuesResumedRun verifies that a run resumed at a later iteration only names workflow IDs this execution
// started, by ID and by narrow prefix.
func TestValuesResumedRun(t *testing.T) {
	w := Workflows{RunID: "run", ExecutionID: "exec", First: 50, Started: 60}
	src := newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), time.Now(), w)
	isStarted := func(id string) bool {
		for i := w.First; i < w.Started; i++ {
			if w.ID(i) == id {
				return true
			}
		}
		return false
	}
	for range 200 {
		for _, id := range src.WorkflowIDs(3) {
			require.True(t, isStarted(id), "%s was never started", id)
		}
		prefix := src.WorkflowIDPrefix(false)
		matched := 0
		for i := w.First; i < w.Started; i++ {
			if strings.HasPrefix(w.ID(i), prefix) {
				matched++
			}
		}
		require.True(t, matched >= 1, "prefix %q matches no started workflow", prefix)
	}
}

// TestValuesNarrowPrefixBeforeFirstStart verifies that a resumed run with no iteration started yet falls back to the
// execution's own prefix.
func TestValuesNarrowPrefixBeforeFirstStart(t *testing.T) {
	for _, started := range []int{20, 50} {
		w := Workflows{RunID: "run", ExecutionID: "exec", First: 50, Started: started}
		src := newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), time.Now(), w)
		require.Equal(t, "w-run-exec-", src.WorkflowIDPrefix(false))
	}
}

// TestValuesBroadPrefix verifies that a broad prefix covers the workflows of every execution in the run.
func TestValuesBroadPrefix(t *testing.T) {
	w := testWorkflows(3)
	prefix := newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), time.Now(), w).WorkflowIDPrefix(true)
	require.Equal(t, "w-run-", prefix)
	other := w
	other.ExecutionID = "other"
	require.True(t, strings.HasPrefix(other.ID(7), prefix))
}

// TestWorkflowsID verifies that workflow IDs use the format omes gives the workflow it starts for an iteration.
func TestWorkflowsID(t *testing.T) {
	require.Equal(t, "w-run-exec-7", testWorkflows(8).ID(7))
}

// TestValuesWorkflowIDs verifies that workflow IDs name iterations that have started, never iteration 0, which omes
// doesn't run, and that a run with none started yet names the first iteration.
func TestValuesWorkflowIDs(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	ids := newValues(testDefs(), r, time.Now(), testWorkflows(20)).WorkflowIDs(300)
	require.Len(t, ids, 300)
	for _, id := range ids {
		n, ok := strings.CutPrefix(id, "w-run-exec-")
		require.True(t, ok, id)
		i, err := strconv.Atoi(n)
		require.NoError(t, err, id)
		require.True(t, i >= 1 && i < 20, id)
	}
	require.Equal(t, []string{"w-run-exec-1", "w-run-exec-1"},
		newValues(testDefs(), r, time.Now(), testWorkflows(0)).WorkflowIDs(2))
}

// TestValuesLiteralsTooMany verifies that the value source refuses more distinct values than the pool holds.
func TestValuesLiteralsTooMany(t *testing.T) {
	src := newValues(testDefs(), rand.New(rand.NewPCG(1, 2)), time.Now(), testWorkflows(1))
	_, err := src.Literals("VisTenant", 51)
	require.ErrorContains(t, err, "cardinality 50")
}

// fixedValues renders from fixed values. Each narrow prefix draw takes the next of narrows.
type fixedValues struct {
	literals map[string]string
	lists    map[string]string
	narrows  []string
	broad    string
	ids      []string
	now      time.Time
}

func (f *fixedValues) Literal(attr string) (string, error) {
	v, ok := f.literals[attr]
	if !ok {
		return "", errors.New("no value for " + attr)
	}
	return v, nil
}

func (f *fixedValues) Literals(attr string, n int) (string, error) {
	v, ok := f.lists[attr]
	if !ok {
		return "", errors.New("no list for " + attr)
	}
	if got := len(strings.Split(v, ", ")); got != n {
		return "", fmt.Errorf("list for %s has %d values, asked for %d", attr, got, n)
	}
	return v, nil
}

func (f *fixedValues) WorkflowIDPrefix(broad bool) string {
	if broad {
		return f.broad
	}
	next := f.narrows[0]
	f.narrows = f.narrows[1:]
	return next
}

func (f *fixedValues) WorkflowIDs(n int) []string {
	return f.ids[:n]
}

func (f *fixedValues) Now() time.Time {
	return f.now
}

func testWorkflows(started int) Workflows {
	return Workflows{RunID: "run", ExecutionID: "exec", Started: started}
}

func testDefs() map[string]*kitchensink.SearchAttributeDefinition {
	pool := func(card, maxItems uint32) *kitchensink.ValueGenerator {
		return &kitchensink.ValueGenerator{Variant: &kitchensink.ValueGenerator_Pool{Pool: &kitchensink.PoolGenerator{
			Cardinality: card, Distribution: kitchensink.PoolDistribution_POOL_DISTRIBUTION_UNIFORM,
			MaxItems: maxItems}}}
	}
	return map[string]*kitchensink.SearchAttributeDefinition{
		"VisTenant": {Name: "VisTenant", Type: enums.INDEXED_VALUE_TYPE_KEYWORD, Generator: pool(50, 0)},
		"VisTags":   {Name: "VisTags", Type: enums.INDEXED_VALUE_TYPE_KEYWORD_LIST, Generator: pool(20, 3)},
		"VisPriority": {Name: "VisPriority", Type: enums.INDEXED_VALUE_TYPE_INT,
			Generator: &kitchensink.ValueGenerator{Variant: &kitchensink.ValueGenerator_IntRange{
				IntRange: &kitchensink.IntRangeGenerator{Min: 1, Max: 100}}}},
	}
}
