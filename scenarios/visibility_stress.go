// Package scenarios
// Visibility stress test scenario.
//
// This scenario stress-tests the Temporal visibility store (Elasticsearch or SQL) by
// generating controlled read and write traffic. It uses a hybrid executor + workflow model:
// the executor spawns short-lived workflows with instructions baked into their input (no
// signals), and separate goroutines handle deletes and queries.
//
// Three independent presets control behavior:
//   - loadPreset:  write traffic (workflow starts, CSA updates, deletes, failure rates)
//   - queryPreset: read traffic (List/Count queries with varying filter complexity)
//   - csaPreset:   which custom search attributes to register and use
package scenarios

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/omes/loadgen"
	vstypes "github.com/temporalio/omes/loadgen/visibility"
)

// ---------------------------------------------------------------------------
// Presets
//
// Three preset types control the scenario's behavior. Each can be selected
// via --option flags and overridden with individual --option key=value pairs.
// ---------------------------------------------------------------------------

// vsLoadPreset controls write-side traffic: workflow creation rate, CSA update
// frequency, explicit deletion rate, and the distribution of terminal statuses.
type vsLoadPreset struct {
	WfRPS          float64       // Workflow starts per second.
	UpdatesPerWF   float64       // CSA upsert calls per workflow (can be fractional, e.g., 0.5).
	UpdateDelay    time.Duration // Sleep between consecutive CSA updates within a workflow.
	DeleteRPS      float64       // Explicit deletes per second (0 = rely on retention only).
	FailPercent    float64       // Fraction of WFs that intentionally fail (0.10 = 10%).
	TimeoutPercent float64       // Fraction of WFs that intentionally timeout (0.05 = 5%).
	// TODO: MemoUpdatesPerWF float64
	// TODO: MemoSizeBytes    int
}

// vsQueryPreset controls read-side traffic: query rate, the weighted distribution
// of query complexity (no-filter, open, closed, simple CSA, compound CSA,
// disjunction), and the selectivity distribution used to build CSA predicates.
type vsQueryPreset struct {
	CountRPS              float64 // CountWorkflowExecutions calls per second.
	ListRPS               float64 // ListWorkflowExecutions calls per second.
	ListNoFilterWeight    int     // Weight for unfiltered list queries.
	ListOpenWeight        int     // Weight for ExecutionStatus = 'Running' queries.
	ListClosedWeight      int     // Weight for closed + time range queries.
	ListSimpleCSAWeight   int     // Weight for single CSA filter queries.
	ListCompoundCSAWeight int     // Weight for multi-CSA ANDed filter queries.
	ListDisjunctionWeight int     // Weight for multi-CSA ORed filter queries.

	// SelMu and SelSigma parameterize the log-normal selectivity distribution:
	// log10(selectivity) ~ Normal(SelMu, SelSigma). SelMu=-2 puts the median
	// clause at 1% of the corpus. See vsRand.sampleSelectivity.
	SelMu    float64
	SelSigma float64

	// PageContinueProb is the chance of fetching one more page after each page of
	// a List result, giving a geometric page-count distribution. MaxPages
	// truncates it. See visibilityStressExecutor.shouldFetchNextPage.
	PageContinueProb float64
	MaxPages         int
}

var vsLoadPresets = map[string]vsLoadPreset{
	"light": {
		WfRPS: 10, UpdatesPerWF: 5,
		UpdateDelay: 1 * time.Second, DeleteRPS: 2,
		FailPercent: 0.10, TimeoutPercent: 0.05,
	},
	"moderate": {
		WfRPS: 100, UpdatesPerWF: 10,
		UpdateDelay: 1 * time.Second, DeleteRPS: 20,
		FailPercent: 0.10, TimeoutPercent: 0.05,
	},
	"heavy": {
		WfRPS: 1000, UpdatesPerWF: 20,
		UpdateDelay: 500 * time.Millisecond, DeleteRPS: 200,
		FailPercent: 0.10, TimeoutPercent: 0.05,
	},
	"no-failures": {
		WfRPS: 100, UpdatesPerWF: 10,
		UpdateDelay: 1 * time.Second, DeleteRPS: 20,
		FailPercent: 0, TimeoutPercent: 0,
	},
}

var vsQueryPresets = map[string]vsQueryPreset{
	"light": {
		CountRPS: 1, ListRPS: 2,
		ListNoFilterWeight: 3, ListOpenWeight: 2, ListClosedWeight: 2,
		ListSimpleCSAWeight: 2, ListCompoundCSAWeight: 1, ListDisjunctionWeight: 1,
		SelMu: vsDefaultSelMu, SelSigma: vsDefaultSelSigma,
		PageContinueProb: vsDefaultPageContinueProb, MaxPages: vsDefaultMaxQueryPages,
	},
	"moderate": {
		CountRPS: 5, ListRPS: 10,
		ListNoFilterWeight: 3, ListOpenWeight: 2, ListClosedWeight: 2,
		ListSimpleCSAWeight: 2, ListCompoundCSAWeight: 1, ListDisjunctionWeight: 1,
		SelMu: vsDefaultSelMu, SelSigma: vsDefaultSelSigma,
		PageContinueProb: vsDefaultPageContinueProb, MaxPages: vsDefaultMaxQueryPages,
	},
	"heavy": {
		CountRPS: 10, ListRPS: 25,
		ListNoFilterWeight: 1, ListOpenWeight: 2, ListClosedWeight: 2,
		ListSimpleCSAWeight: 3, ListCompoundCSAWeight: 2, ListDisjunctionWeight: 2,
		SelMu: vsDefaultSelMu, SelSigma: vsDefaultSelSigma,
		PageContinueProb: vsDefaultPageContinueProb, MaxPages: vsDefaultMaxQueryPages,
	},
}

var vsCSAPresets = map[string][]string{
	"small": {
		"VS_Int_01", "VS_Keyword_01", "VS_Bool_01",
		"VS_Double_01", "VS_Text_01", "VS_Datetime_01",
	},
	"medium": {
		"VS_Int_01", "VS_Int_02",
		"VS_Keyword_01", "VS_Keyword_02",
		"VS_Bool_01",
		"VS_Double_01", "VS_Double_02",
		"VS_Text_01",
		"VS_Datetime_01", "VS_Datetime_02",
	},
	"heavy": {
		"VS_Int_01", "VS_Int_02", "VS_Int_03", "VS_Int_04", "VS_Int_05",
		"VS_Keyword_01", "VS_Keyword_02", "VS_Keyword_03", "VS_Keyword_04", "VS_Keyword_05",
		"VS_Bool_01", "VS_Bool_02",
		"VS_Double_01", "VS_Double_02", "VS_Double_03",
		"VS_Text_01", "VS_Text_02", "VS_Text_03",
		"VS_Datetime_01", "VS_Datetime_02",
	},
}

const (
	// vsDefaultVocabZipfSkew is the exponent of the Zipf distribution used to draw
	// vocabulary words on the write side. Values must be > 1; larger means more
	// skewed. 1.2 gives a long but not degenerate tail: the most popular word lands
	// on roughly a fifth of documents while the rarest lands on well under 1%.
	vsDefaultVocabZipfSkew = 1.2

	// vsDefaultSelMu and vsDefaultSelSigma parameterize log10(selectivity) for
	// generated CSA predicates. vsDefaultSelMu is only the read-only-mode fallback:
	// when a loadPreset is set, selMu is derived from the corpus that preset builds.
	// See deriveSelMu.
	vsDefaultSelMu    = -2.0
	vsDefaultSelSigma = 1.0

	// vsTargetMedianHits is the result-set size, in documents, that the median
	// generated query aims for when selMu is derived rather than supplied. Roughly
	// one page: deep enough to exercise the sort, short of a full scan.
	vsTargetMedianHits = 1000.0

	// vsMinDerivedSelMu floors the derived selMu so a huge corpus can't produce a
	// predicate so narrow that every query matches nothing.
	vsMinDerivedSelMu = -6.0

	// vsDefaultPageContinueProb is the chance a List query fetches one more page.
	// 0.4 means ~60% of queries read a single page and the mean is ~1.7, which is
	// roughly how clients behave: most read the first page of a UI or CLI listing
	// and only some scroll on.
	vsDefaultPageContinueProb = 0.4

	// vsDefaultMaxQueryPages truncates the geometric page count. At the default
	// continuation probability the odds of reaching it are ~0.03%, so it bounds
	// the worst case without meaningfully clipping the tail. A fixed low cap
	// would truncate exactly the deep-pagination tail worth measuring.
	vsDefaultMaxQueryPages = 10

	// vsMinSelectivity floors the sampled selectivity so an extreme draw can't
	// produce a predicate that is unsatisfiable by construction.
	vsMinSelectivity = 1e-6

	// vsTextWordsMin and vsTextWordsMax bound the number of vocabulary words packed
	// into a generated Text CSA value. Text fields are analyzed, so a value built
	// from vocabulary words is matchable by the same words the querier filters on.
	vsTextWordsMin = 3
	vsTextWordsMax = 8
)

var keywordVocabulary = []string{
	"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel",
	"india", "juliet", "kilo", "lima", "mike", "november", "oscar", "papa",
	"quebec", "romeo", "sierra", "tango", "uniform", "victor", "whiskey", "xray",
	"yankee", "zulu", "red", "blue", "green", "yellow", "orange", "purple",
	"black", "white", "silver", "gold", "copper", "iron", "steel", "bronze",
	"north", "south", "east", "west", "up", "down", "left", "right", "center",
	"spring", "summer", "autumn", "winter", "dawn", "dusk", "noon", "midnight",
	"rain", "snow", "wind", "storm", "cloud", "sun", "moon", "star",
	"river", "lake", "ocean", "mountain", "valley", "forest", "desert", "island",
	"apple", "cherry", "mango", "peach", "plum", "grape", "lemon", "melon",
	"oak", "pine", "elm", "ash", "birch", "cedar", "maple", "willow",
	"hawk", "wolf", "bear", "deer", "fox", "owl", "eagle", "lion",
	"ruby", "jade", "opal", "onyx",
}

