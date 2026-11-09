package loadgen

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/loadgen/visibility"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
)

// TestReadLoadInputJSON checks that the example file decodes and keeps its two weighted inputs.
func TestReadLoadInputJSON(t *testing.T) {
	in, err := ReadLoadInput(filepath.Join("..", "scenarios", "testdata", "load_input_example.json"))
	require.NoError(t, err)
	require.Len(t, in.GetInputs(), 2)
	_, err = kitchensink.NewMaterializer(in)
	require.NoError(t, err, "the example must be a valid LoadInput")
}

// TestReadLoadInputBinary checks that a binary-encoded LoadInput reads back equal to the source.
func TestReadLoadInputBinary(t *testing.T) {
	src := tenantLoadInput(nil)
	data, err := proto.Marshal(src)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "load_input.pb")
	require.NoError(t, os.WriteFile(path, data, 0o600))

	got, err := ReadLoadInput(path)
	require.NoError(t, err)
	require.True(t, proto.Equal(src, got), "got %v, want %v", got, src)
}

// TestLoadInputRunSetsStartAttributes checks that the started workflow's TypedSearchAttributes hold OmesExecutionID
// and VisTenant.
func TestLoadInputRunSetsStartAttributes(t *testing.T) {
	var (
		mu      sync.Mutex
		started []*KitchenSinkWorkflowOptions
	)
	exec := FuzzExecutor{
		InitLoadInput: loadInputs(tenantLoadInput(nil)),
		executeWorkflow: func(_ context.Context, _ *Run, options *KitchenSinkWorkflowOptions) error {
			mu.Lock()
			defer mu.Unlock()
			started = append(started, options)
			return nil
		},
	}
	require.NoError(t, exec.Run(t.Context(), loadInputInfo(t, 3)))

	require.Len(t, started, 3)
	for _, options := range started {
		attrs := options.StartOptions.TypedSearchAttributes
		execID, ok := attrs.GetString(temporal.NewSearchAttributeKeyString(OmesExecutionIDSearchAttribute))
		require.True(t, ok, "OmesExecutionID missing")
		require.Equal(t, "exec", execID)
		tenant, ok := attrs.GetKeyword(temporal.NewSearchAttributeKeyKeyword("VisTenant"))
		require.True(t, ok, "VisTenant missing")
		require.Regexp(t, regexp.MustCompile(`^VisTenant-\d+$`), tenant)

		upsert := options.Params.GetWorkflowInput().GetInitialActions()[0].GetActions()[0].GetUpsertSearchAttributes()
		require.Empty(t, upsert.GetGenerated(), "the workflow must get the materialized input")
		require.Contains(t, upsert.GetSearchAttributes(), "VisTenant")
	}
}

// TestLoadInputRunWithoutReads checks that a LoadInput with no visibility_reads gets no reader.
func TestLoadInputRunWithoutReads(t *testing.T) {
	l, err := newLoadInputRun(loadInputInfo(t, 2), tenantLoadInput(nil), newCountingVisibility())
	require.NoError(t, err)
	require.Nil(t, l.reader)
}

// TestLoadInputValuesDifferPerExecution checks that omes processes sharing a run ID, but not an execution ID, write
// different values.
func TestLoadInputValuesDifferPerExecution(t *testing.T) {
	tenants := func(executionID string) []string {
		var (
			mu  sync.Mutex
			got = map[int]string{}
		)
		exec := FuzzExecutor{
			InitLoadInput: loadInputs(tenantLoadInput(nil)),
			executeWorkflow: func(_ context.Context, run *Run, options *KitchenSinkWorkflowOptions) error {
				tenant, _ := options.StartOptions.TypedSearchAttributes.GetKeyword(
					temporal.NewSearchAttributeKeyKeyword("VisTenant"))
				mu.Lock()
				defer mu.Unlock()
				got[run.Iteration] = tenant
				return nil
			},
		}
		info := loadInputInfo(t, 5)
		info.ExecutionID = executionID
		require.NoError(t, exec.Run(t.Context(), info))
		var byIteration []string
		for i := 1; i <= 5; i++ {
			byIteration = append(byIteration, got[i])
		}
		return byIteration
	}
	require.NotEqual(t, tenants("exec-a"), tenants("exec-b"))
}

