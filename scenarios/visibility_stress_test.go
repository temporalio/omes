package scenarios

import (
	"context"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/loadgen"
	"github.com/temporalio/omes/workers"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/serviceerror"
	workflowservice "go.temporal.io/api/workflowservice/v1"
	sdkclient "go.temporal.io/sdk/client"
	sdkmocks "go.temporal.io/sdk/mocks"
	"golang.org/x/time/rate"
)

// TestVisibilityStressConfigure tests the configuration/validation logic without
// running any workflows.
func TestVisibilityStressConfigure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		options     map[string]string
		config      loadgen.RunConfiguration
		expectError string
	}{
		{
			name:        "no presets → error",
			options:     map[string]string{},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "at least one of loadPreset or queryPreset must be set",
		},
		{
			name:        "iterations not supported",
			options:     map[string]string{"loadPreset": "light"},
			config:      loadgen.RunConfiguration{Iterations: 10},
			expectError: "does not support --iterations",
		},
		{
			name:        "duration required",
			options:     map[string]string{"loadPreset": "light"},
			config:      loadgen.RunConfiguration{},
			expectError: "requires --duration",
		},
		{
			name:        "unknown loadPreset",
			options:     map[string]string{"loadPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown loadPreset",
		},
		{
			name:        "unknown queryPreset",
			options:     map[string]string{"queryPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown queryPreset",
		},
		{
			name:        "unknown csaPreset",
			options:     map[string]string{"loadPreset": "light", "csaPreset": "nonexistent"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "unknown csaPreset",
		},
		{
			name:        "read-only without csaPreset → error",
			options:     map[string]string{"queryPreset": "light"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "csaPreset is required in read-only mode",
		},
		{
			name:        "failPercent + timeoutPercent >= 1.0 → error",
			options:     map[string]string{"loadPreset": "light", "failPercent": "0.6", "timeoutPercent": "0.5"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "failPercent + timeoutPercent must be < 1.0",
		},
		{
			name:        "retention too short → error",
			options:     map[string]string{"loadPreset": "light", "retention": "1h"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "retention must be >= 24h",
		},
		{
			name:    "valid write-only config",
			options: map[string]string{"loadPreset": "light"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:    "valid read-only config",
			options: map[string]string{"queryPreset": "light", "csaPreset": "small"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:        "pageContinueProb out of range → error",
			options:     map[string]string{"queryPreset": "light", "csaPreset": "small", "pageContinueProb": "1.5"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "pageContinueProb must be in [0,1]",
		},
		{
			name:        "maxQueryPages below 1 → error",
			options:     map[string]string{"queryPreset": "light", "csaPreset": "small", "maxQueryPages": "0"},
			config:      loadgen.RunConfiguration{Duration: 1 * time.Minute},
			expectError: "maxQueryPages must be >= 1",
		},
		{
			name:    "valid full config with overrides",
			options: map[string]string{"loadPreset": "moderate", "queryPreset": "light", "csaPreset": "heavy", "wfRPS": "50", "deleteRPS": "0"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
		{
			name:    "cleanup mode skips preset validation",
			options: map[string]string{"cleanup": "true"},
			config:  loadgen.RunConfiguration{},
		},
		{
			name:    "loadPreset defaults csaPreset to medium",
			options: map[string]string{"loadPreset": "light"},
			config:  loadgen.RunConfiguration{Duration: 1 * time.Minute},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executor := &visibilityStressExecutor{}
			info := loadgen.ScenarioInfo{
				RunID:           "test-run",
				Configuration:   tc.config,
				ScenarioOptions: tc.options,
				Namespace:       "default",
			}
			err := executor.Configure(info)
			if tc.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestVisibilityStressQueryConcurrency(t *testing.T) {
	t.Parallel()

	configure := func(value string) (*visibilityStressExecutor, error) {
		options := map[string]string{
			"queryPreset": "heavy",
			"csaPreset":   "medium",
		}
		if value != "" {
			options["queryConcurrency"] = value
		}
		executor := &visibilityStressExecutor{}
		err := executor.Configure(loadgen.ScenarioInfo{
			Configuration:   loadgen.RunConfiguration{Duration: time.Minute},
			ScenarioOptions: options,
			Namespace:       "default",
		})
		return executor, err
	}

	executor, err := configure("")
	require.NoError(t, err)
	assert.Equal(t, 1, executor.config.QueryConcurrency)

	executor, err = configure("100")
	require.NoError(t, err)
	assert.Equal(t, 100, executor.config.QueryConcurrency)

	_, err = configure("0")
	require.ErrorContains(t, err, "queryConcurrency must be >= 1")
}

func TestVisibilityStressDeleteConcurrency(t *testing.T) {
	t.Parallel()

	configure := func(options map[string]string) (*visibilityStressExecutor, error) {
		executor := &visibilityStressExecutor{}
		err := executor.Configure(loadgen.ScenarioInfo{
			Configuration:   loadgen.RunConfiguration{Duration: time.Minute},
			ScenarioOptions: options,
			Namespace:       "default",
		})
		return executor, err
	}

	executor, err := configure(map[string]string{
		"loadPreset": "no-failures",
		"deleteRPS":  "1381.84",
	})
	require.NoError(t, err)
	assert.Equal(t, 56, executor.config.DeleteConcurrency)
	assert.Equal(t, time.Minute, executor.config.DeleteGracePeriod)

	executor, err = configure(map[string]string{
		"loadPreset":        "no-failures",
		"deleteRPS":         "1381.84",
		"deleteConcurrency": "64",
	})
	require.NoError(t, err)
	assert.Equal(t, 64, executor.config.DeleteConcurrency)

	executor, err = configure(map[string]string{
		"loadPreset":        "no-failures",
		"deleteGracePeriod": "45s",
	})
	require.NoError(t, err)
	assert.Equal(t, 45*time.Second, executor.config.DeleteGracePeriod)

	_, err = configure(map[string]string{
		"loadPreset":        "no-failures",
		"deleteConcurrency": "0",
	})
	require.ErrorContains(t, err, "deleteConcurrency must be >= 1")

	_, err = configure(map[string]string{
		"loadPreset":        "no-failures",
		"deleteGracePeriod": "-1s",
	})
	require.ErrorContains(t, err, "deleteGracePeriod must be non-negative")
}

func TestVisibilityStressDeleterQueryScopesToExecutorInvocation(t *testing.T) {
	t.Parallel()

	cutoff := time.Date(2026, time.September, 16, 17, 0, 0, 123, time.FixedZone("test", -7*60*60))
	assert.Equal(t,
		"WorkflowType = 'visibilityStressWorker' AND ExecutionStatus != 'Running' AND "+
			"TaskQueue = 'queue''one' AND WorkflowId STARTS_WITH 'vs-run''id-exec''id-' AND "+
			"CloseTime < '2026-09-17T00:00:00.000000123Z'",
		visibilityStressDeleterQuery("queue'one", "run'id", "exec'id", cutoff),
	)
	assert.NotContains(t,
		visibilityStressDeleterQuery("queue", "run", "exec", time.Time{}),
		"CloseTime")
	assert.Equal(t, "vs-r01-01JABC-", visibilityStressWorkflowIDPrefix("r01", "01JABC"))
	assert.Equal(t, 1000, visibilityStressDeleteSeenLimit(1, time.Minute))
	assert.Equal(t, 2400, visibilityStressDeleteSeenLimit(20, time.Minute))
}

func TestVisibilityStressDeleteShouldRetry(t *testing.T) {
	t.Parallel()

	assert.False(t, visibilityStressDeleteShouldRetry(nil))
	assert.False(t, visibilityStressDeleteShouldRetry(serviceerror.NewNotFound("gone")))
	assert.False(t, visibilityStressDeleteShouldRetry(
		fmt.Errorf("wrapped: %w", serviceerror.NewNotFound("gone"))))
	assert.True(t, visibilityStressDeleteShouldRetry(fmt.Errorf("temporary failure")))
}

func TestVSRecentDeleteSetDeduplicatesAndRetriesFailures(t *testing.T) {
	t.Parallel()

	recent := newVSRecentDeleteSet(2)
	execution := func(id string) *commonpb.WorkflowExecution {
		return &commonpb.WorkflowExecution{WorkflowId: id, RunId: "run"}
	}

	first, ok := recent.add(execution("one"))
	require.True(t, ok)
	_, ok = recent.add(execution("one"))
	require.False(t, ok, "an in-flight or successful delete must not be submitted twice")

	recent.remove(first)
	retry, ok := recent.add(execution("one"))
	require.True(t, ok, "a failed delete must become eligible for retry")
	recent.remove(first)
	_, ok = recent.add(execution("one"))
	require.False(t, ok, "an old failed attempt must not remove a newer generation")

	_, ok = recent.add(execution("two"))
	require.True(t, ok)
	_, ok = recent.add(execution("three"))
	require.True(t, ok)
	_, ok = recent.add(execution("one"))
	require.True(t, ok, "the bounded set must eventually evict its oldest generation")
	_ = retry
}

func TestRunDeleteWorkersUsesConfiguredConcurrency(t *testing.T) {
	t.Parallel()

	const concurrency = 4
	jobs := make(chan vsDeleteCandidate, concurrency)
	results := make(chan vsDeleteResult, concurrency)
	for i := range concurrency {
		jobs <- vsDeleteCandidate{
			execution: &commonpb.WorkflowExecution{WorkflowId: fmt.Sprintf("wf-%d", i)},
		}
	}
	close(jobs)

	entered := make(chan struct{}, concurrency)
	release := make(chan struct{})
	done := make(chan struct{})
	go func() {
		runDeleteWorkers(
			context.Background(),
			jobs,
			results,
			rate.NewLimiter(rate.Inf, 1),
			concurrency,
			func(context.Context, *commonpb.WorkflowExecution) error {
				entered <- struct{}{}
				<-release
				return nil
			},
		)
		close(results)
		close(done)
	}()

	for range concurrency {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("delete workers did not issue requests concurrently")
		}
	}
	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("delete workers did not finish")
	}
	var resultCount int
	for result := range results {
		require.NoError(t, result.err)
		resultCount++
	}
	require.Equal(t, concurrency, resultCount)
}

func TestRunQuerierUsesConfiguredConcurrency(t *testing.T) {
	t.Parallel()

	const concurrency = 4
	entered := make(chan struct{}, concurrency)
	release := make(chan struct{})
	mockClient := &sdkmocks.Client{}
	mockClient.On("ListWorkflow", mock.Anything, mock.Anything).
		Run(func(mock.Arguments) {
			entered <- struct{}{}
			<-release
		}).
		Return(&workflowservice.ListWorkflowExecutionsResponse{}, nil)

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Query: &vsQueryPreset{
				ListRPS:            10_000,
				ListNoFilterWeight: 1,
			},
			QueryConcurrency: concurrency,
			VocabZipfSkew:    vsDefaultVocabZipfSkew,
		},
		clients:    []sdkclient.Client{mockClient},
		namespaces: []string{"default"},
		rng:        newVSRand(42, vsDefaultVocabZipfSkew),
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		executor.runQuerier(ctx, loadgen.ScenarioInfo{})
	}()

	for range concurrency {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("query workers did not issue requests concurrently")
		}
	}

	cancel()
	close(release)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("query workers did not stop after cancellation")
	}
	mockClient.AssertNumberOfCalls(t, "ListWorkflow", concurrency)
}

// TestCSAParsing tests the CSA name prefix parsing logic.
func TestCSAParsing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		expectType  csaType
		expectError bool
	}{
		{"int", "VS_Int_01", csaTypeInt, false},
		{"keyword", "VS_Keyword_01", csaTypeKeyword, false},
		{"bool", "VS_Bool_01", csaTypeBool, false},
		{"double", "VS_Double_01", csaTypeDouble, false},
		{"text", "VS_Text_01", csaTypeText, false},
		{"datetime", "VS_Datetime_01", csaTypeDatetime, false},
		{"unknown prefix", "VS_Foo_01", 0, true},
		{"no prefix", "SomeAttribute", 0, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			def, err := parseCSAName(tc.input)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectType, def.Type)
				assert.Equal(t, tc.input, def.Name)
			}
		})
	}
}

// TestBuildWorkflowInput tests the workflow input builder logic.
func TestBuildWorkflowInput(t *testing.T) {
	t.Parallel()

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Load: &vsLoadPreset{
				UpdatesPerWF:   3,
				UpdateDelay:    100 * time.Millisecond,
				FailPercent:    0,
				TimeoutPercent: 0,
			},
			CSADefs: []csaDef{
				{Name: "VS_Int_01", Type: csaTypeInt},
				{Name: "VS_Keyword_01", Type: csaTypeKeyword},
				{Name: "VS_Bool_01", Type: csaTypeBool},
			},
		},
		rng: newVSRand(42, vsDefaultVocabZipfSkew),
	}

	input := executor.buildWorkflowInput(executor.rng)

	// With updatesPerWF=3 (no fractional part), we should get exactly 3 groups.
	assert.Equal(t, 3, len(input.CSAUpdates))
	assert.Equal(t, 100*time.Millisecond, input.Delay)
	assert.False(t, input.ShouldFail)
	assert.False(t, input.ShouldTimeout)

	// Each group should have 1-3 attributes.
	for _, group := range input.CSAUpdates {
		assert.GreaterOrEqual(t, len(group.Attributes), 1)
		assert.LessOrEqual(t, len(group.Attributes), 3)
	}
}

