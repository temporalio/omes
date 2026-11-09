package visibility

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/metrics"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

const (
	testSeed      = 42
	listTemplate  = "list-tenant"
	countTemplate = "count-recent"
)

type (
	// fakeVisibility records the reader's requests. respond decides each request's outcome; nil succeeds at once.
	fakeVisibility struct {
		respond func(ctx context.Context, kind string) error

		mu     sync.Mutex
		lists  []*workflowservice.ListWorkflowExecutionsRequest
		counts []*workflowservice.CountWorkflowExecutionsRequest
	}

	// fakePacer releases one read per value sent on ticks.
	fakePacer struct {
		ticks chan struct{}
	}

	readerHarness struct {
		r        *Reader
		pacer    *fakePacer
		registry *prometheus.Registry
		logs     *observer.ObservedLogs
		cancel   context.CancelFunc
		done     chan struct{}
	}
)

// TestNewReaderValidatesConfig verifies that NewReader rejects configs it cannot run.
func TestNewReaderValidatesConfig(t *testing.T) {
	reads := func(mutate func(*kitchensink.VisibilityReads)) *kitchensink.VisibilityReads {
		r := testReads()
		mutate(r)
		return r
	}
	for name, tc := range map[string]struct {
		reads *kitchensink.VisibilityReads
		want  string
	}{
		"nil reads":       {nil, "at least one template"},
		"no templates":    {reads(func(r *kitchensink.VisibilityReads) { r.Templates = nil }), "at least one template"},
		"zero rate":       {reads(func(r *kitchensink.VisibilityReads) { r.RatePerSecond = 0 }), "rate must be positive"},
		"zero weight":     {reads(func(r *kitchensink.VisibilityReads) { r.Templates[0].Weight = 0 }), "no weight"},
		"undeclared attr": {reads(func(r *kitchensink.VisibilityReads) { r.Templates[0].Query = "Nope = {Nope}" }), "Nope"},
		"huge page size": {reads(func(r *kitchensink.VisibilityReads) { r.Templates[0].PageSize = math.MaxInt32 + 1 }),
			"page size"},
		"duplicate name": {reads(func(r *kitchensink.VisibilityReads) { r.Templates[1].Name = r.Templates[0].Name }),
			"duplicate"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := NewReader(newConfig(tc.reads, &fakeVisibility{}, newTestHandler()))
			require.ErrorContains(t, err, tc.want)
		})
	}
}

// TestNewReaderRequiresRand verifies that a config without a random source is rejected.
func TestNewReaderRequiresRand(t *testing.T) {
	cfg := newConfig(testReads(), &fakeVisibility{}, newTestHandler())
	cfg.Rand = nil
	_, err := NewReader(cfg)
	require.ErrorContains(t, err, "random source")
}

// TestReaderInFlightCapDefaults verifies that a zero MaxInFlight becomes max(64, ceil(rate)) and an explicit value
// is used as given.
func TestReaderInFlightCapDefaults(t *testing.T) {
	for name, tc := range map[string]struct {
		rate        float64
		maxInFlight uint32
		want        int
	}{
		"floor":        {0.5, 0, 64},
		"at floor":     {64, 0, 64},
		"fractional":   {100.5, 0, 101},
		"above floor":  {500, 0, 500},
		"explicit set": {500, 3, 3},
	} {
		t.Run(name, func(t *testing.T) {
			reads := testReads()
			reads.RatePerSecond, reads.MaxInFlight = tc.rate, tc.maxInFlight
			r, err := NewReader(newConfig(reads, &fakeVisibility{}, newTestHandler()))
			require.NoError(t, err)
			require.Equal(t, tc.want, cap(r.slots))
		})
	}
}

// TestReaderPickTracksWeights verifies that over many draws each template is picked in proportion to its weight.
func TestReaderPickTracksWeights(t *testing.T) {
	reads := testReads()
	reads.Templates = nil
	weights := map[string]uint32{"a": 50, "b": 30, "c": 15, "d": 5}
	for _, name := range []string{"a", "b", "c", "d"} {
		reads.Templates = append(reads.Templates, &kitchensink.VisibilityReadTemplate{
			Name: name, Query: "VisTenant = {VisTenant}", Weight: weights[name]})
	}
	r, err := NewReader(newConfig(reads, &fakeVisibility{}, newTestHandler()))
	require.NoError(t, err)
	const draws = 200_000
	picks := map[string]int{}
	for range draws {
		picks[r.pick().name]++
	}
	for name, w := range weights {
		require.InDelta(t, float64(w)/100, float64(picks[name])/draws, 0.005, name)
	}
}

