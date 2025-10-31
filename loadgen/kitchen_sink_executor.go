package loadgen

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type KitchenSinkExecutor struct {
	GenericExecutor

	TestInput *kitchensink.TestInput

	// Called once on start
	PrepareTestInput func(context.Context, ScenarioInfo, *kitchensink.TestInput) error

	// Called for each iteration. TestInput is copied entirely into KitchenSinkWorkflowOptions on
	// each iteration.
	UpdateWorkflowOptions func(context.Context, *Run, *KitchenSinkWorkflowOptions) error
}

func (k *KitchenSinkExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	if k.PrepareTestInput != nil {
		if err := k.PrepareTestInput(ctx, info, k.TestInput); err != nil {
			return err
		}
	}

	k.GenericExecutor.Execute = func(ctx context.Context, run *Run) error {
		options := run.DefaultKitchenSinkWorkflowOptions()
		testInputClone, ok := proto.Clone(k.TestInput).(*kitchensink.TestInput)
		if !ok {
			panic("failed to clone test input")
		}
		options.Params = testInputClone
		if k.UpdateWorkflowOptions != nil {
			err := k.UpdateWorkflowOptions(ctx, run, &options)
			if err != nil {
				return err
			}
		}
		return run.ExecuteKitchenSinkWorkflow(ctx, &options)
	}

	return k.GenericExecutor.Run(ctx, info)
}
