// Package visibility renders the List and Count queries a load run issues against visibility.
package visibility

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/enums/v1"
)

const (
	agoPrefix  = "ago:"
	wfidPrefix = "wfid:"
	inInfix    = ":in:"

	// maxListLen bounds {wfid:ids:N} and {Attr:in:N}, so a typo doesn't build a query of megabytes.
	maxListLen = 1000

	placeholderForms = "{Attr}, {Attr:in:N}, {ago:<duration>}, {wfid:prefix:narrow}, {wfid:prefix:broad}, " +
		"{wfid:prefix-end:narrow}, or {wfid:ids:N}"
)

const (
	segText segmentKind = iota
	// segAttr is {Attr}.
	segAttr
	// segAttrIn is {Attr:in:N}.
	segAttrIn
	// segAgo is {ago:<duration>}.
	segAgo
	// segPrefix is {wfid:prefix:narrow} or {wfid:prefix:broad}.
	segPrefix
	// segPrefixEnd is {wfid:prefix-end:narrow}.
	segPrefixEnd
	// segIDs is {wfid:ids:N}.
	segIDs
)

var attrNamePattern = kitchensink.AttributeNamePattern

type (
	// valueSource supplies what a query template's placeholders render to.
	valueSource interface {
		// Literal returns a fresh query literal for the declared attribute attr.
		Literal(attr string) (string, error)
		// Literals returns n distinct fresh query literals for the declared attribute attr, joined with commas.
		Literals(attr string, n int) (string, error)
		// WorkflowIDPrefix returns an unquoted prefix of the workflow IDs this execution starts. A broad prefix matches
		// every workflow in the run; a narrow one matches a handful of those started so far.
		WorkflowIDPrefix(broad bool) string
		// WorkflowIDs returns n unquoted IDs of workflows this execution has started.
		WorkflowIDs(n int) []string
		Now() time.Time
	}

	// Workflows names the workflows an execution starts, so reads can filter on their IDs.
	Workflows struct {
		RunID       string
		ExecutionID string
		// First is the first iteration this execution runs. 0 means 1, where omes starts unless it resumes a run.
		First int
		// Started is one past the highest iteration started, so the started IDs end in First through Started-1. 0
		// means none have started.
		Started int
	}

	// definedValues renders attributes from definitions, a seeded rand, and a fixed clock.
	definedValues struct {
		defs      map[string]*kitchensink.SearchAttributeDefinition
		rand      *rand.Rand
		now       time.Time
		workflows Workflows
	}

	segmentKind int

	// segment is a run of query text or one placeholder.
	segment struct {
		kind segmentKind
		text string
		attr string
		ago  time.Duration
		// n is the list length of segAttrIn and segIDs.
		n     int
		broad bool
	}
)

// newValues returns a valueSource that draws attribute values from defs and workflow IDs from w, using r and now.
func newValues(defs map[string]*kitchensink.SearchAttributeDefinition, r *rand.Rand, now time.Time,
	w Workflows) valueSource {
	return &definedValues{defs: defs, rand: r, now: now, workflows: w}
}

// ID is the workflow ID of the given iteration, as omes's DefaultStartWorkflowOptions names it.
func (w Workflows) ID(iteration int) string {
	return fmt.Sprintf("%s%d", w.idPrefix(), iteration)
}

func (w Workflows) idPrefix() string {
	return fmt.Sprintf("w-%s-%s-", w.RunID, w.ExecutionID)
}

func (w Workflows) first() int {
	return max(w.First, 1)
}

func (v *definedValues) Literal(attr string) (string, error) {
	def, ok := v.defs[attr]
	if !ok {
		return "", fmt.Errorf("undeclared search attribute %q", attr)
	}
	return kitchensink.QueryLiteral(def, v.rand, v.now)
}

func (v *definedValues) Literals(attr string, n int) (string, error) {
	def, ok := v.defs[attr]
	if !ok {
		return "", fmt.Errorf("undeclared search attribute %q", attr)
	}
	return kitchensink.QueryLiterals(def, v.rand, n)
}