// TestReaderIssuesRenderedReads verifies that list templates become ListWorkflow calls with the template's page size
// and count templates become CountWorkflow calls, both with a rendered query, and that each read is counted and
// timed under its template, kind and namespace.
func TestReaderIssuesRenderedReads(t *testing.T) {
	c := &fakeVisibility{}
	h := newReaderHarness(t, c, 64)
	h.start(t)
	h.tick(t, 40)
	require.Eventually(t, func() bool {
		return metricValue(t, h.registry, ReadsMetric) == 40
	}, 5*time.Second, 10*time.Millisecond)

	lists, counts := c.requests()
	require.NotEmpty(t, lists)
	require.NotEmpty(t, counts)
	require.Equal(t, 40, len(lists)+len(counts))
	for _, req := range lists {
		require.Equal(t, "ns", req.GetNamespace())
		require.Equal(t, int32(7), req.GetPageSize())
		require.True(t, strings.HasPrefix(req.GetQuery(), "VisTenant = '"), req.GetQuery())
	}
	for _, req := range counts {
		require.Equal(t, "ns", req.GetNamespace())
		require.True(t, strings.HasPrefix(req.GetQuery(), "StartTime > '"), req.GetQuery())
		require.Contains(t, req.GetQuery(), "WorkflowId IN ('")
	}
	list := []string{tagTemplate, listTemplate, tagKind, "list", tagNamespace, "ns"}
	count := []string{tagTemplate, countTemplate, tagKind, "count", tagNamespace, "ns"}
	require.InDelta(t, len(lists), metricValue(t, h.registry, ReadsMetric, list...), 0)
	require.InDelta(t, len(counts), metricValue(t, h.registry, ReadsMetric, count...), 0)
	require.InDelta(t, len(lists), metricValue(t, h.registry, ReadLatencyMetric, list...), 0)
	require.InDelta(t, len(counts), metricValue(t, h.registry, ReadLatencyMetric, count...), 0)
	require.Zero(t, metricValue(t, h.registry, ReadErrorsMetric))
}

// TestReaderDefaultsPageSize verifies that a list template with page size 0 asks for 100 results.
func TestReaderDefaultsPageSize(t *testing.T) {
	reads := testReads()
	reads.Templates = reads.Templates[:1]
	reads.Templates[0].PageSize = 0
	c := &fakeVisibility{}
	h := newReaderHarnessFor(t, reads, c)
	h.start(t)
	h.tick(t, 1)
	require.Eventually(t, func() bool { return c.calls() == 1 }, 5*time.Second, 10*time.Millisecond)
	lists, _ := c.requests()
	require.Equal(t, int32(100), lists[0].GetPageSize())
}

// TestReaderCountsErrorsAndKeepsReading verifies that failed reads are counted by template and error class, that
// they don't stop the loop, and that each template's errors are logged once per log interval rather than per error.
func TestReaderCountsErrorsAndKeepsReading(t *testing.T) {
	var mu sync.Mutex
	countCalls := 0
	c := &fakeVisibility{respond: func(ctx context.Context, kind string) error {
		if kind == kindList {
			return errors.New("boom")
		}
		mu.Lock()
		countCalls++
		serverTimeout := countCalls%2 == 0
		mu.Unlock()
		if serverTimeout {
			return serviceerror.NewDeadlineExceeded("server gave up")
		}
		<-ctx.Done()
		return ctx.Err()
	}}
	h := newReaderHarness(t, c, 64)
	h.r.readTimeout = 10 * time.Millisecond
	h.start(t)
	h.tick(t, 30)
	require.Eventually(t, func() bool {
		return metricValue(t, h.registry, ReadErrorsMetric) == 30
	}, 5*time.Second, 10*time.Millisecond)

	lists, counts := c.requests()
	require.NotEmpty(t, lists)
	require.Greater(t, len(counts), 1)
	listErrors := func(class string) float64 {
		return metricValue(t, h.registry, ReadErrorsMetric, tagTemplate, listTemplate, tagKind, "list",
			tagError, class)
	}
	countErrors := func(class string) float64 {
		return metricValue(t, h.registry, ReadErrorsMetric, tagTemplate, countTemplate, tagKind, "count",
			tagError, class)
	}
	require.InDelta(t, len(lists), listErrors(errorClassOther), 0)
	require.Zero(t, listErrors(errorClassTimeout))
	require.InDelta(t, len(counts), countErrors(errorClassTimeout), 0)
	require.Zero(t, countErrors(errorClassOther))
	require.Zero(t, metricValue(t, h.registry, ReadsMetric))
	// The clock never moves, so each template logs its first error and suppresses the rest.
	require.Equal(t, 2, h.logs.FilterLevelExact(zapcore.WarnLevel).Len())
}