// TestLoadInputRunWithReads checks that reads run while iterations do, and stop once Run returns.
func TestLoadInputRunWithReads(t *testing.T) {
	vis := newCountingVisibility()
	exec := FuzzExecutor{
		InitLoadInput: loadInputs(tenantLoadInput(&kitchensink.VisibilityReads{
			RatePerSecond: 1000,
			Templates: []*kitchensink.VisibilityReadTemplate{
				{Name: "by-tenant", Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_LIST,
					Query: "VisTenant = {VisTenant}", Weight: 1},
				{Name: "count-run", Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_COUNT,
					Query: "WorkflowId STARTS_WITH {wfid:prefix:broad}", Weight: 1},
			},
		})),
		executeWorkflow:  waitForRead(vis),
		visibilityClient: vis,
	}
	require.NoError(t, exec.Run(t.Context(), loadInputInfo(t, 2)))

	afterRun := vis.calls()
	time.Sleep(100 * time.Millisecond)
	require.Equal(t, afterRun, vis.calls(), "reads continued after Run returned")
}

// TestLoadInputReadsDifferPerExecution checks that omes processes sharing a run ID, but not an execution ID, issue
// different reads.
func TestLoadInputReadsDifferPerExecution(t *testing.T) {
	firstQuery := func(executionID string) string {
		vis := newCountingVisibility()
		exec := FuzzExecutor{
			InitLoadInput: loadInputs(tenantLoadInput(&kitchensink.VisibilityReads{
				RatePerSecond: 1000,
				Templates: []*kitchensink.VisibilityReadTemplate{
					{Name: "tenants", Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_COUNT,
						Query: "VisTenant IN ({VisTenant:in:5})", Weight: 1},
				},
			})),
			executeWorkflow:  waitForRead(vis),
			visibilityClient: vis,
		}
		info := loadInputInfo(t, 1)
		info.ExecutionID = executionID
		require.NoError(t, exec.Run(t.Context(), info))
		return vis.firstQuery
	}
	require.NotEqual(t, firstQuery("exec-a"), firstQuery("exec-b"))
}

// TestLoadInputResumedRunReadsStartedIDs checks that a run resumed at a later iteration only reads workflow IDs from
// that iteration on.
func TestLoadInputResumedRunReadsStartedIDs(t *testing.T) {
	vis := newCountingVisibility()
	exec := FuzzExecutor{
		InitLoadInput: loadInputs(tenantLoadInput(&kitchensink.VisibilityReads{
			RatePerSecond: 1000,
			Templates: []*kitchensink.VisibilityReadTemplate{
				{Name: "ids", Kind: kitchensink.VisibilityReadKind_VISIBILITY_READ_KIND_COUNT,
					Query: "WorkflowId IN ({wfid:ids:5})", Weight: 1},
			},
		})),
		executeWorkflow:  waitForRead(vis),
		visibilityClient: vis,
	}
	info := loadInputInfo(t, 51)
	info.Configuration.StartFromIteration = 49
	require.NoError(t, exec.Run(t.Context(), info))

	ids := regexp.MustCompile(`w-run-exec-(\d+)`).FindAllStringSubmatch(vis.firstQuery, -1)
	require.Len(t, ids, 5, "query: %s", vis.firstQuery)
	for _, id := range ids {
		iteration, err := strconv.Atoi(id[1])
		require.NoError(t, err)
		require.GreaterOrEqual(t, iteration, 50, "query: %s", vis.firstQuery)
	}
}