// ---------------------------------------------------------------------------
// Randomness
//
// math/rand.Rand is not safe for concurrent use. Writer goroutines own separate
// vsRand instances; query goroutines briefly serialize access to the executor's
// query RNG while constructing filters and making pagination decisions. The Zipf
// generator is bound to its parent Rand, which is why it lives here rather than
// as a package-level var.
// ---------------------------------------------------------------------------

type vsRand struct {
	*rand.Rand
	vocabZipf *rand.Zipf
}

func newVSRand(seed int64, vocabZipfSkew float64) *vsRand {
	r := rand.New(rand.NewSource(seed))
	return &vsRand{
		Rand:      r,
		vocabZipf: rand.NewZipf(r, vocabZipfSkew, 1.0, uint64(len(keywordVocabulary)-1)),
	}
}

// vocabWord returns a vocabulary word with Zipf-distributed popularity, for use
// on the write side.
//
// Key popularity in real workloads is power-law distributed, and that skew is
// what makes visibility query cost heavy-tailed: most keyword filters hit a rare
// value and match almost nothing, while the occasional one hits a hot value and
// scans a large slice of the corpus. Drawing write values uniformly instead
// pins every keyword filter at exactly 1/len(keywordVocabulary) selectivity,
// which collapses that spread to a single point.
//
// The read side deliberately keeps drawing uniformly (see csaFilterClause), so
// the skew shows up as variance across queries rather than as a constant shift.
func (r *vsRand) vocabWord() string {
	return keywordVocabulary[r.vocabZipf.Uint64()]
}

// sampleSelectivity draws the fraction of the corpus that a generated predicate
// should match, with log10(selectivity) ~ Normal(mu, sigma).
//
// Sampling selectivity and inverting to a threshold — rather than sampling a
// threshold and accepting whatever selectivity results — makes the query cost
// distribution a stated input of the scenario instead of an artifact of the
// write-side value ranges. Uniform thresholds over uniform values yield uniform
// selectivity, which is neither realistic nor tunable.
func (r *vsRand) sampleSelectivity(mu, sigma float64) float64 {
	s := math.Pow(10, mu+sigma*r.NormFloat64())
	return math.Min(1, math.Max(vsMinSelectivity, s))
}

// ---------------------------------------------------------------------------
// CSA Definition
//
// CSA (Custom Search Attribute) names follow the convention VS_<Type>_<N>,
// e.g., "VS_Int_01", "VS_Keyword_02". The type is inferred from the prefix,
// which is the same convention used by the workflow's type dispatch helper
// and the query generator.
// ---------------------------------------------------------------------------

type csaType int

const (
	csaTypeInt csaType = iota
	csaTypeKeyword
	csaTypeBool
	csaTypeDouble
	csaTypeText
	csaTypeDatetime
)

type csaDef struct {
	Name string
	Type csaType
}

func parseCSAName(name string) (csaDef, error) {
	switch {
	case strings.HasPrefix(name, "VS_Int_"):
		return csaDef{Name: name, Type: csaTypeInt}, nil
	case strings.HasPrefix(name, "VS_Keyword_"):
		return csaDef{Name: name, Type: csaTypeKeyword}, nil
	case strings.HasPrefix(name, "VS_Bool_"):
		return csaDef{Name: name, Type: csaTypeBool}, nil
	case strings.HasPrefix(name, "VS_Double_"):
		return csaDef{Name: name, Type: csaTypeDouble}, nil
	case strings.HasPrefix(name, "VS_Text_"):
		return csaDef{Name: name, Type: csaTypeText}, nil
	case strings.HasPrefix(name, "VS_Datetime_"):
		return csaDef{Name: name, Type: csaTypeDatetime}, nil
	default:
		return csaDef{}, fmt.Errorf("unrecognized CSA prefix in %q", name)
	}
}

// usesSelectivityThreshold reports whether a filter clause over this CSA can be
// aimed at a target selectivity by picking a threshold. Keyword and Text
// selectivity is set by the Zipf popularity of the drawn word, and Bool is fixed
// at roughly one half; none of the three respond to selMu.
func (c csaDef) usesSelectivityThreshold() bool {
	switch c.Type {
	case csaTypeInt, csaTypeDouble, csaTypeDatetime:
		return true
	default:
		return false
	}
}

func (c csaDef) indexedValueType() enums.IndexedValueType {
	switch c.Type {
	case csaTypeInt:
		return enums.INDEXED_VALUE_TYPE_INT
	case csaTypeKeyword:
		return enums.INDEXED_VALUE_TYPE_KEYWORD
	case csaTypeBool:
		return enums.INDEXED_VALUE_TYPE_BOOL
	case csaTypeDouble:
		return enums.INDEXED_VALUE_TYPE_DOUBLE
	case csaTypeText:
		return enums.INDEXED_VALUE_TYPE_TEXT
	case csaTypeDatetime:
		return enums.INDEXED_VALUE_TYPE_DATETIME
	default:
		return enums.INDEXED_VALUE_TYPE_UNSPECIFIED
	}
}

func randomCSAValue(c csaDef, rng *vsRand) any {
	switch c.Type {
	case csaTypeInt:
		return float64(rng.Intn(1001))
	case csaTypeKeyword:
		return rng.vocabWord()
	case csaTypeBool:
		return rng.Intn(2) == 1
	case csaTypeDouble:
		return rng.Float64() * 1000.0
	case csaTypeText:
		// Build the value out of vocabulary words rather than random characters.
		// Text CSAs are analyzed, so the querier's `VS_Text_01 = 'alpha'` becomes a
		// match on the token "alpha"; random character junk tokenizes into terms no
		// filter will ever name, making every Text query a guaranteed zero-hit and
		// removing the class from the measured cost distribution entirely.
		words := make([]string, vsTextWordsMin+rng.Intn(vsTextWordsMax-vsTextWordsMin+1))
		for i := range words {
			words[i] = rng.vocabWord()
		}
		return strings.Join(words, " ")
	case csaTypeDatetime:
		offset := time.Duration(rng.Int63n(int64(30 * 24 * time.Hour)))
		return time.Now().Add(-offset).UTC().Format(time.RFC3339)
	default:
		return nil
	}
}

// buildStartSearchAttributes renders one value for every CSA in the preset as a typed
// search-attribute update, for attachment to StartWorkflowOptions.
//
// The whole preset is covered rather than a random subset, because the point is that every
// document in the corpus is answerable by every CSA filter class the querier can generate.
// Values come from randomCSAValue, the same generator the upsert path uses, so the value
// distributions the querier calibrates selMu against are unchanged.
func buildStartSearchAttributes(defs []csaDef, rng *vsRand) (temporal.SearchAttributes, error) {
	updates := make([]temporal.SearchAttributeUpdate, 0, len(defs))
	for _, d := range defs {
		u, err := startSearchAttribute(d, randomCSAValue(d, rng))
		if err != nil {
			return temporal.SearchAttributes{}, fmt.Errorf("CSA %q: %w", d.Name, err)
		}
		updates = append(updates, u)
	}
	return temporal.NewSearchAttributes(updates...), nil
}