// TestReaderSkipsTicksAtInFlightCap verifies that a tick finding the in-flight cap reached is skipped and counted
// rather than waiting for a slot, and that finished reads free their slots.
func TestReaderSkipsTicksAtInFlightCap(t *testing.T) {
	release := make(chan struct{})
	c := &fakeVisibility{respond: func(ctx context.Context, _ string) error {
		select {
		case <-release:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}}
	h := newReaderHarness(t, c, 2)
	h.start(t)
	h.tick(t, 5)
	require.Eventually(t, func() bool {
		return metricValue(t, h.registry, ReadsSkippedMetric) == 3
	}, 5*time.Second, 10*time.Millisecond)
	require.Eventually(t, func() bool { return c.calls() == 2 }, 5*time.Second, 10*time.Millisecond)

	close(release)
	require.Eventually(t, func() bool {
		return metricValue(t, h.registry, ReadsMetric) == 2
	}, 5*time.Second, 10*time.Millisecond)
	// The read counter is bumped before the slot is released, so wait for the slots to drain.
	require.Eventually(t, func() bool { return len(h.r.slots) == 0 }, 5*time.Second, 10*time.Millisecond)
	h.tick(t, 1)
	require.Eventually(t, func() bool {
		return metricValue(t, h.registry, ReadsMetric) == 3
	}, 5*time.Second, 10*time.Millisecond)
}

// TestReaderTimeoutDuringDrain verifies that a read hitting its own timeout after the run's context ended is
// classified as a timeout, not as abandoned, because only the drain timeout abandons reads.
func TestReaderTimeoutDuringDrain(t *testing.T) {
	c := &fakeVisibility{respond: func(ctx context.Context, _ string) error {
		<-ctx.Done()
		return ctx.Err()
	}}
	h := newReaderHarness(t, c, 64)
	h.r.readTimeout = 20 * time.Millisecond
	h.start(t)
	h.tick(t, 1)
	h.cancel()
	requireDone(t, h.done)
	require.Equal(t, int64(0), h.r.abandoned.Load())
	require.InDelta(t, 1, metricValue(t, h.registry, ReadErrorsMetric, tagError, errorClassTimeout), 0)
}

// TestReaderStopsWhenLoadEnds verifies that Run returns promptly once its context is done, after waiting for reads
// in flight up to the drain timeout.
func TestReaderStopsWhenLoadEnds(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		h := newReaderHarness(t, &fakeVisibility{}, 64)
		h.start(t)
		h.cancel()
		requireDone(t, h.done)
	})

	t.Run("waits for reads in flight", func(t *testing.T) {
		release := make(chan struct{})
		c := &fakeVisibility{respond: func(context.Context, string) error {
			<-release
			return nil
		}}
		h := newReaderHarness(t, c, 64)
		h.start(t)
		h.tick(t, 1)
		require.Eventually(t, func() bool { return c.calls() == 1 }, 5*time.Second, 10*time.Millisecond)
		h.cancel()
		require.Never(t, func() bool { return isClosed(h.done) }, 200*time.Millisecond, 10*time.Millisecond)
		close(release)
		requireDone(t, h.done)
		require.InDelta(t, 1, metricValue(t, h.registry, ReadsMetric), 0)
	})

	t.Run("abandons reads after the drain timeout", func(t *testing.T) {
		c := &fakeVisibility{respond: func(ctx context.Context, _ string) error {
			<-ctx.Done()
			return ctx.Err()
		}}
		h := newReaderHarness(t, c, 64)
		h.r.drainTimeout = 50 * time.Millisecond
		h.start(t)
		h.tick(t, 1)
		require.Eventually(t, func() bool { return c.calls() == 1 }, 5*time.Second, 10*time.Millisecond)
		h.cancel()
		requireDone(t, h.done)
		require.Equal(t, int64(1), h.r.abandoned.Load())
		require.Zero(t, metricValue(t, h.registry, ReadErrorsMetric))
	})
}

func (f *fakeVisibility) ListWorkflow(
	ctx context.Context,
	request *workflowservice.ListWorkflowExecutionsRequest,
) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	f.mu.Lock()
	f.lists = append(f.lists, request)
	f.mu.Unlock()
	if err := f.outcome(ctx, kindList); err != nil {
		return nil, err
	}
	return &workflowservice.ListWorkflowExecutionsResponse{}, nil
}

func (f *fakeVisibility) CountWorkflow(
	ctx context.Context,
	request *workflowservice.CountWorkflowExecutionsRequest,
) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	f.mu.Lock()
	f.counts = append(f.counts, request)
	f.mu.Unlock()
	if err := f.outcome(ctx, kindCount); err != nil {
		return nil, err
	}
	return &workflowservice.CountWorkflowExecutionsResponse{}, nil
}

