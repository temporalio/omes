package scenarios

import (
	"context"
	"fmt"
	"hash/fnv"
	"math/rand"

	"github.com/temporalio/omes/loadgen"
	. "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/temporal"
)

const (
	bandwidthActivitiesPerWorkflowFlag = "activities-per-workflow"
	bandwidthPayloadDistributionFlag   = "payload-distribution-json"
	bandwidthDefaultPayloadSize        = 100 * 1024
	bandwidthDefaultPayloadConfig      = `{"size":{"type":"fixed","value":"102400"}}`
)

type bandwidthStressExecutor struct {
	activitiesPerWorkflow int
	payload               *loadgen.PayloadConfig
	rngSeed               int64
}

var _ loadgen.Configurable = (*bandwidthStressExecutor)(nil)

func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Replays sustained variable payload writes for bandwidth limit and storage calibration.",
		Options: func(o *loadgen.OptionSet) {
			o.Int(bandwidthActivitiesPerWorkflowFlag, 1, "Remote payload activities per workflow.")
			o.String(
				bandwidthPayloadDistributionFlag,
				bandwidthDefaultPayloadConfig,
				"JSON activity payload size distribution; use @<file> to read from a file.",
			)
		},
		ExecutorFn: func() loadgen.Executor { return &bandwidthStressExecutor{} },
	})
}

func (b *bandwidthStressExecutor) Configure(info loadgen.ScenarioInfo) error {
	b.activitiesPerWorkflow = info.OptionInt(bandwidthActivitiesPerWorkflowFlag)
	if b.activitiesPerWorkflow <= 0 {
		return fmt.Errorf(
			"%s must be positive, got %d",
			bandwidthActivitiesPerWorkflowFlag,
			b.activitiesPerWorkflow,
		)
	}

	payload, err := loadgen.ParseAndValidatePayloadConfig(info.OptionString(bandwidthPayloadDistributionFlag))
	if err != nil {
		return fmt.Errorf("invalid %s: %w", bandwidthPayloadDistributionFlag, err)
	}
	if payload == nil || payload.Size == nil {
		return fmt.Errorf("%s must configure a size distribution", bandwidthPayloadDistributionFlag)
	}
	b.payload = payload

	h := fnv.New64a()
	_, _ = h.Write([]byte(info.RunID))
	b.rngSeed = int64(h.Sum64())
	return nil
}

func (b *bandwidthStressExecutor) Run(ctx context.Context, info loadgen.ScenarioInfo) error {
	if err := b.Configure(info); err != nil {
		return fmt.Errorf("failed to parse scenario configuration: %w", err)
	}
	info.Configuration.DoNotRegisterSearchAttributes = true

	executor := loadgen.KitchenSinkExecutor{
		TestInput: &TestInput{},
		UpdateWorkflowOptions: func(
			_ context.Context,
			run *loadgen.Run,
			options *loadgen.KitchenSinkWorkflowOptions,
		) error {
			options.StartOptions.TypedSearchAttributes = temporal.NewSearchAttributes()
			options.Params = b.testInput(run.Iteration)
			return nil
		},
	}
	return executor.Run(ctx, info)
}

func (b *bandwidthStressExecutor) testInput(iteration int) *TestInput {
	rng := rand.New(rand.NewSource(b.rngSeed + int64(iteration)))
	actions := make([]*Action, 0, b.activitiesPerWorkflow+1)
	for range b.activitiesPerWorkflow {
		size := int(b.payload.SamplePayloadSize(rng, bandwidthDefaultPayloadSize))
		actions = append(actions, PayloadActivity(size, size, DefaultRemoteActivity))
	}
	actions = append(actions, NewEmptyReturnResultAction())

	return &TestInput{
		WorkflowInput: &WorkflowInput{
			InitialActions: []*ActionSet{{Actions: actions}},
		},
	}
}