// startSearchAttribute converts one generated CSA value into a typed update.
//
// This switches on csaDef.Type, where the worker-side equivalent has to switch on the name
// prefix: the worker receives values as JSON and has lost the type, while the executor still
// holds the csaDef that produced them. Keeping the executor on the declared type means the
// two sides cannot disagree about a CSA whose name does not follow the VS_<Type>_<N> shape.
func startSearchAttribute(c csaDef, val any) (temporal.SearchAttributeUpdate, error) {
	switch c.Type {
	case csaTypeInt:
		// randomCSAValue yields float64 for Int so that the value survives the JSON round
		// trip on the upsert path unchanged; narrow it here.
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 for Int CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyInt64(c.Name).ValueSet(int64(f)), nil
	case csaTypeKeyword:
		sv, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for Keyword CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyKeyword(c.Name).ValueSet(sv), nil
	case csaTypeBool:
		b, ok := val.(bool)
		if !ok {
			return nil, fmt.Errorf("expected bool for Bool CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyBool(c.Name).ValueSet(b), nil
	case csaTypeDouble:
		f, ok := val.(float64)
		if !ok {
			return nil, fmt.Errorf("expected float64 for Double CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyFloat64(c.Name).ValueSet(f), nil
	case csaTypeText:
		// Text maps to the SDK's String key, matching the worker-side upsert path.
		sv, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string for Text CSA, got %T", val)
		}
		return temporal.NewSearchAttributeKeyString(c.Name).ValueSet(sv), nil
	case csaTypeDatetime:
		sv, ok := val.(string)
		if !ok {
			return nil, fmt.Errorf("expected string (RFC3339) for Datetime CSA, got %T", val)
		}
		t, err := time.Parse(time.RFC3339, sv)
		if err != nil {
			return nil, fmt.Errorf("failed to parse Datetime CSA value %q: %w", sv, err)
		}
		return temporal.NewSearchAttributeKeyTime(c.Name).ValueSet(t), nil
	default:
		return nil, fmt.Errorf("unsupported CSA type %v", c.Type)
	}
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

type vsConfig struct {
	Load             *vsLoadPreset
	Query            *vsQueryPreset
	CSADefs          []csaDef
	NamespaceCount   int
	CreateNamespaces bool
	Retention        time.Duration
	Cleanup          bool
	DeleteNamespaces bool
	// WriterConcurrency is the number of goroutines starting workflows in parallel.
	// See runWriter for why this must exceed 1 to reach the higher presets' WfRPS.
	WriterConcurrency int
	// QueryConcurrency is the number of goroutines issuing visibility queries in
	// parallel. They share one limiter, so CountRPS+ListRPS remains the aggregate
	// target for this executor rather than being multiplied by concurrency.
	QueryConcurrency int
	// DeleteConcurrency is the number of goroutines issuing DeleteWorkflowExecution
	// requests in parallel. They share one limiter, so DeleteRPS remains the aggregate
	// target. Deletion must be parallel because each request is a blocking RPC.
	DeleteConcurrency int
	// WorkflowTimeout overrides WorkflowExecutionTimeout for every workflow the writer
	// starts. Zero means derive it from the CSA update count via vsComputeTimeout.
	//
	// The derived value is a function of how much work the workflow does on the upsert
	// path, which stops being meaningful under CSAAtStart: there the attributes are
	// attached at start and the workflow may do nothing at all, so the derived value
	// collapses to vsComputeTimeout's 5s floor. On a loaded cell, workflow-task dispatch
	// latency exceeds 5s, so every workflow expires before a worker reaches it and the
	// corpus comes out entirely TimedOut. Set this to give workflows a budget that
	// reflects dispatch latency rather than update count.
	WorkflowTimeout time.Duration
	// CSAAtStart attaches one value for every CSA in the preset to StartWorkflowOptions,
	// so the visibility document carries them from the moment it is created.
	//
	// Without this, CSAs reach a document only via UpsertTypedSearchAttributes inside the
	// workflow body, which means a workflow that is never dispatched produces a document
	// with system attributes and nothing else. That is fine for load generation and fatal
	// for corpus building: a corpus whose documents have no CSAs cannot serve any of the
	// querier's CSA filter classes, and selMu's selectivity calibration has nothing to
	// select on. It is off by default because turning it on changes what a start costs.
	CSAAtStart bool
	// VocabZipfSkew is the Zipf exponent for write-side vocabulary draws. Lives on
	// the config rather than the load preset because the querier's vsRand needs a
	// valid value even in read-only mode, where Load is nil.
	VocabZipfSkew float64
}

const (
	// vsRPSPerWriter is the workflow-start throughput we assume a single writer
	// goroutine can sustain, used to derive the default WriterConcurrency. Each start
	// is a blocking RPC, so this is really 1/latency: ~60 rps at a 17ms round trip to
	// Temporal Cloud. We budget conservatively so the target rate stays reachable when
	// latency degrades under load.
	vsRPSPerWriter = 25
	// vsMaxWriterConcurrency caps the derived default so an extreme wfRPS override
	// can't spawn an unbounded number of goroutines. Can be exceeded explicitly via
	// the writerConcurrency option.
	vsMaxWriterConcurrency = 256
	// Deletion is also a blocking RPC. Use the same conservative per-goroutine
	// throughput assumption as workflow starts when deriving delete concurrency.
	vsRPSPerDeleter = 25
	// Bound an accidental extreme override while still leaving ample headroom for the
	// benchmark's ~1.4k delete/s writer namespace.
	vsMaxDeleteConcurrency = 256
	// Temporal caps ListWorkflowExecutions pages at 1,000. A deleter that asks for its
	// entire per-second budget in one page silently receives only this many results.
	vsDeletePageSize = 1000
)

// estimateCorpusSize approximates how many documents the visibility index holds
// midway through a run of the given duration.
//
// Two populations: workflows currently running (a steady-state count set by the
// start rate and the workflow lifetime), and closed workflows the deleter hasn't
// caught up with (accumulating at wfRPS-deleteRPS for the whole run). The corpus
// grows roughly linearly from empty, so the midpoint is what a typical query
// during the run actually sees.
func (p vsLoadPreset) estimateCorpusSize(duration time.Duration) float64 {
	lifetime := p.UpdatesPerWF * p.UpdateDelay.Seconds()
	running := p.WfRPS * lifetime
	netCloseRate := math.Max(0, p.WfRPS-p.DeleteRPS)
	return running + netCloseRate*duration.Seconds()/2
}

// vsMeanPages is the expected page count of a truncated geometric distribution
// with continuation probability p and at most maxPages pages: sum of p^k for
// k in [0, maxPages). Assumes every query has enough results to keep paginating,
// so it is an upper bound on what a run actually issues.
func vsMeanPages(p float64, maxPages int) float64 {
	mean, term := 0.0, 1.0
	for k := 0; k < maxPages; k++ {
		mean += term
		term *= p
	}
	return mean
}

// deriveSelMu picks a median selectivity that makes the median query match about
// vsTargetMedianHits documents.
//
// A fixed selMu is the wrong default because selectivity is a fraction while cost
// is fraction times corpus, and the load presets differ by two orders of magnitude
// in corpus size. Deriving it also tracks wfRPS/deleteRPS/duration overrides, which
// a hardcoded per-preset table cannot.
func deriveSelMu(load vsLoadPreset, duration time.Duration) float64 {
	corpus := load.estimateCorpusSize(duration)
	if corpus <= vsTargetMedianHits {
		// Corpus is smaller than the target result set; nothing to narrow.
		return 0
	}
	return math.Max(vsMinDerivedSelMu, math.Log10(vsTargetMedianHits/corpus))
}

// ---------------------------------------------------------------------------
// Executor
// ---------------------------------------------------------------------------

type visibilityStressExecutor struct {
	config      *vsConfig
	clients     []client.Client
	namespaces  []string
	taskQueue   string
	executionID string
	rng         *vsRand
	rngMu       sync.Mutex

	totalCreated atomic.Int64
	totalDeleted atomic.Int64
	totalQueries atomic.Int64
	totalErrors  atomic.Int64
	wfCounter    atomic.Uint64
	nsCounter    atomic.Uint64
}

var _ loadgen.Configurable = (*visibilityStressExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Visibility store stress test.\n" +
			"Options: loadPreset, queryPreset, csaPreset, csaAtStart, workflowTimeout, namespaceCount, createNamespaces, retention, cleanup,\n" +
			"writerConcurrency (parallel workflow starters; defaults to wfRPS/25),\n" +
			"deleteConcurrency (parallel workflow deleters; defaults to deleteRPS/25),\n" +
			"queryConcurrency (parallel visibility readers sharing the configured aggregate RPS; defaults to 1),\n" +
			"vocabZipfSkew (write-side keyword popularity skew, >1),\n" +
			"selMu/selSigma (log10 selectivity distribution for generated CSA filters),\n" +
			"pageContinueProb/maxQueryPages (geometric List pagination depth).\n" +
			"Duration must be set. At least one of loadPreset or queryPreset required.",
		ExecutorFn: func() loadgen.Executor { return &visibilityStressExecutor{} },
	})
}

// Configure parses and validates all scenario options (presets + overrides).
// Called before Run. In cleanup mode, preset validation is skipped.
func (e *visibilityStressExecutor) Configure(info loadgen.ScenarioInfo) error {
	cfg := &vsConfig{
		NamespaceCount:   info.ScenarioOptionInt("namespaceCount", 1),
		CreateNamespaces: info.ScenarioOptionBool("createNamespaces", false),
		Cleanup:          info.ScenarioOptionBool("cleanup", false),
		DeleteNamespaces: info.ScenarioOptionBool("deleteNamespaces", false),
	}

	cfg.CSAAtStart = info.ScenarioOptionBool("csaAtStart", false)
	cfg.VocabZipfSkew = info.ScenarioOptionFloat("vocabZipfSkew", vsDefaultVocabZipfSkew)
	if cfg.VocabZipfSkew <= 1.0 {
		return fmt.Errorf("vocabZipfSkew must be > 1.0, got %.2f", cfg.VocabZipfSkew)
	}

	if v := info.ScenarioOptions["workflowTimeout"]; v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("invalid workflowTimeout %q: %w", v, err)
		}
		if d <= 0 {
			return fmt.Errorf("workflowTimeout must be positive, got %v", d)
		}
		cfg.WorkflowTimeout = d
	}

	retentionStr := info.ScenarioOptionString("retention", "168h")
	var err error
	cfg.Retention, err = time.ParseDuration(retentionStr)
	if err != nil {
		return fmt.Errorf("invalid retention %q: %w", retentionStr, err)
	}
	if cfg.Retention < 24*time.Hour {
		return fmt.Errorf("retention must be >= 24h, got %v", cfg.Retention)
	}
	if cfg.NamespaceCount < 1 {
		return fmt.Errorf("namespaceCount must be >= 1, got %d", cfg.NamespaceCount)
	}

	if cfg.Cleanup {
		e.config = cfg
		return nil
	}

	if info.Configuration.Duration == 0 && info.Configuration.Iterations == 0 {
		return fmt.Errorf("visibility_stress requires --duration")
	}
	if info.Configuration.Iterations > 0 {
		return fmt.Errorf("visibility_stress does not support --iterations; use --duration")
	}

	// Load preset.
	if presetName, ok := info.ScenarioOptions["loadPreset"]; ok {
		preset, found := vsLoadPresets[presetName]
		if !found {
			return fmt.Errorf("unknown loadPreset %q", presetName)
		}
		if v := info.ScenarioOptions["wfRPS"]; v != "" {
			preset.WfRPS = info.ScenarioOptionFloat("wfRPS", preset.WfRPS)
		}
		if v := info.ScenarioOptions["updatesPerWF"]; v != "" {
			preset.UpdatesPerWF = info.ScenarioOptionFloat("updatesPerWF", preset.UpdatesPerWF)
		}
		if v := info.ScenarioOptions["deleteRPS"]; v != "" {
			preset.DeleteRPS = info.ScenarioOptionFloat("deleteRPS", preset.DeleteRPS)
		}
		if v := info.ScenarioOptions["failPercent"]; v != "" {
			preset.FailPercent = info.ScenarioOptionFloat("failPercent", preset.FailPercent)
		}
		if v := info.ScenarioOptions["timeoutPercent"]; v != "" {
			preset.TimeoutPercent = info.ScenarioOptionFloat("timeoutPercent", preset.TimeoutPercent)
		}
		if preset.FailPercent+preset.TimeoutPercent >= 1.0 {
			return fmt.Errorf("failPercent + timeoutPercent must be < 1.0, got %.2f + %.2f", preset.FailPercent, preset.TimeoutPercent)
		}
		if preset.DeleteRPS < 0 {
			return fmt.Errorf("deleteRPS must be non-negative")
		}
		if preset.WfRPS <= 0 {
			return fmt.Errorf("wfRPS must be positive")
		}
		cfg.Load = &preset

		// Derive writer concurrency from the target rate unless overridden.
		defaultConcurrency := int(math.Ceil(preset.WfRPS / vsRPSPerWriter))
		if defaultConcurrency < 1 {
			defaultConcurrency = 1
		}
		if defaultConcurrency > vsMaxWriterConcurrency {
			defaultConcurrency = vsMaxWriterConcurrency
		}
		cfg.WriterConcurrency = info.ScenarioOptionInt("writerConcurrency", defaultConcurrency)
		if cfg.WriterConcurrency < 1 {
			return fmt.Errorf("writerConcurrency must be >= 1, got %d", cfg.WriterConcurrency)
		}

		defaultDeleteConcurrency := int(math.Ceil(preset.DeleteRPS / vsRPSPerDeleter))
		if defaultDeleteConcurrency < 1 {
			defaultDeleteConcurrency = 1
		}
		if defaultDeleteConcurrency > vsMaxDeleteConcurrency {
			defaultDeleteConcurrency = vsMaxDeleteConcurrency
		}
		cfg.DeleteConcurrency = info.ScenarioOptionInt("deleteConcurrency", defaultDeleteConcurrency)
		if cfg.DeleteConcurrency < 1 {
			return fmt.Errorf("deleteConcurrency must be >= 1, got %d", cfg.DeleteConcurrency)
		}
	}

	// Query preset.
	if presetName, ok := info.ScenarioOptions["queryPreset"]; ok {
		preset, found := vsQueryPresets[presetName]
		if !found {
			return fmt.Errorf("unknown queryPreset %q", presetName)
		}
		if v := info.ScenarioOptions["countRPS"]; v != "" {
			preset.CountRPS = info.ScenarioOptionFloat("countRPS", preset.CountRPS)
		}
		if v := info.ScenarioOptions["listRPS"]; v != "" {
			preset.ListRPS = info.ScenarioOptionFloat("listRPS", preset.ListRPS)
		}
		if v := info.ScenarioOptions["selMu"]; v != "" {
			preset.SelMu = info.ScenarioOptionFloat("selMu", preset.SelMu)
		} else if cfg.Load != nil {
			// Scale the default to the corpus this run will actually build. In
			// read-only mode there is no load preset to derive from, so the caller
			// gets vsDefaultSelMu and should set selMu explicitly to match whatever
			// the prior write run left behind.
			preset.SelMu = deriveSelMu(*cfg.Load, info.Configuration.Duration)
		}
		if v := info.ScenarioOptions["selSigma"]; v != "" {
			preset.SelSigma = info.ScenarioOptionFloat("selSigma", preset.SelSigma)
		}
		if preset.SelMu > 0 {
			return fmt.Errorf("selMu must be <= 0 (log10 of a fraction), got %.2f", preset.SelMu)
		}
		if preset.SelSigma <= 0 {
			return fmt.Errorf("selSigma must be positive, got %.2f", preset.SelSigma)
		}
		if v := info.ScenarioOptions["pageContinueProb"]; v != "" {
			preset.PageContinueProb = info.ScenarioOptionFloat("pageContinueProb", preset.PageContinueProb)
		}
		if v := info.ScenarioOptions["maxQueryPages"]; v != "" {
			preset.MaxPages = info.ScenarioOptionInt("maxQueryPages", preset.MaxPages)
		}
		if preset.PageContinueProb < 0 || preset.PageContinueProb > 1 {
			return fmt.Errorf("pageContinueProb must be in [0,1], got %.2f", preset.PageContinueProb)
		}
		if preset.MaxPages < 1 {
			return fmt.Errorf("maxQueryPages must be >= 1, got %d", preset.MaxPages)
		}
		cfg.QueryConcurrency = info.ScenarioOptionInt("queryConcurrency", 1)
		if cfg.QueryConcurrency < 1 {
			return fmt.Errorf("queryConcurrency must be >= 1, got %d", cfg.QueryConcurrency)
		}
		cfg.Query = &preset
	}

	if cfg.Load == nil && cfg.Query == nil {
		return fmt.Errorf("at least one of loadPreset or queryPreset must be set")
	}

	// CSA preset.
	csaPresetName := info.ScenarioOptionString("csaPreset", "")
	if csaPresetName == "" {
		if cfg.Load != nil {
			csaPresetName = "medium"
		} else {
			return fmt.Errorf("csaPreset is required in read-only mode (no loadPreset)")
		}
	}
	csaNames, found := vsCSAPresets[csaPresetName]
	if !found {
		return fmt.Errorf("unknown csaPreset %q", csaPresetName)
	}
	cfg.CSADefs = make([]csaDef, len(csaNames))
	for i, name := range csaNames {
		cfg.CSADefs[i], err = parseCSAName(name)
		if err != nil {
			return err
		}
	}

	e.config = cfg
	return nil
}