// TestBuildWorkflowInputFractional tests fractional updatesPerWF.
func TestBuildWorkflowInputFractional(t *testing.T) {
	t.Parallel()

	executor := &visibilityStressExecutor{
		config: &vsConfig{
			Load: &vsLoadPreset{
				UpdatesPerWF:   0.5, // 50% get 1 update, 50% get 0
				UpdateDelay:    time.Second,
				FailPercent:    0,
				TimeoutPercent: 0,
			},
			CSADefs: []csaDef{
				{Name: "VS_Int_01", Type: csaTypeInt},
			},
		},
		rng: newVSRand(42, vsDefaultVocabZipfSkew),
	}

	// Generate many inputs and check the distribution.
	var withUpdates, withoutUpdates int
	for i := 0; i < 1000; i++ {
		input := executor.buildWorkflowInput(executor.rng)
		if len(input.CSAUpdates) > 0 {
			withUpdates++
		} else {
			withoutUpdates++
		}
	}

	// With 0.5, roughly half should have updates. Allow wide margin.
	assert.InDelta(t, 500, withUpdates, 100, "expected ~50%% to have updates")
	assert.InDelta(t, 500, withoutUpdates, 100, "expected ~50%% to have no updates")
}

// newQueryTestExecutor builds an executor wired for filter-generation tests.
func newQueryTestExecutor(csaDefs []csaDef, weights vsQueryPreset) *visibilityStressExecutor {
	weights.CountRPS = 1
	weights.ListRPS = 1
	weights.SelMu = vsDefaultSelMu
	weights.SelSigma = vsDefaultSelSigma
	return &visibilityStressExecutor{
		config: &vsConfig{
			Query:         &weights,
			CSADefs:       csaDefs,
			VocabZipfSkew: vsDefaultVocabZipfSkew,
		},
		rng: newVSRand(42, vsDefaultVocabZipfSkew),
	}
}