// TestLoadInputRunRejectsUndefinedAttribute checks that a start attribute with no definition fails before any
// workflow starts.
func TestLoadInputRunRejectsUndefinedAttribute(t *testing.T) {
	in := tenantLoadInput(nil)
	in.StartSearchAttributes = append(in.StartSearchAttributes, "VisMissing")
	var executed atomic.Int64
	exec := FuzzExecutor{
		InitLoadInput: loadInputs(in),
		executeWorkflow: func(context.Context, *Run, *KitchenSinkWorkflowOptions) error {
			executed.Add(1)
			return nil
		},
	}
	require.ErrorContains(t, exec.Run(t.Context(), loadInputInfo(t, 1)), "VisMissing")
	require.Zero(t, executed.Load())
}

// TestLoadInputRunRejectsOmesExecutionID checks that a definition or start attribute named OmesExecutionID fails
// before any workflow starts, naming the attribute.
func TestLoadInputRunRejectsOmesExecutionID(t *testing.T) {
	defined := func() *kitchensink.LoadInput {
		in := tenantLoadInput(nil)
		in.SearchAttributes = append(in.SearchAttributes, &kitchensink.SearchAttributeDefinition{
			Name: OmesExecutionIDSearchAttribute,
			Type: enums.INDEXED_VALUE_TYPE_KEYWORD,
			Generator: &kitchensink.ValueGenerator{Variant: &kitchensink.ValueGenerator_Pool{
				Pool: &kitchensink.PoolGenerator{Cardinality: 5},
			}},
		})
		return in
	}
	started := defined()
	started.StartSearchAttributes = append(started.StartSearchAttributes, OmesExecutionIDSearchAttribute)
	tests := map[string]*kitchensink.LoadInput{"defined": defined(), "defined and started": started}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			var executed atomic.Int64
			exec := FuzzExecutor{
				InitLoadInput: loadInputs(in),
				executeWorkflow: func(context.Context, *Run, *KitchenSinkWorkflowOptions) error {
					executed.Add(1)
					return nil
				},
			}
			require.ErrorContains(t, exec.Run(t.Context(), loadInputInfo(t, 1)), OmesExecutionIDSearchAttribute)
			require.Zero(t, executed.Load())
		})
	}
}

// TestVisibilityWorkflowIDMatchesRun checks that visibility.Workflows names iterations the way
// Run.DefaultStartWorkflowOptions does.
func TestVisibilityWorkflowIDMatchesRun(t *testing.T) {
	info := loadInputInfo(t, 1)
	w := visibility.Workflows{RunID: info.RunID, ExecutionID: info.ExecutionID}
	for _, i := range []int{1, 9, 10, 4321} {
		run := &Run{ScenarioInfo: &info, Iteration: i}
		require.Equal(t, run.DefaultStartWorkflowOptions().ID, w.ID(i))
	}
}

// TestLoadInputRetryRepeatsValues checks that a retried iteration starts its workflow with the same generated values.
func TestLoadInputRetryRepeatsValues(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var tenants []string
		exec := FuzzExecutor{
			InitLoadInput: loadInputs(tenantLoadInput(nil)),
			executeWorkflow: func(_ context.Context, _ *Run, options *KitchenSinkWorkflowOptions) error {
				tenant, _ := options.StartOptions.TypedSearchAttributes.GetKeyword(
					temporal.NewSearchAttributeKeyKeyword("VisTenant"))
				tenants = append(tenants, tenant)
				if len(tenants) == 1 {
					return errors.New("transient failure")
				}
				return nil
			},
		}
		info := loadInputInfo(t, 1)
		info.Configuration.MaxIterationAttempts = 2
		require.NoError(t, exec.Run(t.Context(), info))
		require.Len(t, tenants, 2)
		require.Equal(t, tenants[0], tenants[1])
	})
}

type countingVisibility struct {
	list, count atomic.Int64
	once        sync.Once
	firstRead   chan struct{}
	// firstQuery is set before firstRead closes.
	firstQuery string
}