func (f *fakeVisibility) outcome(ctx context.Context, kind string) error {
	if f.respond == nil {
		return nil
	}
	return f.respond(ctx, kind)
}

func (f *fakeVisibility) requests() (
	[]*workflowservice.ListWorkflowExecutionsRequest,
	[]*workflowservice.CountWorkflowExecutionsRequest,
) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]*workflowservice.ListWorkflowExecutionsRequest(nil), f.lists...),
		append([]*workflowservice.CountWorkflowExecutionsRequest(nil), f.counts...)
}

func (f *fakeVisibility) calls() int {
	lists, counts := f.requests()
	return len(lists) + len(counts)
}

func (p *fakePacer) Wait(ctx context.Context) error {
	select {
	case <-p.ticks:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// start runs the reader until the test cancels it or ends.
func (h *readerHarness) start(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	h.cancel = cancel
	go func() {
		defer close(h.done)
		h.r.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-h.done
	})
}

// tick releases n reads, one at a time.
func (h *readerHarness) tick(t *testing.T, n int) {
	t.Helper()
	for range n {
		select {
		case h.pacer.ticks <- struct{}{}:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "reader stopped taking ticks")
		}
	}
}

func testReads() *kitchensink.VisibilityReads {
	return &kitchensink.VisibilityReads{
		RatePerSecond: 10,
		Templates: []*kitchensink.VisibilityReadTemplate{
			{Name: listTemplate, Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_LIST,
				Query: "VisTenant = {VisTenant}", Weight: 50, PageSize: 7},
			{Name: countTemplate, Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_COUNT,
				Query: "StartTime > {ago:15m} AND WorkflowId IN ({wfid:ids:1})", Weight: 50},
		},
	}
}

func newConfig(reads *kitchensink.VisibilityReads, c Client, h client.MetricsHandler) ReaderConfig {
	return ReaderConfig{
		Client:      c,
		Namespace:   "ns",
		Reads:       reads,
		Definitions: testDefs(),
		Workflows:   func() Workflows { return testWorkflows(2) },
		Rand:        rand.New(rand.NewPCG(testSeed, 1)),
		Metrics:     h,
		Logger:      zap.NewNop().Sugar(),
	}
}

func newReaderHarness(t *testing.T, c Client, maxInFlight uint32) *readerHarness {
	t.Helper()
	reads := testReads()
	reads.MaxInFlight = maxInFlight
	return newReaderHarnessFor(t, reads, c)
}

// newReaderHarnessFor builds a reader with a fake pacer and a fixed clock, recording metrics through omes's
// prometheus handler.
func newReaderHarnessFor(t *testing.T, reads *kitchensink.VisibilityReads, c Client) *readerHarness {
	t.Helper()
	m := &metrics.Metrics{Registry: prometheus.NewRegistry(), Cache: map[string]any{}}
	core, logs := observer.New(zapcore.InfoLevel)
	cfg := newConfig(reads, c, m.NewHandler())
	cfg.Logger = zap.New(core).Sugar()
	r, err := NewReader(cfg)
	require.NoError(t, err)
	pc := &fakePacer{ticks: make(chan struct{})}
	r.pacer = pc
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	r.now = func() time.Time { return now }
	return &readerHarness{r: r, pacer: pc, registry: m.Registry, logs: logs, done: make(chan struct{})}
}

// metricValue sums the series of the named counter, or the sample counts of the named histogram, whose labels
// include every name and value pair in labels. It fails the test if the registry has no such metric.
func metricValue(t *testing.T, registry *prometheus.Registry, name string, labels ...string) float64 {
	t.Helper()
	families, err := registry.Gather()
	require.NoError(t, err)
	var registered []string
	for _, family := range families {
		registered = append(registered, family.GetName())
		if family.GetName() != name {
			continue
		}
		total := 0.0
	series:
		for _, m := range family.GetMetric() {
			have := map[string]string{}
			for _, lp := range m.GetLabel() {
				have[lp.GetName()] = lp.GetValue()
			}
			for i := 0; i+1 < len(labels); i += 2 {
				if have[labels[i]] != labels[i+1] {
					continue series
				}
			}
			total += m.GetCounter().GetValue() + float64(m.GetHistogram().GetSampleCount())
		}
		return total
	}
	require.FailNow(t, "metric not registered", "%s; registered: %s", name, strings.Join(registered, ", "))
	return 0
}

func requireDone(t *testing.T, done <-chan struct{}) {
	t.Helper()
	require.Eventually(t, func() bool { return isClosed(done) }, 2*time.Second, 5*time.Millisecond)
}

func isClosed(ch <-chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}

func newTestHandler() client.MetricsHandler {
	m := &metrics.Metrics{Registry: prometheus.NewRegistry(), Cache: map[string]any{}}
	return m.NewHandler()
}
