package visibility

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	ReadLatencyMetric  = "omes_visibility_read_latency_seconds"
	ReadsMetric        = "omes_visibility_reads_total"
	ReadErrorsMetric   = "omes_visibility_read_errors_total"
	ReadsSkippedMetric = "omes_visibility_reads_skipped_total"

	tagTemplate  = "template"
	tagKind      = "kind"
	tagError     = "error"
	tagNamespace = "namespace"

	kindList  = "list"
	kindCount = "count"

	errorClassTimeout = "timeout"
	errorClassOther   = "other"

	readTimeout = 10 * time.Second
	// readDrainTimeout bounds how long Run waits, once its context ends, for the reads still in flight.
	readDrainTimeout     = 5 * time.Second
	readErrorLogInterval = 30 * time.Second

	minReadsInFlight = 64
	defaultPageSize  = 100
)

type (
	// Client is the part of client.Client the reader calls.
	Client interface {
		ListWorkflow(
			ctx context.Context,
			request *workflowservice.ListWorkflowExecutionsRequest,
		) (*workflowservice.ListWorkflowExecutionsResponse, error)
		CountWorkflow(
			ctx context.Context,
			request *workflowservice.CountWorkflowExecutionsRequest,
		) (*workflowservice.CountWorkflowExecutionsResponse, error)
	}

	ReaderConfig struct {
		Client      Client
		Namespace   string
		Reads       *kitchensink.VisibilityReads
		Definitions map[string]*kitchensink.SearchAttributeDefinition
		// Workflows names the workflows this execution has started, for queries that filter on their IDs.
		Workflows func() Workflows
		// Rand picks templates and placeholder values. Only Run's goroutine uses it. Seed it differently in each
		// process, so processes don't issue the same reads in lockstep.
		Rand    *rand.Rand
		Metrics client.MetricsHandler
		Logger  *zap.SugaredLogger
	}

	// Reader issues visibility reads open-loop: one per pacer tick, each in its own goroutine, so a slow cluster
	// doesn't lower the offered rate. A tick that finds the in-flight cap reached is skipped and counted instead.
	Reader struct {
		client       Client
		namespace    string
		defs         map[string]*kitchensink.SearchAttributeDefinition
		pacer        pacer
		templates    []readTemplate
		cumulative   []int
		rng          *rand.Rand
		now          func() time.Time
		workflows    func() Workflows
		logger       *zap.SugaredLogger
		slots        chan struct{}
		readTimeout  time.Duration
		drainTimeout time.Duration

		completed atomic.Int64
		failed    atomic.Int64
		skipped   atomic.Int64
		abandoned atomic.Int64

		mu sync.Mutex
		// errorLogs holds one entry per template; mu guards the entries.
		errorLogs map[string]*errorLog
	}

	// pacer blocks until the next read is due. *rate.Limiter is one.
	pacer interface {
		Wait(ctx context.Context) error
	}

	readTemplate struct {
		name     string
		kind     string
		query    string
		pageSize int32
		metrics  templateMetrics
	}

	templateMetrics struct {
		latency client.MetricsTimer
		reads   client.MetricsCounter
		skipped client.MetricsCounter
		errors  map[string]client.MetricsCounter
	}

	// errorLog rate-limits one template's error logs.
	errorLog struct {
		last       time.Time
		suppressed int
	}
)

// NewReader checks every template against the definitions and registers each template's metrics up front, so every
// series is exported from the start of the run rather than from its first event.
func NewReader(cfg ReaderConfig) (*Reader, error) {
	if cfg.Reads == nil || len(cfg.Reads.Templates) == 0 {
		return nil, errors.New("visibility reads need at least one template")
	}
	if cfg.Rand == nil {
		return nil, errors.New("visibility reads need a random source")
	}
	ratePerSecond := cfg.Reads.RatePerSecond
	if !(ratePerSecond > 0) || math.IsInf(ratePerSecond, 0) {
		return nil, fmt.Errorf("visibility read rate must be positive, got %v", ratePerSecond)
	}
	maxInFlight := int(cfg.Reads.MaxInFlight)
	if maxInFlight == 0 {
		maxInFlight = max(minReadsInFlight, int(math.Ceil(ratePerSecond)))
	}
	r := &Reader{
		client:       cfg.Client,
		namespace:    cfg.Namespace,
		defs:         cfg.Definitions,
		pacer:        rate.NewLimiter(rate.Limit(ratePerSecond), 1),
		rng:          cfg.Rand,
		now:          time.Now,
		workflows:    cfg.Workflows,
		logger:       cfg.Logger,
		slots:        make(chan struct{}, maxInFlight),
		readTimeout:  readTimeout,
		drainTimeout: readDrainTimeout,
		errorLogs:    make(map[string]*errorLog, len(cfg.Reads.Templates)),
	}
	total := 0
	for _, t := range cfg.Reads.Templates {
		if t.Weight == 0 {
			return nil, fmt.Errorf("read template %q has no weight", t.Name)
		}
		if _, dup := r.errorLogs[t.Name]; dup {
			return nil, fmt.Errorf("duplicate read template name %q", t.Name)
		}
		var kind string
		switch t.Kind {
		case kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_LIST:
			kind = kindList
		case kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_COUNT:
			kind = kindCount
		default:
			return nil, fmt.Errorf("read template %q has unknown kind %v", t.Name, t.Kind)
		}
		if err := checkQuery(t.Query, cfg.Definitions); err != nil {
			return nil, fmt.Errorf("read template %q: %w", t.Name, err)
		}
		if t.PageSize > math.MaxInt32 {
			return nil, fmt.Errorf("read template %q has page size %d above %d", t.Name, t.PageSize, math.MaxInt32)
		}
		pageSize := int32(t.PageSize)
		if pageSize == 0 {
			pageSize = defaultPageSize
		}
		total += int(t.Weight)
		r.cumulative = append(r.cumulative, total)
		r.templates = append(r.templates, readTemplate{
			name: t.Name, kind: kind, query: t.Query, pageSize: pageSize,
			metrics: newTemplateMetrics(cfg.Metrics, t.Name, kind, cfg.Namespace),
		})
		r.errorLogs[t.Name] = &errorLog{}
	}
	return r, nil
}

