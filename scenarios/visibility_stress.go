package scenarios

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen"
	vstypes "github.com/temporalio/omes/loadgen/visibility"
	commonpb "go.temporal.io/api/common/v1"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"golang.org/x/time/rate"
	"google.golang.org/protobuf/types/known/durationpb"
)

// ---------------------------------------------------------------------------
// Presets
// ---------------------------------------------------------------------------

type vsLoadPreset struct {
	WfRPS          float64
	UpdatesPerWF   float64
	UpdateDelay    time.Duration
	DeleteRPS      float64
	FailPercent    float64
	TimeoutPercent float64
	// TODO: MemoUpdatesPerWF float64
	// TODO: MemoSizeBytes    int
}

type vsQueryPreset struct {
	CountRPS              float64
	ListRPS               float64
	ListNoFilterWeight    int
	ListOpenWeight        int
	ListClosedWeight      int
	ListSimpleCSAWeight   int
	ListCompoundCSAWeight int
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
		CountRPS: 5, ListRPS: 10,
		ListNoFilterWeight: 3, ListOpenWeight: 2, ListClosedWeight: 2,
		ListSimpleCSAWeight: 2, ListCompoundCSAWeight: 1,
	},
	"read-heavy": {
		CountRPS: 50, ListRPS: 200,
		ListNoFilterWeight: 1, ListOpenWeight: 2, ListClosedWeight: 2,
		ListSimpleCSAWeight: 3, ListCompoundCSAWeight: 2,
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
// CSA Definition
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

func randomCSAValue(c csaDef, rng *rand.Rand) any {
	switch c.Type {
	case csaTypeInt:
		return float64(rng.Intn(1001))
	case csaTypeKeyword:
		return keywordVocabulary[rng.Intn(len(keywordVocabulary))]
	case csaTypeBool:
		return rng.Intn(2) == 1
	case csaTypeDouble:
		return rng.Float64() * 1000.0
	case csaTypeText:
		length := 10 + rng.Intn(41)
		b := make([]byte, length)
		const chars = "abcdefghijklmnopqrstuvwxyz0123456789 "
		for i := range b {
			b[i] = chars[rng.Intn(len(chars))]
		}
		return string(b)
	case csaTypeDatetime:
		offset := time.Duration(rng.Int63n(int64(30 * 24 * time.Hour)))
		return time.Now().Add(-offset).UTC().Format(time.RFC3339)
	default:
		return nil
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
	rng         *rand.Rand

	totalCreated atomic.Int64
	totalDeleted atomic.Int64
	totalQueries atomic.Int64
	totalErrors  atomic.Int64
	wfCounter    atomic.Uint64
}

var _ loadgen.Configurable = (*visibilityStressExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Visibility store stress test.\n" +
			"Options: loadPreset, queryPreset, csaPreset, namespaceCount, createNamespaces, retention, cleanup.\n" +
			"Duration must be set. At least one of loadPreset or queryPreset required.",
		ExecutorFn: func() loadgen.Executor { return &visibilityStressExecutor{} },
	})
}

