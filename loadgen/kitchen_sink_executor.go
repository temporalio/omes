package loadgen

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type KitchenSinkExecutor struct {
	TestInput *kitchensink.TestInput

	// RecoverOriginalStart retains the iteration request and acknowledged execution across retries.
	RecoverOriginalStart bool

	// Called once on start
	PrepareTestInput func(context.Context, ScenarioInfo, *kitchensink.TestInput) error

	// Called for each iteration. TestInput is copied entirely into KitchenSinkWorkflowOptions on
	// each iteration.
	UpdateWorkflowOptions func(context.Context, *Run, *KitchenSinkWorkflowOptions) error
}

func (k KitchenSinkExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	if k.PrepareTestInput != nil {
		if err := k.PrepareTestInput(ctx, info, k.TestInput); err != nil {
			return err
		}
	}
	// Create generic executor and run it
	ge := &GenericExecutor{
		Execute: func(ctx context.Context, run *Run) error {
			if !k.RecoverOriginalStart {
				options, err := k.iterationOptions(ctx, run)
				if err != nil {
					return err
				}
				return run.ExecuteKitchenSinkWorkflow(ctx, &options)
			}
			// Retries reuse the first attempt's request instead of constructing a new one.
			if run.kitchenSinkExecution == nil {
				options, err := k.iterationOptions(ctx, run)
				if err != nil {
					return err
				}
				run.kitchenSinkExecution = freezeKitchenSinkExecution(run, options)
			}
			frozen := run.kitchenSinkExecution
			return run.executeKitchenSinkWorkflow(ctx, &frozen.options, frozen.executor)
		},
	}
	return ge.Run(ctx, info)
}

func (k KitchenSinkExecutor) iterationOptions(ctx context.Context, run *Run) (KitchenSinkWorkflowOptions, error) {
	options := run.DefaultKitchenSinkWorkflowOptions()
	testInputClone, ok := proto.Clone(k.TestInput).(*kitchensink.TestInput)
	if !ok {
		panic("failed to clone test input")
	}
	options.Params = testInputClone
	if k.UpdateWorkflowOptions != nil {
		if err := k.UpdateWorkflowOptions(ctx, run, &options); err != nil {
			return options, err
		}
	}
	return options, nil
}

type kitchenSinkExecution struct {
	options  KitchenSinkWorkflowOptions
	executor *kitchensink.ClientActionsExecutor
}

func freezeKitchenSinkExecution(run *Run, options KitchenSinkWorkflowOptions) *kitchenSinkExecution {
	options.Params = proto.Clone(options.Params).(*kitchensink.TestInput)
	executor := run.newClientActionsExecutor(&options)
	executor.RecoverOriginalStart = true
	return &kitchenSinkExecution{options: options, executor: executor}
}