func queryTestTotalWeight(q vsQueryPreset) int {
	return q.ListNoFilterWeight + q.ListOpenWeight + q.ListClosedWeight +
		q.ListSimpleCSAWeight + q.ListCompoundCSAWeight + q.ListDisjunctionWeight
}

// TestQueryGeneration tests that generated queries reference only CSAs from the preset.
func TestQueryGeneration(t *testing.T) {
	t.Parallel()

	weights := vsQueryPreset{
		ListNoFilterWeight:    1,
		ListOpenWeight:        1,
		ListClosedWeight:      1,
		ListSimpleCSAWeight:   5, // bias toward CSA filters
		ListCompoundCSAWeight: 5,
		ListDisjunctionWeight: 5,
	}
	executor := newQueryTestExecutor([]csaDef{
		{Name: "VS_Int_01", Type: csaTypeInt},
		{Name: "VS_Keyword_01", Type: csaTypeKeyword},
	}, weights)
	totalWeight := queryTestTotalWeight(weights)

	// Generate many queries and verify they only reference known CSAs.
	for i := 0; i < 100; i++ {
		filter, class := executor.generateFilter(false, totalWeight)

		assert.NotEmpty(t, class)

		// Should never reference VS_Int_02, VS_Keyword_02, etc. (not in preset).
		assert.NotContains(t, filter, "VS_Int_02")
		assert.NotContains(t, filter, "VS_Keyword_02")
		assert.NotContains(t, filter, "VS_Bool_")
		assert.NotContains(t, filter, "VS_Double_")
		assert.NotContains(t, filter, "VS_Text_")
		assert.NotContains(t, filter, "VS_Datetime_")
	}
}

