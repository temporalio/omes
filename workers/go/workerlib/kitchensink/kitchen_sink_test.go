package kitchensink

import (
	"github.com/nexus-rpc/sdk-go/nexus"
	ks "github.com/temporalio/omes/loadgen/kitchensink"
	"go.temporal.io/sdk/converter"
)

var _ nexus.Operation[*ks.NexusOperationRequest, converter.RawValue] = KitchenSinkNexusOperation