// WorkflowIDPrefix's narrow prefix is a started workflow's ID without its last digit, drawn from the iterations with
// the most digits so far. Iterations with more digits haven't started, so it matches at most eleven workflow IDs. Each
// ID still covers several rows: one per continue-as-new run, plus its children, which have their own IDs. The broad
// prefix covers every workflow omes starts, not their children.
func (v *definedValues) WorkflowIDPrefix(broad bool) string {
	if broad {
		return fmt.Sprintf("w-%s-", v.workflows.RunID)
	}
	started := v.workflows.Started
	if started <= 10 || started <= v.workflows.first() {
		return v.workflows.idPrefix()
	}
	lo := 1
	for lo*10 <= started-1 {
		lo *= 10
	}
	// A resumed run never started the iterations below First under this execution ID.
	lo = max(lo, v.workflows.first())
	return v.workflows.ID((lo + v.rand.IntN(started-lo)) / 10)
}

func (v *definedValues) WorkflowIDs(n int) []string {
	first := v.workflows.first()
	// Before any iteration starts, name the first one rather than one that never exists.
	started := max(v.workflows.Started, first+1)
	ids := make([]string, n)
	for i := range ids {
		ids[i] = v.workflows.ID(first + v.rand.IntN(started-first))
	}
	return ids
}

func (v *definedValues) Now() time.Time {
	return v.now
}

// checkQuery reports whether query is a valid template whose attribute placeholders all name attributes in defs.
func checkQuery(query string, defs map[string]*kitchensink.SearchAttributeDefinition) error {
	if strings.TrimSpace(query) == "" {
		return errors.New("required")
	}
	segs, err := parseQuery(query)
	if err != nil {
		return err
	}
	var errs []error
	for _, s := range segs {
		switch s.kind {
		case segAttr:
			if _, ok := defs[s.attr]; !ok {
				errs = append(errs, fmt.Errorf("placeholder {%s} names an undeclared search attribute", s.attr))
			}
		case segAttrIn:
			if err := checkAttrIn(s, defs); err != nil {
				errs = append(errs, fmt.Errorf("placeholder {%s%s%d}: %w", s.attr, inInfix, s.n, err))
			}
		default:
		}
	}
	return errors.Join(errs...)
}

func checkAttrIn(s segment, defs map[string]*kitchensink.SearchAttributeDefinition) error {
	def, ok := defs[s.attr]
	if !ok {
		return errors.New("names an undeclared search attribute")
	}
	typ := def.GetType()
	if typ != enums.INDEXED_VALUE_TYPE_KEYWORD && typ != enums.INDEXED_VALUE_TYPE_KEYWORD_LIST {
		return fmt.Errorf("IN lists need a Keyword or KeywordList attribute, not %s", typ)
	}
	if card := int(def.GetGenerator().GetPool().GetCardinality()); s.n > card {
		return fmt.Errorf("asks for %d distinct values of an attribute with cardinality %d", s.n, card)
	}
	return nil
}

// render fills query's placeholders from src: {Attr} becomes src.Literal("Attr"), {Attr:in:5} five distinct
// literals joined with commas, and {ago:15m} the quoted RFC3339 time 15m before src.Now(). {wfid:prefix:narrow} and
// {wfid:prefix:broad} become a quoted workflow ID prefix, {wfid:prefix-end:narrow} the narrow prefix with ~ appended,
// an upper bound that turns the prefix into a range, and {wfid:ids:5} five quoted workflow IDs joined with commas.
// Every narrow placeholder in one render shares one draw, so a range's bounds agree.
func render(query string, src valueSource) (string, error) {
	segs, err := parseQuery(query)
	if err != nil {
		return "", err
	}
	var narrow string
	narrowPrefix := func() string {
		if narrow == "" {
			narrow = src.WorkflowIDPrefix(false)
		}
		return narrow
	}
	var b strings.Builder
	for _, s := range segs {
		switch s.kind {
		case segAttr:
			lit, err := src.Literal(s.attr)
			if err != nil {
				return "", fmt.Errorf("placeholder {%s}: %w", s.attr, err)
			}
			b.WriteString(lit)
		case segAttrIn:
			lits, err := src.Literals(s.attr, s.n)
			if err != nil {
				return "", fmt.Errorf("placeholder {%s%s%d}: %w", s.attr, inInfix, s.n, err)
			}
			b.WriteString(lits)
		case segAgo:
			b.WriteString(quote(src.Now().Add(-s.ago).UTC().Format(time.RFC3339)))
		case segPrefix:
			if s.broad {
				b.WriteString(quote(src.WorkflowIDPrefix(true)))
			} else {
				b.WriteString(quote(narrowPrefix()))
			}
		case segPrefixEnd:
			b.WriteString(quote(narrowPrefix() + "~"))
		case segIDs:
			ids := src.WorkflowIDs(s.n)
			for i, id := range ids {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(quote(id))
			}
		default:
			b.WriteString(s.text)
		}
	}
	return b.String(), nil
}