// TestDisjunctionFilter verifies the disjunction class ORs distinct CSA clauses.
func TestDisjunctionFilter(t *testing.T) {
	t.Parallel()

	executor := newQueryTestExecutor([]csaDef{
		{Name: "VS_Int_01", Type: csaTypeInt},
		{Name: "VS_Keyword_01", Type: csaTypeKeyword},
		{Name: "VS_Bool_01", Type: csaTypeBool},
		{Name: "VS_Double_01", Type: csaTypeDouble},
	}, vsQueryPreset{})

	for i := 0; i < 100; i++ {
		filter := executor.generateDisjunctionFilter()

		assert.True(t, strings.HasPrefix(filter, "("), "disjunction should be parenthesized: %s", filter)
		assert.True(t, strings.HasSuffix(filter, ")"), "disjunction should be parenthesized: %s", filter)
		assert.Contains(t, filter, " OR ", "disjunction should contain OR: %s", filter)

		// Clauses must be over distinct CSAs, otherwise the filter can be
		// self-contradictory or trivially redundant.
		for _, csa := range []string{"VS_Keyword_01", "VS_Bool_01", "VS_Double_01"} {
			assert.LessOrEqual(t, strings.Count(filter, csa), 1,
				"CSA %s repeated in %s", csa, filter)
		}
	}
}