func newCountingVisibility() *countingVisibility {
	return &countingVisibility{firstRead: make(chan struct{})}
}

func (c *countingVisibility) ListWorkflow(
	_ context.Context,
	req *workflowservice.ListWorkflowExecutionsRequest,
) (*workflowservice.ListWorkflowExecutionsResponse, error) {
	c.list.Add(1)
	c.recordFirst(req.GetQuery())
	return &workflowservice.ListWorkflowExecutionsResponse{}, nil
}

func (c *countingVisibility) CountWorkflow(
	_ context.Context,
	req *workflowservice.CountWorkflowExecutionsRequest,
) (*workflowservice.CountWorkflowExecutionsResponse, error) {
	c.count.Add(1)
	c.recordFirst(req.GetQuery())
	return &workflowservice.CountWorkflowExecutionsResponse{}, nil
}

func (c *countingVisibility) recordFirst(query string) {
	c.once.Do(func() {
		c.firstQuery = query
		close(c.firstRead)
	})
}

func (c *countingVisibility) calls() int64 {
	return c.list.Load() + c.count.Load()
}

// waitForRead holds each iteration until the reader has issued its first read.
func waitForRead(vis *countingVisibility) func(context.Context, *Run, *KitchenSinkWorkflowOptions) error {
	return func(ctx context.Context, _ *Run, _ *KitchenSinkWorkflowOptions) error {
		select {
		case <-vis.firstRead:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Second):
			return errors.New("no read was issued while the iteration ran")
		}
	}
}

// tenantLoadInput sets VisTenant at start and upserts it in the workflow's first action.
func tenantLoadInput(reads *kitchensink.VisibilityReads) *kitchensink.LoadInput {
	return &kitchensink.LoadInput{
		Inputs: []*kitchensink.WeightedTestInput{{
			Weight: 1,
			Input: &kitchensink.TestInput{WorkflowInput: &kitchensink.WorkflowInput{
				InitialActions: []*kitchensink.ActionSet{{Actions: []*kitchensink.Action{
					{Variant: &kitchensink.Action_UpsertSearchAttributes{
						UpsertSearchAttributes: &kitchensink.UpsertSearchAttributesAction{Generated: []string{"VisTenant"}},
					}},
					kitchensink.NewEmptyReturnResultAction(),
				}}},
			}},
		}},
		SearchAttributes: []*kitchensink.SearchAttributeDefinition{{
			Name: "VisTenant",
			Type: enums.INDEXED_VALUE_TYPE_KEYWORD,
			Generator: &kitchensink.ValueGenerator{Variant: &kitchensink.ValueGenerator_Pool{
				Pool: &kitchensink.PoolGenerator{Cardinality: 1000},
			}},
		}},
		StartSearchAttributes: []string{"VisTenant"},
		VisibilityReads:       reads,
	}
}

func loadInputs(in *kitchensink.LoadInput) func(context.Context, ScenarioInfo) (*kitchensink.LoadInput, error) {
	return func(context.Context, ScenarioInfo) (*kitchensink.LoadInput, error) {
		return in, nil
	}
}

// loadInputInfo names a Nexus endpoint, so the run doesn't try to create one without a client.
func loadInputInfo(t *testing.T, iterations int) ScenarioInfo {
	return ScenarioInfo{
		RunID:          "run",
		ExecutionID:    "exec",
		Namespace:      "ns",
		MetricsHandler: client.MetricsNopHandler,
		Logger:         zaptest.NewLogger(t, zaptest.Level(zap.WarnLevel)).Sugar(),
		Configuration:  RunConfiguration{Iterations: iterations},
		Options: (&Scenario{Options: DeclareFuzzExecutorOptions}).MustResolveOptions(map[string]string{
			NexusEndpointOption: "endpoint",
		}),
	}
}