// Run is the main entry point. It configures the executor, sets up namespaces and CSAs,
// then runs the steady-state phase (writer + deleter + querier goroutines) for --duration.
func (e *visibilityStressExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	e.taskQueue = loadgen.TaskQueueForRun(info.RunID)
	e.executionID = info.ExecutionID
	e.rng = newVSRand(time.Now().UnixNano(), e.config.VocabZipfSkew)

	// Resolve namespaces.
	if e.config.NamespaceCount == 1 {
		e.namespaces = []string{info.Namespace}
	} else {
		e.namespaces = make([]string, e.config.NamespaceCount)
		for i := range e.namespaces {
			e.namespaces[i] = fmt.Sprintf("vs-stress-%s-%d", info.RunID, i)
		}
	}

	// Setup namespaces (multi-NS only).
	if e.config.NamespaceCount > 1 {
		if err := e.setupNamespaces(ctx, info); err != nil {
			return err
		}
	}

	// Dial clients. For single-NS, reuse info.Client. For multi-NS, we currently
	// only support single-NS properly (multi-NS dialing needs connection params
	// exposed via ScenarioInfo).
	// TODO: Support multi-namespace client dialing by exposing ClientOptions in ScenarioInfo.
	e.clients = make([]client.Client, len(e.namespaces))
	for i := range e.namespaces {
		e.clients[i] = info.Client
	}

	if e.config.Cleanup {
		return e.runCleanup(ctx, info)
	}

	if !info.Configuration.DoNotRegisterSearchAttributes {
		if err := e.registerCSAs(ctx, info); err != nil {
			return err
		}
	}

	// Wait for at least one worker to be polling each namespace's task queue
	// before starting the steady-state phase. This lets users start the scenario
	// before the workers (e.g., for multi-namespace where the scenario creates
	// namespaces that workers need).
	if e.config.Load != nil {
		if err := e.waitForWorkers(ctx, info); err != nil {
			return err
		}
	}

	e.logConfig(info)
	return e.runSteadyState(ctx, info)
}

// ---------------------------------------------------------------------------
// Setup
// ---------------------------------------------------------------------------

func (e *visibilityStressExecutor) setupNamespaces(ctx context.Context, info loadgen.ScenarioInfo) error {
	if !e.config.CreateNamespaces {
		for _, ns := range e.namespaces {
			_, err := info.Client.WorkflowService().DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
				Namespace: ns,
			})
			if err != nil {
				return fmt.Errorf("namespace %s does not exist (pass createNamespaces=true to create): %w", ns, err)
			}
		}
		return nil
	}

	for _, ns := range e.namespaces {
		_, err := info.Client.WorkflowService().RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
			Namespace:                        ns,
			WorkflowExecutionRetentionPeriod: durationpb.New(e.config.Retention),
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create namespace %s: %w", ns, err)
		}
		info.Logger.Infof("Namespace %s ready", ns)
	}
	return nil
}