// TestDisjunctionSelectedByWeight verifies the disjunction class is reachable
// through generateFilter's weighted selection, and that a zero weight excludes it.
func TestDisjunctionSelectedByWeight(t *testing.T) {
	t.Parallel()

	csas := []csaDef{
		{Name: "VS_Int_01", Type: csaTypeInt},
		{Name: "VS_Keyword_01", Type: csaTypeKeyword},
	}

	t.Run("reachable when weighted", func(t *testing.T) {
		weights := vsQueryPreset{ListSimpleCSAWeight: 1, ListDisjunctionWeight: 1}
		executor := newQueryTestExecutor(csas, weights)
		total := queryTestTotalWeight(weights)

		var seen int
		for i := 0; i < 200; i++ {
			if _, class := executor.generateFilter(false, total); class == vsClassDisjunction {
				seen++
			}
		}
		assert.Greater(t, seen, 0, "disjunction class should be selected sometimes")
	})

	t.Run("excluded at zero weight", func(t *testing.T) {
		weights := vsQueryPreset{ListNoFilterWeight: 1, ListSimpleCSAWeight: 1}
		executor := newQueryTestExecutor(csas, weights)
		total := queryTestTotalWeight(weights)

		for i := 0; i < 200; i++ {
			_, class := executor.generateFilter(false, total)
			assert.NotEqual(t, vsClassDisjunction, class)
		}
	})
}

// TestVocabZipfSkewsWrites verifies write-side vocabulary draws are Zipf rather
// than uniform. Without the skew every keyword filter has identical selectivity,
// which flattens the query cost distribution.
func TestVocabZipfSkewsWrites(t *testing.T) {
	t.Parallel()

	rng := newVSRand(42, vsDefaultVocabZipfSkew)
	counts := make(map[string]int)
	const draws = 20000
	for i := 0; i < draws; i++ {
		counts[rng.vocabWord()]++
	}

	// The head of the distribution must dominate: rank 0 should appear far more
	// often than the uniform expectation of draws/len(vocabulary).
	uniformExpectation := draws / len(keywordVocabulary)
	assert.Greater(t, counts[keywordVocabulary[0]], 5*uniformExpectation,
		"most popular word should be heavily favored over uniform")

	// And the tail must stay reachable, otherwise there is no spread to measure.
	assert.Greater(t, len(counts), len(keywordVocabulary)/2,
		"most of the vocabulary should still be drawn at least once")
}

// TestTextCSAValuesAreMatchable guards the write/read mismatch: Text CSA values
// must be built from the same vocabulary the querier filters on, or every Text
// filter is a guaranteed zero-hit.
func TestTextCSAValuesAreMatchable(t *testing.T) {
	t.Parallel()

	rng := newVSRand(42, vsDefaultVocabZipfSkew)
	vocab := make(map[string]bool, len(keywordVocabulary))
	for _, w := range keywordVocabulary {
		vocab[w] = true
	}

	textCSA := csaDef{Name: "VS_Text_01", Type: csaTypeText}
	for i := 0; i < 200; i++ {
		value, ok := randomCSAValue(textCSA, rng).(string)
		require.True(t, ok, "text CSA value should be a string")

		words := strings.Fields(value)
		require.GreaterOrEqual(t, len(words), vsTextWordsMin)
		require.LessOrEqual(t, len(words), vsTextWordsMax)
		for _, w := range words {
			assert.True(t, vocab[w], "text value contains non-vocabulary token %q", w)
		}
	}
}

