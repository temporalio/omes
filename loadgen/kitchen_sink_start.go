package loadgen

import (
	"github.com/golang/protobuf/proto"
	"github.com/temporalio/omes/loadgen/kitchensink"
)

type kitchenSinkExecution struct {
	options  KitchenSinkWorkflowOptions
	executor *kitchensink.ClientActionsExecutor
}

func freezeKitchenSinkExecution(options KitchenSinkWorkflowOptions, executor *kitchensink.ClientActionsExecutor) *kitchenSinkExecution {
	options.Params = proto.Clone(options.Params).(*kitchensink.TestInput)
	executor.WorkflowInput = options.Params.GetWorkflowInput()
	executor.RecoverOriginalStart = true
	return &kitchenSinkExecution{options: options, executor: executor}
}
