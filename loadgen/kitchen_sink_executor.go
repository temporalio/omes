package loadgen

import (
	"context"

	"github.com/temporalio/omes/loadgen/kitchensink"
)

type KitchenSinkExecutor struct {
	WorkflowParams        kitchensink.WorkflowParams
	PrepareWorkflowParams func(context.Context, RunOptions, *kitchensink.WorkflowParams) error
	// Should not mutate any fields in or beneath params, but only replace
	UpdateWorkflowParams func(context.Context, *Run, *kitchensink.WorkflowParams) error
	DefaultConfiguration RunConfiguration
}

func (k KitchenSinkExecutor) Run(ctx context.Context, options RunOptions) error {
	// Build base set of params
	params := k.WorkflowParams
	if k.PrepareWorkflowParams != nil {
		if err := k.PrepareWorkflowParams(ctx, options, &params); err != nil {
			return err
		}
	}
	// Create generic executor and run it
	return GenericExecutor{
		DefaultConfiguration: k.DefaultConfiguration,
		Execute: func(ctx context.Context, run *Run) error {
			params := params
			if k.UpdateWorkflowParams != nil {
				k.UpdateWorkflowParams(ctx, run, &params)
			}
			return run.ExecuteKitchenSinkWorkflow(ctx, &params)
		},
	}.Run(ctx, options)
}

func (k KitchenSinkExecutor) GetDefaultConfiguration() RunConfiguration {
	return k.DefaultConfiguration
}