// TestSampleSelectivityIsLogNormal verifies the selectivity sampler produces a
// log-normal spread inside (0,1] rather than a constant or an out-of-range value.
func TestSampleSelectivityIsLogNormal(t *testing.T) {
	t.Parallel()

	rng := newVSRand(42, vsDefaultVocabZipfSkew)
	const draws = 20000
	var logSum float64
	var belowMedian int
	for i := 0; i < draws; i++ {
		s := rng.sampleSelectivity(vsDefaultSelMu, vsDefaultSelSigma)
		require.Greater(t, s, 0.0)
		require.LessOrEqual(t, s, 1.0)
		logSum += math.Log10(s)
		if s < math.Pow(10, vsDefaultSelMu) {
			belowMedian++
		}
	}

	// Clamping at 1.0 truncates the top tail, so the observed mean of log10(s)
	// sits slightly above mu; check it is in the right neighborhood.
	assert.InDelta(t, vsDefaultSelMu, logSum/draws, 0.3)

	// Roughly half the draws should fall below the median 10^mu.
	assert.InDelta(t, 0.5, float64(belowMedian)/draws, 0.05)
}

// TestCSAFilterClauseTargetsSelectivity verifies threshold inversion: a small
// target selectivity must yield a restrictive predicate, and a large one a
// permissive predicate.
func TestCSAFilterClauseTargetsSelectivity(t *testing.T) {
	t.Parallel()

	doubleCSA := csaDef{Name: "VS_Double_01", Type: csaTypeDouble}

	// sigma≈0 pins selectivity at exactly 10^mu so the inversion is checkable.
	executor := newQueryTestExecutor([]csaDef{doubleCSA}, vsQueryPreset{})
	executor.config.Query.SelSigma = 1e-9

	var restrictiveThreshold, permissiveThreshold float64
	_, err := fmt.Sscanf(
		executor.csaFilterClause(doubleCSA, -2), // 1% of the corpus
		"VS_Double_01 > %f", &restrictiveThreshold)
	require.NoError(t, err)
	_, err = fmt.Sscanf(
		executor.csaFilterClause(doubleCSA, math.Log10(0.9)), // 90% of the corpus
		"VS_Double_01 > %f", &permissiveThreshold)
	require.NoError(t, err)

	// Values are uniform over [0,1000], so matching 1% means a threshold near 990
	// and matching 90% means a threshold near 100.
	assert.InDelta(t, 990.0, restrictiveThreshold, 1.0)
	assert.InDelta(t, 100.0, permissiveThreshold, 1.0)
	assert.Greater(t, restrictiveThreshold, permissiveThreshold)
}

// TestComputeTimeout tests the execution timeout calculation.
func TestComputeTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5*time.Second, vsComputeTimeout(0))
	assert.Equal(t, 5*time.Second, vsComputeTimeout(1))
	assert.Equal(t, 5*time.Second, vsComputeTimeout(2))
	assert.Equal(t, 6*time.Second, vsComputeTimeout(3))
	assert.Equal(t, 20*time.Second, vsComputeTimeout(10))
}

// TestDeriveSelMu checks that the default selectivity target scales with the
// corpus each load preset builds, instead of staying fixed as a fraction.
func TestDeriveSelMu(t *testing.T) {
	t.Parallel()

	const runDuration = time.Hour

	tests := []struct {
		preset      string
		expectHitsX float64 // expected median hits, within a factor of 2
	}{
		{"light", vsTargetMedianHits},
		{"moderate", vsTargetMedianHits},
		{"heavy", vsTargetMedianHits},
	}

	var previousMu float64 = 1 // sentinel above any valid selMu
	for _, tc := range tests {
		t.Run(tc.preset, func(t *testing.T) {
			load := vsLoadPresets[tc.preset]
			mu := deriveSelMu(load, runDuration)
			corpus := load.estimateCorpusSize(runDuration)

			// The whole point: median query lands near the target hit count
			// regardless of how large the corpus is.
			medianHits := corpus * math.Pow(10, mu)
			assert.InEpsilon(t, tc.expectHitsX, medianHits, 1.0,
				"preset %s: corpus=%.0f mu=%.2f", tc.preset, corpus, mu)

			t.Logf("%s: corpus≈%.0f selMu=%.2f medianHits≈%.0f", tc.preset, corpus, mu, medianHits)
		})

		// Heavier presets must yield a strictly narrower default.
		mu := deriveSelMu(vsLoadPresets[tc.preset], runDuration)
		assert.Less(t, mu, previousMu, "preset %s should narrow selMu further", tc.preset)
		previousMu = mu
	}
}

// TestDeriveSelMuEdgeCases covers corpora too small to narrow and the floor.
func TestDeriveSelMuEdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("corpus smaller than target returns 0", func(t *testing.T) {
		// deleteRPS >= wfRPS means nothing accumulates; only running WFs exist.
		tiny := vsLoadPreset{WfRPS: 1, UpdatesPerWF: 1, UpdateDelay: time.Second, DeleteRPS: 10}
		assert.Equal(t, 0.0, deriveSelMu(tiny, time.Hour))
	})

	t.Run("enormous corpus is floored", func(t *testing.T) {
		huge := vsLoadPreset{WfRPS: 100000, UpdatesPerWF: 1, UpdateDelay: time.Second}
		assert.Equal(t, vsMinDerivedSelMu, deriveSelMu(huge, 30*24*time.Hour))
	})
}