// Run issues reads until ctx is done, then waits up to the drain timeout for the reads still in flight.
func (r *Reader) Run(ctx context.Context) {
	// Reads outlive ctx so the ones in flight when the load ends can finish; abandon cuts off the stragglers.
	readsCtx, abandon := context.WithCancel(context.WithoutCancel(ctx))
	defer abandon()
	var inFlight sync.WaitGroup
	for ctx.Err() == nil && r.pacer.Wait(ctx) == nil {
		r.tick(readsCtx, &inFlight)
	}

	drained := make(chan struct{})
	go func() {
		inFlight.Wait()
		close(drained)
	}()
	timer := time.NewTimer(r.drainTimeout)
	defer timer.Stop()
	select {
	case <-drained:
	case <-timer.C:
		abandon()
		<-drained
	}
	r.logger.Infof("Visibility reads finished: %d completed, %d failed, %d skipped at the in-flight cap of %d, "+
		"%d abandoned after the %s drain timeout", r.completed.Load(), r.failed.Load(), r.skipped.Load(),
		cap(r.slots), r.abandoned.Load(), r.drainTimeout)
}

func (r *Reader) tick(ctx context.Context, inFlight *sync.WaitGroup) {
	t := r.pick()
	query, err := render(t.query, newValues(r.defs, r.rng, r.now(), r.workflows()))
	if err != nil {
		r.fail(t, errorClassOther, fmt.Errorf("render query: %w", err))
		return
	}
	select {
	case r.slots <- struct{}{}:
	default:
		// Blocking here would quietly lower the offered rate, so the tick is dropped and counted instead.
		t.metrics.skipped.Inc(1)
		r.skipped.Add(1)
		return
	}
	inFlight.Go(func() {
		defer func() { <-r.slots }()
		r.read(ctx, t, query)
	})
}

func (r *Reader) pick() *readTemplate {
	n := r.rng.IntN(r.cumulative[len(r.cumulative)-1])
	// The first template whose cumulative weight exceeds n.
	i, _ := slices.BinarySearch(r.cumulative, n+1)
	return &r.templates[i]
}

func (r *Reader) read(ctx context.Context, t *readTemplate, query string) {
	readCtx, cancel := context.WithTimeout(ctx, r.readTimeout)
	defer cancel()
	start := time.Now()
	var err error
	switch t.kind {
	case kindList:
		_, err = r.client.ListWorkflow(readCtx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: r.namespace,
			PageSize:  t.pageSize,
			Query:     query,
		})
	case kindCount:
		_, err = r.client.CountWorkflow(readCtx, &workflowservice.CountWorkflowExecutionsRequest{
			Namespace: r.namespace,
			Query:     query,
		})
	default:
		err = fmt.Errorf("unknown read kind %q", t.kind)
	}
	elapsed := time.Since(start)

	var deadlineExceeded *serviceerror.DeadlineExceeded
	switch {
	case err == nil:
		t.metrics.latency.Record(elapsed)
		t.metrics.reads.Inc(1)
		r.completed.Add(1)
	case ctx.Err() != nil:
		// Run cut this read off after its drain timeout, so the cluster didn't fail it.
		r.abandoned.Add(1)
	case errors.Is(readCtx.Err(), context.DeadlineExceeded) || errors.As(err, &deadlineExceeded):
		r.fail(t, errorClassTimeout, err)
	default:
		r.fail(t, errorClassOther, err)
	}
}

// fail counts a failed read and logs it, at most once per template per readErrorLogInterval so a failing cluster
// doesn't flood the log.
func (r *Reader) fail(t *readTemplate, class string, err error) {
	t.metrics.errors[class].Inc(1)
	r.failed.Add(1)
	now := r.now()
	r.mu.Lock()
	l := r.errorLogs[t.name]
	if !l.last.IsZero() && now.Sub(l.last) < readErrorLogInterval {
		l.suppressed++
		r.mu.Unlock()
		return
	}
	suppressed := l.suppressed
	l.last, l.suppressed = now, 0
	r.mu.Unlock()
	r.logger.Warnf("Visibility read %q (%s) failed with a %s error, %d more since its last log: %v", t.name, t.kind,
		class, suppressed, err)
}

func newTemplateMetrics(handler client.MetricsHandler, name, kind, namespace string) templateMetrics {
	h := handler.WithTags(map[string]string{tagTemplate: name, tagKind: kind, tagNamespace: namespace})
	return templateMetrics{
		latency: h.Timer(ReadLatencyMetric),
		reads:   h.Counter(ReadsMetric),
		skipped: h.Counter(ReadsSkippedMetric),
		errors: map[string]client.MetricsCounter{
			errorClassTimeout: h.WithTags(map[string]string{tagError: errorClassTimeout}).Counter(ReadErrorsMetric),
			errorClassOther:   h.WithTags(map[string]string{tagError: errorClassOther}).Counter(ReadErrorsMetric),
		},
	}
}
