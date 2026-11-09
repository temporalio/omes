package loadgen

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/temporalio/omes/loadgen/kitchensink"
	"github.com/temporalio/omes/loadgen/visibility"
	"go.temporal.io/sdk/temporal"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// loadInputRun replays a LoadInput: each iteration materializes one of its inputs, and an optional reader issues
// visibility reads beside the iterations.
type loadInputRun struct {
	materializer *kitchensink.Materializer
	defs         []*kitchensink.SearchAttributeDefinition
	// seed is the first half of every PCG seed in this execution. The second half is the iteration number, or
	// readerStream for the reader.
	seed uint64
	// reader is nil when the LoadInput has no visibility_reads.
	reader *visibility.Reader
	// started is one past the highest iteration whose workflow options were built. Reads may name an iteration's
	// workflow just before it starts.
	started atomic.Int64
}

// readerStream is the second PCG seed half for the reader. Iterations count up from 1 and never reach it.
const readerStream = math.MaxUint64

// ReadLoadInput reads protojson from a path ending in .json, and binary proto from any other path.
func ReadLoadInput(path string) (*kitchensink.LoadInput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read load input: %w", err)
	}
	in := &kitchensink.LoadInput{}
	if strings.HasSuffix(path, ".json") {
		err = protojson.Unmarshal(data, in)
	} else {
		err = proto.Unmarshal(data, in)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode load input %s: %w", path, err)
	}
	return in, nil
}

// newLoadInputRun validates the LoadInput, including its read templates, so a bad input fails before anything starts.
func newLoadInputRun(info ScenarioInfo, in *kitchensink.LoadInput, vis visibility.Client) (*loadInputRun, error) {
	reserved := slices.Contains(in.GetStartSearchAttributes(), OmesExecutionIDSearchAttribute)
	for _, def := range in.GetSearchAttributes() {
		reserved = reserved || def.GetName() == OmesExecutionIDSearchAttribute
	}
	if reserved {
		return nil, fmt.Errorf("invalid load input: search attribute %q is set by omes and can't be defined",
			OmesExecutionIDSearchAttribute)
	}
	m, err := kitchensink.NewMaterializer(in)
	if err != nil {
		return nil, fmt.Errorf("invalid load input: %w", err)
	}
	l := &loadInputRun{
		materializer: m,
		defs:         in.GetSearchAttributes(),
		// Processes started with the same run ID have different execution IDs, so each writes its own values.
		seed: fnv64a(info.RunID + "\x00" + info.ExecutionID),
	}
	if in.GetVisibilityReads() == nil {
		return l, nil
	}
	defs := make(map[string]*kitchensink.SearchAttributeDefinition, len(l.defs))
	for _, def := range l.defs {
		defs[def.GetName()] = def
	}
	// GenericExecutor numbers iterations from StartFromIteration+1.
	first := info.Configuration.StartFromIteration + 1
	l.reader, err = visibility.NewReader(visibility.ReaderConfig{
		Client:      vis,
		Namespace:   info.Namespace,
		Reads:       in.GetVisibilityReads(),
		Definitions: defs,
		Workflows: func() visibility.Workflows {
			return visibility.Workflows{
				RunID:       info.RunID,
				ExecutionID: info.ExecutionID,
				First:       first,
				Started:     int(l.started.Load()),
			}
		},
		Rand:    rand.New(rand.NewPCG(l.seed, readerStream)),
		Metrics: info.MetricsHandler,
		Logger:  info.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("invalid load input: %w", err)
	}
	return l, nil
}

// run runs the iterations, and the reader beside them until they end.
func (l *loadInputRun) run(ctx context.Context, info ScenarioInfo, ge *GenericExecutor) error {
	if l.reader == nil {
		return ge.Run(ctx, info)
	}
	readCtx, stopReads := context.WithCancel(ctx)
	var reads sync.WaitGroup
	reads.Go(func() { l.reader.Run(readCtx) })
	err := ge.Run(ctx, info)
	stopReads()
	reads.Wait()
	return err
}

// options materializes the run's iteration. The random source depends only on the run ID, execution ID, and
// iteration, so a retried iteration writes the same values, apart from Datetime values, which depend on the clock.
func (l *loadInputRun) options(run *Run) (KitchenSinkWorkflowOptions, error) {
	options := run.DefaultKitchenSinkWorkflowOptions()
	r := rand.New(rand.NewPCG(l.seed, uint64(run.Iteration)))
	params, startAttrs, err := l.materializer.Iteration(r, time.Now())
	if err != nil {
		return options, err
	}
	options.Params = params
	options.StartOptions.TypedSearchAttributes = temporal.NewSearchAttributes(
		options.StartOptions.TypedSearchAttributes.Copy(),
		startAttrs.Copy(),
	)
	l.markStarted(run.Iteration)
	return options, nil
}

// markStarted records that iteration has started. Iterations start concurrently and out of order, so started only
// ever moves up.
func (l *loadInputRun) markStarted(iteration int) {
	next := int64(iteration) + 1
	for {
		cur := l.started.Load()
		if next <= cur || l.started.CompareAndSwap(cur, next) {
			return
		}
	}
}

func fnv64a(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