// registerCSAs registers all CSAs from the csaPreset on every namespace, then polls
// ListSearchAttributes until they're visible (propagation can take time on Elasticsearch).
// "Already exists" errors are treated as success (idempotent).
func (e *visibilityStressExecutor) registerCSAs(ctx context.Context, info loadgen.ScenarioInfo) error {
	saMap := make(map[string]enums.IndexedValueType, len(e.config.CSADefs))
	for _, csa := range e.config.CSADefs {
		saMap[csa.Name] = csa.indexedValueType()
	}

	for i, ns := range e.namespaces {
		_, err := e.clients[i].OperatorService().AddSearchAttributes(ctx, &operatorservice.AddSearchAttributesRequest{
			SearchAttributes: saMap,
			Namespace:        ns,
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to register CSAs on namespace %s: %w", ns, err)
		}
	}

	// Poll until CSAs are visible.
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		allReady := true
		for i, ns := range e.namespaces {
			resp, err := e.clients[i].OperatorService().ListSearchAttributes(ctx, &operatorservice.ListSearchAttributesRequest{
				Namespace: ns,
			})
			if err != nil {
				allReady = false
				break
			}
			for _, csa := range e.config.CSADefs {
				if _, ok := resp.CustomAttributes[csa.Name]; !ok {
					allReady = false
					break
				}
			}
			if !allReady {
				break
			}
		}
		if allReady {
			info.Logger.Infof("All %d CSAs propagated on %d namespace(s)", len(e.config.CSADefs), len(e.namespaces))
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("CSAs did not propagate within 30s")
}

// waitForWorkers polls DescribeTaskQueue on each namespace until at least one workflow
// poller is detected. This allows the scenario to be started before workers (useful for
// multi-namespace where the scenario creates namespaces that workers need to connect to).
// Times out after 2 minutes.
func (e *visibilityStressExecutor) waitForWorkers(ctx context.Context, info loadgen.ScenarioInfo) error {
	deadline := time.Now().Add(2 * time.Minute)
	info.Logger.Infof("Waiting for workers to start polling task queue %s on %d namespace(s)...", e.taskQueue, len(e.namespaces))

	for time.Now().Before(deadline) {
		allReady := true
		for i, ns := range e.namespaces {
			resp, err := e.clients[i].DescribeTaskQueue(ctx, e.taskQueue, enums.TASK_QUEUE_TYPE_WORKFLOW)
			if err != nil {
				info.Logger.Debugf("DescribeTaskQueue failed for %s: %v", ns, err)
				allReady = false
				break
			}
			if len(resp.Pollers) == 0 {
				allReady = false
				break
			}
		}
		if allReady {
			info.Logger.Infof("Workers detected on all %d namespace(s)", len(e.namespaces))
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for workers")
		case <-time.After(1 * time.Second):
		}
	}
	return fmt.Errorf("timed out waiting for workers after 2m; ensure workers are running with --task-queue %s on each namespace", e.taskQueue)
}

func (e *visibilityStressExecutor) logConfig(info loadgen.ScenarioInfo) {
	var mode string
	switch {
	case e.config.Load != nil && e.config.Query != nil:
		mode = "write+read"
	case e.config.Load != nil:
		mode = "write-only"
	default:
		mode = "read-only"
	}
	info.Logger.Infof("Mode: %s", mode)
	info.Logger.Infof("Namespaces: %v", e.namespaces)
	info.Logger.Infof("Task queue: %s", e.taskQueue)
	info.Logger.Infof("CSAs: %d total, csaAtStart=%v", len(e.config.CSADefs), e.config.CSAAtStart)

	if e.config.WorkflowTimeout > 0 {
		info.Logger.Infof("Workflow timeout: %v (explicit)", e.config.WorkflowTimeout)
		// The workflow sleeps UpdateDelay between consecutive update groups, so a budget
		// below that guarantees workflows that DO get dispatched still time out mid-run.
		if l := e.config.Load; l != nil && l.UpdatesPerWF > 1 {
			need := time.Duration((l.UpdatesPerWF - 1) * float64(l.UpdateDelay))
			if e.config.WorkflowTimeout < need {
				info.Logger.Warnf(
					"workflowTimeout=%v is below the %v of sleeps implied by updatesPerWF=%.1f and updateDelay=%v; dispatched workflows will time out mid-run",
					e.config.WorkflowTimeout, need, l.UpdatesPerWF, l.UpdateDelay)
			}
		}
	} else {
		info.Logger.Infof("Workflow timeout: derived from updatesPerWF (min 5s)")
	}

	if l := e.config.Load; l != nil {
		info.Logger.Infof("Write: wfRPS=%.1f, updatesPerWF=%.1f, effective CSA update RPS≈%.0f, deleteRPS=%.1f",
			l.WfRPS, l.UpdatesPerWF, l.WfRPS*l.UpdatesPerWF, l.DeleteRPS)
		info.Logger.Infof("       writerConcurrency=%d, deleteConcurrency=%d (max achievable rps ≈ concurrency/rpc_latency)",
			e.config.WriterConcurrency, e.config.DeleteConcurrency)
		info.Logger.Infof("       failPercent=%.2f, timeoutPercent=%.2f, updateDelay=%v",
			l.FailPercent, l.TimeoutPercent, l.UpdateDelay)
	}
	if q := e.config.Query; q != nil {
		info.Logger.Infof("Read: countRPS=%.1f, listRPS=%.1f, queryConcurrency=%d",
			q.CountRPS, q.ListRPS, e.config.QueryConcurrency)
		info.Logger.Infof("      filter weights: noFilter=%d open=%d closed=%d simpleCSA=%d compoundCSA=%d disjunction=%d",
			q.ListNoFilterWeight, q.ListOpenWeight, q.ListClosedWeight,
			q.ListSimpleCSAWeight, q.ListCompoundCSAWeight, q.ListDisjunctionWeight)
		info.Logger.Infof("      selectivity: log10(s)~Normal(mu=%.2f, sigma=%.2f), median≈%.4f%% of corpus",
			q.SelMu, q.SelSigma, 100*math.Pow(10, q.SelMu))
		if l := e.config.Load; l != nil {
			corpus := l.estimateCorpusSize(info.Configuration.Duration)
			info.Logger.Infof("      est. corpus at run midpoint≈%.0f docs, median query≈%.0f hits",
				corpus, corpus*math.Pow(10, q.SelMu))
		}
		info.Logger.Infof("      pagination: continueProb=%.2f, maxPages=%d, mean pages≈%.2f",
			q.PageContinueProb, q.MaxPages, vsMeanPages(q.PageContinueProb, q.MaxPages))
	}
	info.Logger.Infof("Vocabulary: %d words, write-side Zipf skew=%.2f (reads draw uniformly)",
		len(keywordVocabulary), e.config.VocabZipfSkew)
}

// ---------------------------------------------------------------------------
// Steady-State
// ---------------------------------------------------------------------------

func (e *visibilityStressExecutor) runSteadyState(ctx context.Context, info loadgen.ScenarioInfo) error {
	ctx, cancel := context.WithTimeout(ctx, info.Configuration.Duration)
	defer cancel()

	// Writer goroutine.
	if e.config.Load != nil {
		go e.runWriter(ctx, info)
	}

	// Deleter goroutines.
	if e.config.Load != nil && e.config.Load.DeleteRPS > 0 {
		go e.runDeleters(ctx, info)
	}

	// Querier goroutine.
	if e.config.Query != nil {
		go e.runQuerier(ctx, info)
	}

	<-ctx.Done()

	// Wait for in-flight workflows to complete before returning. This is a pragmatic
	// workaround: run-scenario-with-worker kills the worker as soon as Run() returns,
	// but workflows started near the end of --duration haven't finished yet. Sleeping
	// here delays Run()'s return, keeping the worker alive for the drain period.
	// The writer/deleter/querier goroutines have already exited (their ctx is cancelled).
	if e.config.Load != nil {
		drainTime := time.Duration(math.Ceil(e.config.Load.UpdatesPerWF)) * e.config.Load.UpdateDelay
		drainTime += 5 * time.Second // buffer for scheduling delays
		info.Logger.Infof("Draining: waiting %v for in-flight workflows to complete...", drainTime)
		time.Sleep(drainTime)
	}

	info.Logger.Infof("Run complete. Created: %d, Deleted: %d, Queries: %d, Errors: %d",
		e.totalCreated.Load(), e.totalDeleted.Load(), e.totalQueries.Load(), e.totalErrors.Load())
	return nil
}

// runWriter starts workflows at the configured wfRPS rate. Each workflow is fire-and-forget:
// we don't wait for completion. The workflow input encodes all CSA update instructions,
// failure/timeout behavior, and delay between updates.
// runWriter fans out WriterConcurrency goroutines that share a single rate limiter,
// so wfRPS is an achievable target rather than an upper bound. Each start is a blocking
// StartWorkflowExecution RPC, so one goroutine can only reach 1/latency — roughly 60 rps
// against Temporal Cloud, which left the higher presets an order of magnitude short of
// their nominal wfRPS. The limiter is shared, so adding goroutines raises the ceiling
// without raising the rate beyond what was configured.
func (e *visibilityStressExecutor) runWriter(ctx context.Context, info loadgen.ScenarioInfo) {
	limiter := rate.NewLimiter(rate.Limit(e.config.Load.WfRPS), 1)
	startTime := time.Now()

	var wg sync.WaitGroup
	for w := 0; w < e.config.WriterConcurrency; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			e.runWriterLoop(ctx, info, limiter, startTime, w)
		}(w)
	}
	wg.Wait()
}

// runWriterLoop is a single writer goroutine. All shared state it touches is atomic:
// the rate limiter, the workflow/namespace counters, and the totals.
func (e *visibilityStressExecutor) runWriterLoop(
	ctx context.Context, info loadgen.ScenarioInfo,
	limiter *rate.Limiter, startTime time.Time, workerIdx int,
) {
	// Each goroutine needs its own vsRand: math/rand.Rand is not safe for concurrent
	// use, and e.rng is owned by the querier goroutine. Offset the seed by workerIdx so
	// the writers don't all generate the same CSA update sequence.
	rng := newVSRand(time.Now().UnixNano()+int64(workerIdx), e.config.VocabZipfSkew)

	logEvery := int64(math.Max(1, e.config.Load.WfRPS))

	for {
		if err := limiter.Wait(ctx); err != nil {
			return
		}

		nsIdx := int((e.nsCounter.Add(1) - 1) % uint64(len(e.namespaces)))

		input := e.buildWorkflowInput(rng)
		wfID := fmt.Sprintf("%s%d", visibilityStressWorkflowIDPrefix(info.RunID, e.executionID), e.wfCounter.Add(1))

		opts := client.StartWorkflowOptions{
			ID:                       wfID,
			TaskQueue:                e.taskQueue,
			WorkflowExecutionTimeout: e.workflowTimeout(len(input.CSAUpdates)),
		}

		if e.config.CSAAtStart {
			sa, err := buildStartSearchAttributes(e.config.CSADefs, rng)
			if err != nil {
				// A malformed CSA definition is a configuration fault, not a transient
				// one, so it would recur on every iteration; count it and move on rather
				// than spinning silently.
				e.totalErrors.Add(1)
				info.Logger.Warnf("Failed to build start search attributes: %v", err)
				continue
			}
			opts.TypedSearchAttributes = sa
		}

		// Fire and forget.
		// TODO: better error handling (retry? circuit breaker?)
		_, err := e.clients[nsIdx].ExecuteWorkflow(ctx, opts, "visibilityStressWorker", input)
		if err != nil {
			e.totalErrors.Add(1)
			info.Logger.Warnf("Failed to start workflow: %v", err)
			continue
		}

		// Log off the shared total so the cadence stays ~1/sec regardless of how many
		// writers are running; whichever goroutine lands on the boundary emits the line.
		if created := e.totalCreated.Add(1); created%logEvery == 0 {
			elapsed := time.Since(startTime)
			info.Logger.Infof("[writer] t=%v created=%d errors=%d actual_rps=%.1f",
				elapsed.Round(time.Second), created,
				e.totalErrors.Load(), float64(created)/elapsed.Seconds())
		}
	}
}