func (e *visibilityStressExecutor) Configure(info loadgen.ScenarioInfo) error {
	cfg := &vsConfig{
		NamespaceCount:   info.ScenarioOptionInt("namespaceCount", 1),
		CreateNamespaces: info.ScenarioOptionBool("createNamespaces", false),
		Cleanup:          info.ScenarioOptionBool("cleanup", false),
		DeleteNamespaces: info.ScenarioOptionBool("deleteNamespaces", false),
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

func (e *visibilityStressExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := e.Configure(info); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	e.taskQueue = loadgen.TaskQueueForRun(info.RunID)
	e.executionID = info.ExecutionID
	e.rng = rand.New(rand.NewSource(time.Now().UnixNano()))

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

	if err := e.registerCSAs(ctx, info); err != nil {
		return err
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
	info.Logger.Infof("CSAs: %d total", len(e.config.CSADefs))

	if l := e.config.Load; l != nil {
		info.Logger.Infof("Write: wfRPS=%.1f, updatesPerWF=%.1f, effective CSA update RPS≈%.0f, deleteRPS=%.1f",
			l.WfRPS, l.UpdatesPerWF, l.WfRPS*l.UpdatesPerWF, l.DeleteRPS)
		info.Logger.Infof("       failPercent=%.2f, timeoutPercent=%.2f, updateDelay=%v",
			l.FailPercent, l.TimeoutPercent, l.UpdateDelay)
	}
	if q := e.config.Query; q != nil {
		info.Logger.Infof("Read: countRPS=%.1f, listRPS=%.1f", q.CountRPS, q.ListRPS)
	}
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

	info.Logger.Infof("Run complete. Created: %d, Deleted: %d, Queries: %d, Errors: %d",
		e.totalCreated.Load(), e.totalDeleted.Load(), e.totalQueries.Load(), e.totalErrors.Load())
	return nil
}

func (e *visibilityStressExecutor) runWriter(ctx context.Context, info loadgen.ScenarioInfo) {
	limiter := rate.NewLimiter(rate.Limit(e.config.Load.WfRPS), 1)
	var nsIndex int
	startTime := time.Now()
	var tickCount int64

	for {
		if err := limiter.Wait(ctx); err != nil {
			return
		}

		nsIdx := nsIndex % len(e.namespaces)
		nsIndex++

		input := e.buildWorkflowInput()
		wfID := fmt.Sprintf("vs-%s-%s-%d", info.RunID, e.executionID, e.wfCounter.Add(1))

		opts := client.StartWorkflowOptions{
			ID:                       wfID,
			TaskQueue:                e.taskQueue,
			WorkflowExecutionTimeout: vsComputeTimeout(len(input.CSAUpdates)),
		}

		// Fire and forget.
		// TODO: better error handling (retry? circuit breaker?)
		_, err := e.clients[nsIdx].ExecuteWorkflow(ctx, opts, "visibilityStressWorker", input)
		if err != nil {
			e.totalErrors.Add(1)
			info.Logger.Warnf("Failed to start workflow: %v", err)
			continue
		}

		e.totalCreated.Add(1)
		tickCount++

		logEvery := int64(math.Max(1, e.config.Load.WfRPS))
		if tickCount%logEvery == 0 {
			elapsed := time.Since(startTime)
			info.Logger.Infof("[writer] t=%v created=%d errors=%d actual_rps=%.1f",
				elapsed.Round(time.Second), e.totalCreated.Load(),
				e.totalErrors.Load(), float64(e.totalCreated.Load())/elapsed.Seconds())
		}
	}
}

func (e *visibilityStressExecutor) runDeleters(ctx context.Context, info loadgen.ScenarioInfo) {
	perNsDeleteRPS := e.config.Load.DeleteRPS / float64(len(e.namespaces))
	for i, ns := range e.namespaces {
		i, ns := i, ns
		go e.runDeleterForNamespace(ctx, info, i, ns, perNsDeleteRPS)
	}
}

func (e *visibilityStressExecutor) runDeleterForNamespace(
	ctx context.Context, info loadgen.ScenarioInfo,
	nsIdx int, ns string, deleteRPS float64,
) {
	if deleteRPS <= 0 {
		return
	}

	tickInterval := time.Second
	batchSize := int32(math.Max(1, deleteRPS))
	if deleteRPS < 1 {
		tickInterval = time.Duration(float64(time.Second) / deleteRPS)
		batchSize = 1
	}

	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	query := fmt.Sprintf(
		"WorkflowType = 'visibilityStressWorker' AND ExecutionStatus != 'Running' AND TaskQueue = '%s'",
		e.taskQueue)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		resp, err := e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
			Namespace: ns,
			Query:     query,
			PageSize:  batchSize,
		})
		if err != nil {
			// TODO: better error handling
			info.Logger.Warnf("[deleter/%s] List failed: %v", ns, err)
			continue
		}

		for _, exec := range resp.Executions {
			wfExec := exec.Execution
			_, err := e.clients[nsIdx].WorkflowService().DeleteWorkflowExecution(ctx,
				&workflowservice.DeleteWorkflowExecutionRequest{
					Namespace: ns,
					WorkflowExecution: &commonpb.WorkflowExecution{
						WorkflowId: wfExec.WorkflowId,
						RunId:      wfExec.RunId,
					},
				})
			if err != nil {
				// TODO: better error handling
				info.Logger.Warnf("[deleter/%s] Delete %s failed: %v", ns, wfExec.WorkflowId, err)
				continue
			}
			e.totalDeleted.Add(1)
		}
	}
}

func (e *visibilityStressExecutor) runQuerier(ctx context.Context, info loadgen.ScenarioInfo) {
	q := e.config.Query
	totalRPS := q.CountRPS + q.ListRPS
	if totalRPS <= 0 {
		return
	}

	limiter := rate.NewLimiter(rate.Limit(totalRPS), 1)
	var nsIndex int

	totalWeight := q.ListNoFilterWeight + q.ListOpenWeight + q.ListClosedWeight +
		q.ListSimpleCSAWeight + q.ListCompoundCSAWeight

	for {
		if err := limiter.Wait(ctx); err != nil {
			return
		}

		nsIdx := nsIndex % len(e.namespaces)
		nsIndex++
		ns := e.namespaces[nsIdx]

		isCount := e.rng.Float64() < q.CountRPS/totalRPS
		filter := e.generateFilter(isCount, totalWeight)

		if isCount {
			_, err := e.clients[nsIdx].CountWorkflow(ctx, &workflowservice.CountWorkflowExecutionsRequest{
				Namespace: ns,
				Query:     filter,
			})
			if err != nil {
				// TODO: better error handling
				info.Logger.Warnf("[querier] Count failed: %v", err)
			}
		} else {
			resp, err := e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
				Namespace: ns,
				Query:     filter,
			})
			if err != nil {
				// TODO: better error handling
				info.Logger.Warnf("[querier] List failed: %v", err)
			} else if len(resp.NextPageToken) > 0 {
				resp2, err := e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
					Namespace:     ns,
					Query:         filter,
					NextPageToken: resp.NextPageToken,
				})
				if err == nil && len(resp2.NextPageToken) > 0 {
					_, _ = e.clients[nsIdx].ListWorkflow(ctx, &workflowservice.ListWorkflowExecutionsRequest{
						Namespace:     ns,
						Query:         filter,
						NextPageToken: resp2.NextPageToken,
					})
				}
			}
		}
		e.totalQueries.Add(1)
	}
}

