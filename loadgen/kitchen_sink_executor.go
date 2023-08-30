package loadgen

import (
	"context"

	"github.com/temporalio/omes/loadgen/kitchensink"
)

type KitchenSinkExecutor struct {
	WorkflowParams kitchensink.WorkflowParams

	// Called once on start
	PrepareWorkflowParams func(context.Context, ScenarioInfo, *kitchensink.WorkflowParams) error

	// Called for each iteration. Developers must not mutate any fields in or
	// beneath Params, but only replace those fields (it is only shallow copied
	// between each execution). Therefore to add an action to an action set, an
	// entirely new slice must be created, do not append.
	UpdateWorkflowOptions func(context.Context, *Run, *KitchenSinkWorkflowOptions) error

	DefaultConfiguration RunConfiguration
}

func (k KitchenSinkExecutor) Run(ctx context.Context, info ScenarioInfo) error {
	// Build base set of params
	params := k.WorkflowParams
	if k.PrepareWorkflowParams != nil {
		if err := k.PrepareWorkflowParams(ctx, info, &params); err != nil {
			return err
		}
	}
	// Create generic executor and run it
	ge := &GenericExecutor{
		DefaultConfiguration: k.DefaultConfiguration,
		Execute: func(ctx context.Context, run *Run) error {
			options := run.DefaultKitchenSinkWorkflowOptions()
			// Shallow copies params, users are expected not to mutate any slices
			options.Params = params
			if k.UpdateWorkflowOptions != nil {
				err := k.UpdateWorkflowOptions(ctx, run, &options)
				if err != nil {
					return err
				}
			}
			return run.ExecuteKitchenSinkWorkflow(ctx, &options)
		},
	}
	return ge.Run(ctx, info)
}

func (k KitchenSinkExecutor) GetDefaultConfiguration() RunConfiguration {
	return k.DefaultConfiguration
}