func visibilityStressWorkflowIDPrefix(runID, executionID string) string {
	return fmt.Sprintf("vs-%s-%s-", runID, executionID)
}

func visibilityQueryString(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func visibilityStressDeleterQuery(taskQueue, runID, executionID string) string {
	return fmt.Sprintf(
		"WorkflowType = 'visibilityStressWorker' AND ExecutionStatus != 'Running' AND TaskQueue = %s AND WorkflowId STARTS_WITH %s",
		visibilityQueryString(taskQueue),
		visibilityQueryString(visibilityStressWorkflowIDPrefix(runID, executionID)),
	)
}

func (e *visibilityStressExecutor) runDeleters(ctx context.Context, info loadgen.ScenarioInfo) {
	perNsDeleteRPS := e.config.Load.DeleteRPS / float64(len(e.namespaces))
	for i, ns := range e.namespaces {
		i, ns := i, ns
		go e.runDeleterForNamespace(ctx, info, i, ns, perNsDeleteRPS)
	}
}

type vsDeleteCandidate struct {
	execution  *commonpb.WorkflowExecution
	key        string
	generation uint64
}

type vsDeleteResult struct {
	candidate vsDeleteCandidate
	err       error
}

type vsRecentDeleteSet struct {
	mu         sync.Mutex
	limit      int
	seen       map[string]uint64
	order      []vsDeleteCandidate
	head       int
	generation uint64
}

func newVSRecentDeleteSet(limit int) *vsRecentDeleteSet {
	return &vsRecentDeleteSet{
		limit: limit,
		seen:  make(map[string]uint64, limit),
		order: make([]vsDeleteCandidate, 0, limit),
	}
}

func (s *vsRecentDeleteSet) add(execution *commonpb.WorkflowExecution) (vsDeleteCandidate, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := execution.WorkflowId + "\x00" + execution.RunId
	if _, ok := s.seen[key]; ok {
		return vsDeleteCandidate{}, false
	}

	s.generation++
	candidate := vsDeleteCandidate{execution: execution, key: key, generation: s.generation}
	s.seen[key] = candidate.generation
	s.order = append(s.order, candidate)

	for len(s.order)-s.head > s.limit {
		old := s.order[s.head]
		s.head++
		if s.seen[old.key] == old.generation {
			delete(s.seen, old.key)
		}
	}
	if s.head > s.limit && s.head*2 > len(s.order) {
		s.order = append([]vsDeleteCandidate(nil), s.order[s.head:]...)
		s.head = 0
	}
	return candidate, true
}

func (s *vsRecentDeleteSet) remove(candidate vsDeleteCandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.seen[candidate.key] == candidate.generation {
		delete(s.seen, candidate.key)
	}
}

// runDeleteWorkers consumes a continuous stream of candidates. The shared limiter controls
// aggregate throughput; concurrency hides blocking delete-RPC latency and therefore does not
// multiply DeleteRPS. The scanner runs independently, so its List latency is also overlapped.
func runDeleteWorkers(
	ctx context.Context,
	jobs <-chan vsDeleteCandidate,
	results chan<- vsDeleteResult,
	limiter *rate.Limiter,
	concurrency int,
	deleteFn func(context.Context, *commonpb.WorkflowExecution) error,
) {
	var wg sync.WaitGroup
	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for candidate := range jobs {
				if err := limiter.Wait(ctx); err != nil {
					select {
					case results <- vsDeleteResult{candidate: candidate, err: err}:
					case <-ctx.Done():
					}
					continue
				}
				result := vsDeleteResult{
					candidate: candidate,
					err:       deleteFn(ctx, candidate.execution),
				}
				select {
				case results <- result:
				case <-ctx.Done():
					return
				}
			}
		}()
	}
	wg.Wait()
}

// runDeleterForNamespace continuously pages through terminal workflows (via the visibility
// store) and deletes them at the configured aggregate rate.
//
// A single serial loop tops out at roughly 1/delete-RPC-latency (about 125/s in the test
// cells), far below the benchmark's ~1.4k/s writer namespace. This implementation uses one
// scanner feeding a bounded channel and a persistent delete-worker pool. The recent-ID set
// prevents a scan from
// repeatedly submitting the same successfully deleted workflows while the visibility index
// waits for its next refresh.
// The query is scoped by both TaskQueue and this executor invocation's workflow-ID prefix.
// TaskQueue alone is not sufficient when the same run ID is reused: a restarted executor would
// otherwise rediscover visibility records whose asynchronous delete tasks were submitted by an
// earlier invocation. Those duplicate DeleteWorkflowExecution calls amplify transfer-queue work
// without producing new visibility deletes.
func (e *visibilityStressExecutor) runDeleterForNamespace(
	ctx context.Context, info loadgen.ScenarioInfo,
	nsIdx int, ns string, deleteRPS float64,
) {
	if deleteRPS <= 0 {
		return
	}

	limiter := rate.NewLimiter(rate.Limit(deleteRPS), 1)
	// Keep enough IDs to span a minute at the target rate. That comfortably exceeds the
	// benchmark index refresh interval without growing for the lifetime of a long run.
	seenLimit := int(math.Max(vsDeletePageSize, math.Ceil(deleteRPS*60)))
	recent := newVSRecentDeleteSet(seenLimit)
	var nextPageToken []byte
	endOfScanPause := time.Second
	if pageFillTime := time.Duration(float64(time.Second) * vsDeletePageSize / deleteRPS); pageFillTime < endOfScanPause {
		endOfScanPause = pageFillTime
	}
	if endOfScanPause < 50*time.Millisecond {
		endOfScanPause = 50 * time.Millisecond
	}

	query := visibilityStressDeleterQuery(e.taskQueue, info.RunID, e.executionID)
	jobs := make(chan vsDeleteCandidate, 2*vsDeletePageSize)
	results := make(chan vsDeleteResult, 2*vsDeletePageSize)
	workersDone := make(chan struct{})
	go func() {
		runDeleteWorkers(ctx, jobs, results, limiter, e.config.DeleteConcurrency,
			func(deleteCtx context.Context, wfExec *commonpb.WorkflowExecution) error {
				_, err := e.clients[nsIdx].WorkflowService().DeleteWorkflowExecution(deleteCtx,
					&workflowservice.DeleteWorkflowExecutionRequest{
						Namespace: ns,
						WorkflowExecution: &commonpb.WorkflowExecution{
							WorkflowId: wfExec.WorkflowId,
							RunId:      wfExec.RunId,
						},
					})
				return err
			})
		close(results)
		close(workersDone)
	}()

	var submitted atomic.Int64
	var deleteErrors atomic.Int64
	resultsDone := make(chan struct{})
	go func() {
		defer close(resultsDone)
		for result := range results {
			if result.err != nil {
				recent.remove(result.candidate)
				if ctx.Err() == nil {
					deleteErrors.Add(1)
					info.Logger.Warnf("[deleter/%s] Delete %s failed: %v",
						ns, result.candidate.execution.WorkflowId, result.err)
				}
				continue
			}
			if deleted := e.totalDeleted.Add(1); deleted%vsDeletePageSize == 0 {
				info.Logger.Infof("[deleter/%s] deleted=%d errors=%d submitted=%d queued=%d",
					ns, deleted, deleteErrors.Load(), submitted.Load(), len(jobs))
			}
		}
	}()

	defer func() {
		close(jobs)
		<-workersDone
		<-resultsDone
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		// The deleter's own List is a visibility read on the same search path as the
		// querier's, so it is metered too. Left unmeasured it is invisible background
		// load that scales with deleteRPS and skews the querier's latencies.
		start := time.Now()
		resp, err := e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace:     ns,
			Query:         query,
			PageSize:      vsDeletePageSize,
			NextPageToken: nextPageToken,
		})
		e.recordQuery(info, "list", vsClassDeleterList, 1, time.Since(start), err)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			info.Logger.Warnf("[deleter/%s] List failed: %v", ns, err)
			nextPageToken = nil
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
			continue
		}

		submittedThisPage := 0
		for _, exec := range resp.Executions {
			candidate, ok := recent.add(exec.Execution)
			if !ok {
				continue
			}
			select {
			case jobs <- candidate:
				submitted.Add(1)
				submittedThisPage++
			case <-ctx.Done():
				recent.remove(candidate)
				return
			}
		}

		nextPageToken = resp.NextPageToken
		if len(nextPageToken) == 0 || submittedThisPage == 0 {
			// At the end of a scan (or on a page containing only recently submitted IDs),
			// yield long enough for one page of new work to accumulate. This keeps the
			// deleter's List traffic near the minimum required by its delete target.
			select {
			case <-ctx.Done():
				return
			case <-time.After(endOfScanPause):
			}
		}
	}
}

const (
	vsQueryLatencyMetric = "omes_visibility_query_latency"
	vsQueryErrorMetric   = "omes_visibility_query_errors"
)