// ---------------------------------------------------------------------------
// Query Generation
// ---------------------------------------------------------------------------

func (e *visibilityStressExecutor) generateFilter(isCount bool, totalWeight int) string {
	if isCount {
		switch e.rng.Intn(3) {
		case 0:
			return "WorkflowType = 'visibilityStressWorker'"
		case 1:
			return "ExecutionStatus = 'Running'"
		default:
			kwCSAs := e.csasByType(csaTypeKeyword)
			if len(kwCSAs) > 0 {
				csa := kwCSAs[e.rng.Intn(len(kwCSAs))]
				val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
				return fmt.Sprintf("%s = '%s'", csa.Name, val)
			}
			return "WorkflowType = 'visibilityStressWorker'"
		}
	}

	q := e.config.Query
	roll := e.rng.Intn(totalWeight)
	cumulative := 0

	cumulative += q.ListNoFilterWeight
	if roll < cumulative {
		return "WorkflowType = 'visibilityStressWorker'"
	}

	cumulative += q.ListOpenWeight
	if roll < cumulative {
		return "WorkflowType = 'visibilityStressWorker' AND ExecutionStatus = 'Running'"
	}

	cumulative += q.ListClosedWeight
	if roll < cumulative {
		since := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
		return fmt.Sprintf("ExecutionStatus != 'Running' AND CloseTime > '%s'", since)
	}

	cumulative += q.ListSimpleCSAWeight
	if roll < cumulative {
		return e.generateSimpleCSAFilter()
	}

	return e.generateCompoundCSAFilter()
}

func (e *visibilityStressExecutor) generateSimpleCSAFilter() string {
	csa := e.config.CSADefs[e.rng.Intn(len(e.config.CSADefs))]
	return e.csaFilterClause(csa)
}

func (e *visibilityStressExecutor) generateCompoundCSAFilter() string {
	n := 2 + e.rng.Intn(2)
	if n > len(e.config.CSADefs) {
		n = len(e.config.CSADefs)
	}
	perm := e.rng.Perm(len(e.config.CSADefs))
	clauses := make([]string, n)
	for i := 0; i < n; i++ {
		clauses[i] = e.csaFilterClause(e.config.CSADefs[perm[i]])
	}
	return strings.Join(clauses, " AND ")
}

func (e *visibilityStressExecutor) csaFilterClause(csa csaDef) string {
	switch csa.Type {
	case csaTypeInt:
		v := e.rng.Intn(1001)
		if e.rng.Intn(2) == 0 {
			return fmt.Sprintf("%s > %d", csa.Name, v)
		}
		lo, hi := e.rng.Intn(500), 500+e.rng.Intn(501)
		return fmt.Sprintf("%s > %d AND %s < %d", csa.Name, lo, csa.Name, hi)
	case csaTypeKeyword:
		val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
		return fmt.Sprintf("%s = '%s'", csa.Name, val)
	case csaTypeBool:
		if e.rng.Intn(2) == 0 {
			return fmt.Sprintf("%s = true", csa.Name)
		}
		return fmt.Sprintf("%s = false", csa.Name)
	case csaTypeDouble:
		v := e.rng.Float64() * 1000.0
		return fmt.Sprintf("%s > %f", csa.Name, v)
	case csaTypeText:
		val := keywordVocabulary[e.rng.Intn(len(keywordVocabulary))]
		return fmt.Sprintf("%s = '%s'", csa.Name, val)
	case csaTypeDatetime:
		offset := time.Duration(e.rng.Int63n(int64(30 * 24 * time.Hour)))
		t := time.Now().Add(-offset).UTC().Format(time.RFC3339)
		return fmt.Sprintf("%s > '%s'", csa.Name, t)
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

func (e *visibilityStressExecutor) buildWorkflowInput() *vstypes.VisibilityWorkerInput {
	cfg := e.config.Load

	wholeUpdates := int(cfg.UpdatesPerWF)
	fraction := cfg.UpdatesPerWF - float64(wholeUpdates)
	numUpdates := wholeUpdates
	if e.rng.Float64() < fraction {
		numUpdates++
	}

	roll := e.rng.Float64()
	shouldFail := roll < cfg.FailPercent
	shouldTimeout := !shouldFail && roll < cfg.FailPercent+cfg.TimeoutPercent

	groups := make([]vstypes.CSAUpdateGroup, 0, numUpdates)
	shuffled := make([]csaDef, len(e.config.CSADefs))
	copy(shuffled, e.config.CSADefs)
	e.rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })

	for i := 0; i < numUpdates; i++ {
		groupSize := 1 + e.rng.Intn(3)
		attrs := make(map[string]any, groupSize)
		for j := 0; j < groupSize; j++ {
			csa := shuffled[(i*3+j)%len(shuffled)]
			attrs[csa.Name] = randomCSAValue(csa, e.rng)
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