func parseQuery(query string) ([]segment, error) {
	var segs []segment
	rest := query
	for rest != "" {
		open := strings.IndexAny(rest, "{}")
		if open < 0 {
			segs = append(segs, segment{kind: segText, text: rest})
			break
		}
		if rest[open] == '}' {
			return nil, fmt.Errorf("unmatched } at offset %d", len(query)-len(rest)+open)
		}
		if open > 0 {
			segs = append(segs, segment{kind: segText, text: rest[:open]})
		}
		end := strings.IndexAny(rest[open+1:], "{}")
		if end < 0 || rest[open+1+end] != '}' {
			return nil, fmt.Errorf("unclosed { at offset %d", len(query)-len(rest)+open)
		}
		seg, err := parsePlaceholder(rest[open+1 : open+1+end])
		if err != nil {
			return nil, err
		}
		segs = append(segs, seg)
		rest = rest[open+1+end+1:]
	}
	return segs, nil
}

func parsePlaceholder(body string) (segment, error) {
	if d, ok := strings.CutPrefix(body, agoPrefix); ok {
		ago, err := time.ParseDuration(d)
		if err != nil {
			return segment{}, fmt.Errorf("placeholder {%s}: %w", body, err)
		}
		if ago < 0 {
			return segment{}, fmt.Errorf("placeholder {%s}: duration must not be negative", body)
		}
		return segment{kind: segAgo, ago: ago}, nil
	}
	if spec, ok := strings.CutPrefix(body, wfidPrefix); ok {
		return parseWorkflowIDPlaceholder(body, spec)
	}
	if attr, count, ok := strings.Cut(body, inInfix); ok && attrNamePattern.MatchString(attr) {
		n, err := parseListLen(body, count)
		if err != nil {
			return segment{}, err
		}
		return segment{kind: segAttrIn, attr: attr, n: n}, nil
	}
	if !attrNamePattern.MatchString(body) {
		return segment{}, fmt.Errorf("placeholder {%s}: want %s", body, placeholderForms)
	}
	return segment{kind: segAttr, attr: body}, nil
}

func parseWorkflowIDPlaceholder(body, spec string) (segment, error) {
	switch spec {
	case "prefix:narrow":
		return segment{kind: segPrefix}, nil
	case "prefix:broad":
		return segment{kind: segPrefix, broad: true}, nil
	case "prefix-end:narrow":
		return segment{kind: segPrefixEnd}, nil
	default:
	}
	if count, ok := strings.CutPrefix(spec, "ids:"); ok {
		n, err := parseListLen(body, count)
		if err != nil {
			return segment{}, err
		}
		return segment{kind: segIDs, n: n}, nil
	}
	return segment{}, fmt.Errorf("placeholder {%s}: want %s", body, placeholderForms)
}

func parseListLen(body, count string) (int, error) {
	n, err := strconv.Atoi(count)
	if err != nil || n < 1 || n > maxListLen {
		return 0, fmt.Errorf("placeholder {%s}: list length must be an integer in [1, %d]", body, maxListLen)
	}
	return n, nil
}

func quote(s string) string {
	return "'" + s + "'"
}