// shouldFetchNextPage decides whether to read one more page of a List result,
// having just read page n.
//
// Real clients stop scrolling at an unpredictable point, so the page count is a
// random variable rather than a constant. Each additional page is taken with
// probability PageContinueProb, making the count geometric and truncated at
// MaxPages. A fixed cap instead puts a hard edge on the cost distribution right
// where the deep-pagination tail would be.
func (e *visibilityStressExecutor) shouldFetchNextPage(page int) bool {
	q := e.config.Query
	if page >= q.MaxPages {
		return false
	}
	return e.rng.Float64() < q.PageContinueProb
}

// recordQuery emits latency and error metrics for a single visibility read.
//
// Tagging by class matters: aggregate percentiles over the whole query mix are
// dominated by whichever class carries the most weight, so they shift when the
// queryPreset weights change and runs stop being comparable. Per-class series
// plus the known weights let any aggregate be reconstructed afterwards.
func (e *visibilityStressExecutor) recordQuery(
	info loadgen.ScenarioInfo,
	operation, class string,
	page int,
	elapsed time.Duration,
	err error,
) {
	if info.MetricsHandler == nil {
		return
	}
	handler := info.MetricsHandler.WithTags(map[string]string{
		"operation":   operation,
		"query_class": class,
		"page":        strconv.Itoa(page),
	})
	handler.Timer(vsQueryLatencyMetric).Record(elapsed)
	if err != nil {
		handler.Counter(vsQueryErrorMetric).Inc(1)
	}
}

// runQuerier fans out QueryConcurrency workers that share one rate limiter, so
// CountRPS+ListRPS is the aggregate target for this executor. Query type is
// chosen by weighted random from the queryPreset distribution. List queries
// fetch a geometrically distributed number of pages to exercise search_after
// pagination; see shouldFetchNextPage.
func (e *visibilityStressExecutor) runQuerier(ctx context.Context, info loadgen.ScenarioInfo) {
	q := e.config.Query
	totalRPS := q.CountRPS + q.ListRPS
	if totalRPS <= 0 {
		return
	}

	startTime := time.Now()
	limiter := rate.NewLimiter(rate.Limit(totalRPS), 1)
	var nsIndex atomic.Uint64
	var queryErrors atomic.Int64
	var lastLogNanos atomic.Int64
	lastLogNanos.Store(startTime.UnixNano())

	var wg sync.WaitGroup
	for workerIdx := 0; workerIdx < e.config.QueryConcurrency; workerIdx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.runQueryLoop(ctx, info, limiter, startTime, &nsIndex, &queryErrors, &lastLogNanos)
		}()
	}
	wg.Wait()
}

// runQueryLoop is one synchronous visibility reader. The limiter, namespace
// counter, totals, and log timestamp are safe to share. Filter generation and
// pagination decisions briefly hold rngMu because math/rand.Rand is not safe for
// concurrent use; no network call is made while holding that lock.
func (e *visibilityStressExecutor) runQueryLoop(
	ctx context.Context,
	info loadgen.ScenarioInfo,
	limiter *rate.Limiter,
	startTime time.Time,
	nsIndex *atomic.Uint64,
	queryErrors *atomic.Int64,
	lastLogNanos *atomic.Int64,
) {
	q := e.config.Query
	totalRPS := q.CountRPS + q.ListRPS

	totalWeight := q.ListNoFilterWeight + q.ListOpenWeight + q.ListClosedWeight +
		q.ListSimpleCSAWeight + q.ListCompoundCSAWeight + q.ListDisjunctionWeight

	// Log every ~1 second worth of queries, with a minimum of every 1s.
	logInterval := int64(math.Max(1, totalRPS))

	for {
		if err := limiter.Wait(ctx); err != nil {
			return
		}

		nsIdx := int((nsIndex.Add(1) - 1) % uint64(len(e.namespaces)))
		ns := e.namespaces[nsIdx]

		e.rngMu.Lock()
		isCount := e.rng.Float64() < q.CountRPS/totalRPS
		filter, class := e.generateFilter(isCount, totalWeight)
		e.rngMu.Unlock()

		var queryType string
		if isCount {
			queryType = "count"
			start := time.Now()
			_, err := e.clients[nsIdx].CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
				Namespace: ns,
				Query:     filter,
			})
			e.recordQuery(info, queryType, class, 1, time.Since(start), err)
			if err != nil {
				// TODO: better error handling
				queryErrors.Add(1)
				info.Logger.Warnf("[querier] Count failed (filter=%s): %v", filter, err)
			}
		} else {
			queryType = "list"
			// Each page is timed separately: page 1 and a deep page are different
			// cost points, and averaging them hides that.
			var pageToken []byte
			for page := 1; ; page++ {
				start := time.Now()
				resp, err := e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
					Namespace:     ns,
					Query:         filter,
					NextPageToken: pageToken,
				})
				e.recordQuery(info, queryType, class, page, time.Since(start), err)
				if err != nil {
					// TODO: better error handling
					queryErrors.Add(1)
					info.Logger.Warnf("[querier] List failed (filter=%s): %v", filter, err)
					break
				}
				if len(resp.NextPageToken) == 0 {
					break
				}
				e.rngMu.Lock()
				fetchNext := e.shouldFetchNextPage(page)
				e.rngMu.Unlock()
				if !fetchNext {
					break
				}
				pageToken = resp.NextPageToken
			}
		}

		total := e.totalQueries.Add(1)

		// Periodic logging. The CAS keeps a slow workload from producing one
		// fallback log line per reader every five seconds.
		now := time.Now()
		shouldLog := total%logInterval == 0
		if !shouldLog {
			last := lastLogNanos.Load()
			if now.UnixNano()-last >= int64(5*time.Second) {
				shouldLog = lastLogNanos.CompareAndSwap(last, now.UnixNano())
			}
		} else {
			lastLogNanos.Store(now.UnixNano())
		}
		if shouldLog {
			elapsed := now.Sub(startTime)
			info.Logger.Infof("[querier] t=%v queries=%d errors=%d actual_rps=%.1f last=%s class=%s filter=%s",
				elapsed.Round(time.Second), total, queryErrors.Load(),
				float64(total)/elapsed.Seconds(), queryType, class, filter)
		}
	}
}

// ---------------------------------------------------------------------------
// Query Generation
// ---------------------------------------------------------------------------

// Query classes, used both to pick a filter shape and to tag the latency metrics
// so each class's cost distribution can be fitted separately.
const (
	vsClassNoFilter    = "no_filter"
	vsClassOpen        = "open"
	vsClassClosed      = "closed_time_range"
	vsClassSimpleCSA   = "simple_csa"
	vsClassCompoundCSA = "compound_csa"
	vsClassDisjunction = "disjunction_csa"
	vsClassKeyword     = "keyword"
	vsClassDeleterList = "deleter_list"
)

// generateFilter returns a visibility query and the class it belongs to.
func (e *visibilityStressExecutor) generateFilter(isCount bool, totalWeight int) (string, string) {
	if isCount {
		switch e.rng.Intn(3) {
		case 0:
			return "WorkflowType = 'visibilityStressWorker'", vsClassNoFilter
		case 1:
			return "ExecutionStatus = 'Running'", vsClassOpen
		default:
			kwCSAs := e.csasByType(csaTypeKeyword)
			if len(kwCSAs) > 0 {
				csa := kwCSAs[e.rng.Intn(len(kwCSAs))]
				val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
				return fmt.Sprintf("%s = '%s'", csa.Name, val), vsClassKeyword
			}
			return "WorkflowType = 'visibilityStressWorker'", vsClassNoFilter
		}
	}

	q := e.config.Query
	roll := e.rng.Intn(totalWeight)
	cumulative := 0

	cumulative += q.ListNoFilterWeight
	if roll < cumulative {
		return "WorkflowType = 'visibilityStressWorker'", vsClassNoFilter
	}

	cumulative += q.ListOpenWeight
	if roll < cumulative {
		return "WorkflowType = 'visibilityStressWorker' AND ExecutionStatus = 'Running'", vsClassOpen
	}

	cumulative += q.ListClosedWeight
	if roll < cumulative {
		since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
		return fmt.Sprintf("ExecutionStatus != 'Running' AND CloseTime > '%s'", since), vsClassClosed
	}

	cumulative += q.ListSimpleCSAWeight
	if roll < cumulative {
		return e.generateSimpleCSAFilter(), vsClassSimpleCSA
	}

	cumulative += q.ListCompoundCSAWeight
	if roll < cumulative {
		return e.generateCompoundCSAFilter(), vsClassCompoundCSA
	}

	return e.generateDisjunctionFilter(), vsClassDisjunction
}

// vsClauseMuFn maps the query-level selectivity target onto a per-clause target,
// given how many of the clauses can actually be tuned by a threshold.
//
// Without this the target applies to every clause independently, and a multi-clause
// filter overshoots badly: three ANDed clauses at 10^-2.5 each combine to 10^-7.5,
// which matches nothing on any corpus this scenario builds. That would make the
// compound class degenerate in the same way the Text class used to be.
type vsClauseMuFn func(queryMu float64, tunableClauses int) float64

// vsConjunctionMu spreads the target across ANDed clauses: selectivities multiply,
// so their log10s add and each clause takes an equal share.
func vsConjunctionMu(queryMu float64, tunableClauses int) float64 {
	return queryMu / float64(tunableClauses)
}

// vsDisjunctionMu tightens each ORed clause: selectivities roughly sum, so n
// clauses at s together match about n*s.
func vsDisjunctionMu(queryMu float64, tunableClauses int) float64 {
	return queryMu - math.Log10(float64(tunableClauses))
}

func (e *visibilityStressExecutor) generateSimpleCSAFilter() string {
	csa := e.config.CSADefs[e.rng.Intn(len(e.config.CSADefs))]
	return e.csaFilterClause(csa, e.config.Query.SelMu)
}

func (e *visibilityStressExecutor) generateCompoundCSAFilter() string {
	return strings.Join(e.distinctCSAClauses(2, 3, vsConjunctionMu), " AND ")
}