// TestConfigureDerivesSelMuFromLoad verifies the wiring: an explicit selMu wins,
// otherwise the load preset determines it, and read-only mode falls back.
func TestConfigureDerivesSelMuFromLoad(t *testing.T) {
	t.Parallel()

	configure := func(options map[string]string) *vsConfig {
		executor := &visibilityStressExecutor{}
		require.NoError(t, executor.Configure(loadgen.ScenarioInfo{
			RunID:           "test-run",
			Configuration:   loadgen.RunConfiguration{Duration: time.Hour},
			ScenarioOptions: options,
			Namespace:       "default",
		}))
		return executor.config
	}

	t.Run("explicit selMu wins", func(t *testing.T) {
		cfg := configure(map[string]string{
			"loadPreset": "heavy", "queryPreset": "light", "selMu": "-1.25",
		})
		assert.Equal(t, -1.25, cfg.Query.SelMu)
	})

	t.Run("derived from load preset", func(t *testing.T) {
		light := configure(map[string]string{"loadPreset": "light", "queryPreset": "light"})
		heavy := configure(map[string]string{"loadPreset": "heavy", "queryPreset": "light"})
		assert.Less(t, heavy.Query.SelMu, light.Query.SelMu,
			"heavier load builds a bigger corpus and needs a narrower default")
	})

	t.Run("read-only falls back to the static default", func(t *testing.T) {
		cfg := configure(map[string]string{"queryPreset": "light", "csaPreset": "small"})
		assert.Equal(t, vsDefaultSelMu, cfg.Query.SelMu)
	})
}

// TestClauseMuCorrection verifies that multi-clause filters target the configured
// query-level selectivity rather than applying it to every clause independently.
func TestClauseMuCorrection(t *testing.T) {
	t.Parallel()

	t.Run("conjunction splits the target", func(t *testing.T) {
		// Three ANDed clauses at -1 each multiply out to the -3 query target.
		assert.Equal(t, -1.0, vsConjunctionMu(-3, 3))
		assert.Equal(t, -1.5, vsConjunctionMu(-3, 2))
		assert.Equal(t, -3.0, vsConjunctionMu(-3, 1))
	})

	t.Run("disjunction tightens each clause", func(t *testing.T) {
		// Two ORed clauses each matching s together match ~2s, so each is pulled
		// down by log10(2).
		assert.InDelta(t, -3-math.Log10(2), vsDisjunctionMu(-3, 2), 1e-9)
		assert.Equal(t, -3.0, vsDisjunctionMu(-3, 1))
	})

	t.Run("compound stays satisfiable at heavy-preset selMu", func(t *testing.T) {
		// Regression guard for the degenerate case: without the correction, three
		// clauses at 10^-3.5 combine to 10^-10.5 and match nothing on any corpus.
		executor := newQueryTestExecutor([]csaDef{
			{Name: "VS_Int_01", Type: csaTypeInt},
			{Name: "VS_Double_01", Type: csaTypeDouble},
			{Name: "VS_Datetime_01", Type: csaTypeDatetime},
		}, vsQueryPreset{})
		executor.config.Query.SelMu = -3.5
		executor.config.Query.SelSigma = 1e-9

		// All three CSAs are threshold-tunable, so each clause targets -3.5/n.
		// Verify via the Double clause: at n=3 the per-clause target is ~10^-1.17,
		// which is a threshold near 1000*(1-0.068)=932, not 1000*(1-0.0003)=999.7.
		for i := 0; i < 50; i++ {
			filter := executor.generateCompoundCSAFilter()
			if !strings.Contains(filter, "VS_Double_01 > ") {
				continue
			}
			var threshold float64
			_, err := fmt.Sscanf(
				filter[strings.Index(filter, "VS_Double_01 > "):],
				"VS_Double_01 > %f", &threshold)
			require.NoError(t, err)
			assert.Less(t, threshold, 990.0,
				"per-clause threshold should be relaxed by the clause-count split: %s", filter)
			return
		}
		t.Fatal("no Double clause generated in 50 attempts")
	})
}

// TestShouldFetchNextPage verifies the page count is geometric and bounded, so
// the cost distribution has a pagination tail instead of a hard edge at a fixed cap.
func TestShouldFetchNextPage(t *testing.T) {
	t.Parallel()

	newExecutor := func(prob float64, maxPages int) *visibilityStressExecutor {
		e := newQueryTestExecutor([]csaDef{{Name: "VS_Int_01", Type: csaTypeInt}}, vsQueryPreset{})
		e.config.Query.PageContinueProb = prob
		e.config.Query.MaxPages = maxPages
		return e
	}

	// samplePageCount walks the loop the querier runs, assuming results never run out.
	samplePageCount := func(e *visibilityStressExecutor) int {
		pages := 1
		for e.shouldFetchNextPage(pages) {
			pages++
		}
		return pages
	}

	t.Run("respects the cap", func(t *testing.T) {
		e := newExecutor(1.0, 4) // always continue
		for i := 0; i < 100; i++ {
			assert.Equal(t, 4, samplePageCount(e))
		}
	})

	t.Run("single page when probability is zero", func(t *testing.T) {
		e := newExecutor(0, 10)
		for i := 0; i < 100; i++ {
			assert.Equal(t, 1, samplePageCount(e))
		}
	})

	t.Run("geometric between the bounds", func(t *testing.T) {
		e := newExecutor(vsDefaultPageContinueProb, vsDefaultMaxQueryPages)

		const draws = 20000
		counts := make(map[int]int)
		total := 0
		for i := 0; i < draws; i++ {
			pages := samplePageCount(e)
			require.GreaterOrEqual(t, pages, 1)
			require.LessOrEqual(t, pages, vsDefaultMaxQueryPages)
			counts[pages]++
			total += pages
		}

		// P(exactly 1 page) = 1 - p, P(exactly 2) = p(1-p).
		assert.InDelta(t, 1-vsDefaultPageContinueProb, float64(counts[1])/draws, 0.02)
		assert.InDelta(t, vsDefaultPageContinueProb*(1-vsDefaultPageContinueProb),
			float64(counts[2])/draws, 0.02)

		// Mean matches the closed form used in the startup log.
		assert.InDelta(t, vsMeanPages(vsDefaultPageContinueProb, vsDefaultMaxQueryPages),
			float64(total)/draws, 0.05)

		// The tail must actually be reachable, otherwise the cap is just a lower cap.
		assert.Greater(t, counts[4], 0, "deep pages should occur sometimes")
	})
}

func TestVSMeanPages(t *testing.T) {
	t.Parallel()

	// Never continue → always exactly one page.
	assert.InDelta(t, 1.0, vsMeanPages(0, 10), 1e-9)
	// Always continue → always exactly maxPages.
	assert.InDelta(t, 5.0, vsMeanPages(1, 5), 1e-9)
	// Truncated geometric: 1 + p + p^2 for a 3-page cap.
	assert.InDelta(t, 1+0.4+0.16, vsMeanPages(0.4, 3), 1e-9)
	// Approaches the untruncated 1/(1-p) as the cap grows.
	assert.InDelta(t, 1/(1-0.4), vsMeanPages(0.4, 40), 1e-6)
}

// TestVisibilityStressWriteOnly runs the full scenario in write-only mode
// against a real dev server. This is the integration test.
func TestVisibilityStressWriteOnly(t *testing.T) {
	t.Parallel()

	env := workers.SetupTestEnvironment(t,
		workers.WithExecutorTimeout(1*time.Minute))

	executor := &visibilityStressExecutor{}
	scenarioInfo := loadgen.ScenarioInfo{
		RunID: fmt.Sprintf("vs-test-%d", time.Now().Unix()),
		Configuration: loadgen.RunConfiguration{
			Duration: 5 * time.Second,
		},
		ScenarioOptions: map[string]string{
			"loadPreset":     "light",
			"csaPreset":      "small",
			"wfRPS":          "5", // low rate for test speed
			"updatesPerWF":   "2", // few updates per WF
			"deleteRPS":      "0", // no deletes (keep test simple)
			"failPercent":    "0", // no failures
			"timeoutPercent": "0", // no timeouts
		},
	}

	_, err := env.RunExecutorTest(t, executor, scenarioInfo, clioptions.LangGo)
	require.NoError(t, err, "Executor should complete successfully")

	// Verify some workflows were created.
	require.Greater(t, executor.totalCreated.Load(), int64(0),
		"Should have created at least one workflow")

	t.Logf("Created %d workflows, errors: %d",
		executor.totalCreated.Load(), executor.totalErrors.Load())
}