// generateDisjunctionFilter ORs several CSA clauses together.
//
// This is the expensive shape, and the one the AND-based compound class does not
// cover. A conjunction costs roughly its rarest clause, because the engine leads
// with the smallest posting list and skips — so adding terms to an AND generally
// makes a query cheaper. A disjunction has to evaluate and union every clause,
// so cost grows with the sum of the clauses' selectivities rather than the min.
func (e *visibilityStressExecutor) generateDisjunctionFilter() string {
	clauses := e.distinctCSAClauses(2, 3, vsDisjunctionMu)
	if len(clauses) < 2 {
		// A one-clause "disjunction" is just a simple filter; don't wrap it.
		return strings.Join(clauses, "")
	}
	return "(" + strings.Join(clauses, " OR ") + ")"
}

// distinctCSAClauses builds between minClauses and maxClauses filter clauses over
// distinct CSAs, so a compound filter never contradicts itself by constraining
// the same attribute twice. muFn redistributes the query-level selectivity target
// across the clauses according to how they will be combined.
func (e *visibilityStressExecutor) distinctCSAClauses(
	minClauses, maxClauses int, muFn vsClauseMuFn,
) []string {
	n := minClauses + e.rng.Intn(maxClauses-minClauses+1)
	if n > len(e.config.CSADefs) {
		n = len(e.config.CSADefs)
	}
	perm := e.rng.Perm(len(e.config.CSADefs))
	chosen := make([]csaDef, n)
	for i := range chosen {
		chosen[i] = e.config.CSADefs[perm[i]]
	}

	// Only threshold clauses honor selMu, so the target is split across those
	// rather than across every clause. Dividing by n when two of three clauses are
	// Keyword or Bool would make the one tunable clause far narrower than intended.
	tunable := 0
	for _, csa := range chosen {
		if csa.usesSelectivityThreshold() {
			tunable++
		}
	}
	if tunable == 0 {
		tunable = 1
	}
	clauseMu := muFn(e.config.Query.SelMu, tunable)

	clauses := make([]string, n)
	for i, csa := range chosen {
		clauses[i] = e.csaFilterClause(csa, clauseMu)
	}
	return clauses
}

// csaFilterClause builds a predicate over one CSA targeting a log-normally
// distributed selectivity. Because the write-side value distributions are known
// and uniform (see randomCSAValue), the target fraction inverts directly into a
// threshold.
//
// Keyword and Text clauses are the exception: their selectivity is set by the
// Zipf popularity of the drawn word, not by a threshold. The read side draws
// uniformly on purpose, so which word it lands on is what varies the cost.
func (e *visibilityStressExecutor) csaFilterClause(csa csaDef, selMu float64) string {
	s := e.rng.sampleSelectivity(selMu, e.config.Query.SelSigma)

	switch csa.Type {
	case csaTypeInt:
		// Values are uniform over [0,1000], so P(X > t) = (1000-t)/1000.
		if e.rng.Intn(2) == 0 {
			return fmt.Sprintf("%s > %d", csa.Name, int(math.Round(1000*(1-s))))
		}
		width := math.Max(1, 1000*s)
		lo := e.rng.Float64() * (1000 - width)
		return fmt.Sprintf("%s > %d AND %s < %d",
			csa.Name, int(lo), csa.Name, int(math.Ceil(lo+width)))
	case csaTypeKeyword:
		val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
		return fmt.Sprintf("%s = '%s'", csa.Name, val)
	case csaTypeBool:
		// Booleans have exactly two values; selectivity is fixed at ~0.5.
		if e.rng.Intn(2) == 0 {
			return fmt.Sprintf("%s = true", csa.Name)
		}
		return fmt.Sprintf("%s = false", csa.Name)
	case csaTypeDouble:
		return fmt.Sprintf("%s > %f", csa.Name, 1000*(1-s))
	case csaTypeText:
		val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
		return fmt.Sprintf("%s = '%s'", csa.Name, val)
	case csaTypeDatetime:
		// Values are now minus a uniform offset over the last 30 days, so a lower
		// bound s of the way back matches a fraction s of them.
		t := time.Now().Add(-time.Duration(s * float64(30*24*time.Hour)))
		return fmt.Sprintf("%s > '%s'", csa.Name, t.UTC().Format(time.RFC3339))
	default:
		return "WorkflowType = 'visibilityStressWorker'"
	}
}

func (e *visibilityStressExecutor) csasByType(t csaType) []csaDef {
	var result []csaDef
	for _, csa := range e.config.CSADefs {
		if csa.Type == t {
			result = append(result, csa)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// Workflow Input Builder
// ---------------------------------------------------------------------------

// buildWorkflowInput constructs a VisibilityWorkerInput for one workflow.
// It resolves fractional updatesPerWF probabilistically, rolls the failure/timeout
// dice, and builds randomized CSA update groups from the csaPreset.
//
// rng is supplied by the caller rather than taken from e.rng: this runs on every writer
// goroutine, and math/rand.Rand is not safe for concurrent use.
func (e *visibilityStressExecutor) buildWorkflowInput(rng *vsRand) *vstypes.VisibilityWorkerInput {
	cfg := e.config.Load

	wholeUpdates := int(cfg.UpdatesPerWF)
	fraction := cfg.UpdatesPerWF - float64(wholeUpdates)
	numUpdates := wholeUpdates
	if rng.Float64() < fraction {
		numUpdates++
	}

	roll := rng.Float64()
	shouldFail := roll < cfg.FailPercent
	shouldTimeout := !shouldFail && roll < cfg.FailPercent+cfg.TimeoutPercent

	groups := make([]vstypes.CSAUpdateGroup, 0, numUpdates)
	shuffled := make([]csaDef, len(e.config.CSADefs))
	copy(shuffled, e.config.CSADefs)
	rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	for i := 0; i < numUpdates; i++ {
		groupSize := 1 + rng.Intn(3)
		attrs := make(map[string]any, groupSize)
		for j := 0; j < groupSize; j++ {
			csa := shuffled[(i*3+j)%len(shuffled)]
			attrs[csa.Name] = randomCSAValue(csa, rng)
		}
		groups = append(groups, vstypes.CSAUpdateGroup{Attributes: attrs})
	}

	return &vstypes.VisibilityWorkerInput{
		CSAUpdates:    groups,
		Delay:         cfg.UpdateDelay,
		ShouldFail:    shouldFail,
		ShouldTimeout: shouldTimeout,
	}
}

// workflowTimeout returns the execution timeout for a single workflow: the explicit
// workflowTimeout option when set, otherwise the value derived from its CSA update count.
func (e *visibilityStressExecutor) workflowTimeout(numCSAUpdates int) time.Duration {
	if e.config.WorkflowTimeout > 0 {
		return e.config.WorkflowTimeout
	}
	return vsComputeTimeout(numCSAUpdates)
}

func vsComputeTimeout(numCSAUpdates int) time.Duration {
	t := time.Duration(numCSAUpdates*2) * time.Second
	if t < 5*time.Second {
		t = 5 * time.Second
	}
	return t
}

// ---------------------------------------------------------------------------
// Cleanup
// ---------------------------------------------------------------------------

// runCleanup terminates all running workflows and deletes all workflows (terminal and running)
// for this scenario's task queue across all namespaces. Used with --option cleanup=true.
func (e *visibilityStressExecutor) runCleanup(ctx context.Context, info loadgen.ScenarioInfo) error {
	info.Logger.Info("Running cleanup mode...")

	for i, ns := range e.namespaces {
		info.Logger.Infof("Cleaning up namespace %s...", ns)

		// Terminate running workflows.
		query := fmt.Sprintf(
			"WorkflowType = 'visibilityStressWorker' AND ExecutionStatus = 'Running' AND TaskQueue = '%s'",
			e.taskQueue)
		for {
			resp, err := e.clients[i].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
				Namespace: ns,
				Query:     query,
				PageSize:  100,
			})
			if err != nil {
				return fmt.Errorf("[cleanup/%s] list running failed: %w", ns, err)
			}
			if len(resp.Executions) == 0 {
				break
			}
			for _, exec := range resp.Executions {
				_ = e.clients[i].TerminateWorkflow(ctx, exec.Execution.WorkflowId, exec.Execution.RunId, "cleanup")
			}
		}
		info.Logger.Infof("[cleanup/%s] Terminated all running workflows", ns)

		// Delete all workflows.
		deleteQuery := fmt.Sprintf(
			"WorkflowType = 'visibilityStressWorker' AND TaskQueue = '%s'",
			e.taskQueue)
		for {
			resp, err := e.clients[i].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
				Namespace: ns,
				Query:     deleteQuery,
				PageSize:  100,
			})
			if err != nil {
				return fmt.Errorf("[cleanup/%s] list for deletion failed: %w", ns, err)
			}
			if len(resp.Executions) == 0 {
				break
			}
			for _, exec := range resp.Executions {
				wfExec := exec.Execution
				_, err := e.clients[i].WorkflowService().DeleteWorkflowExecution(ctx,
					&workflowservice.DeleteWorkflowExecutionRequest{
						Namespace: ns,
						WorkflowExecution: &commonpb.WorkflowExecution{
							WorkflowId: wfExec.WorkflowId,
							RunId:      wfExec.RunId,
						},
					})
				if err != nil {
					info.Logger.Warnf("[cleanup/%s] delete %s failed: %v", ns, wfExec.WorkflowId, err)
				}
			}
		}
		info.Logger.Infof("[cleanup/%s] Deleted all workflows", ns)
	}

	if e.config.DeleteNamespaces && e.config.NamespaceCount > 1 {
		info.Logger.Warn("Namespace deletion not supported via API in all environments. Delete manually if needed.")
	}

	info.Logger.Info("Cleanup complete.")
	return nil
}
