package encryption

import (
	"context"

	"github.com/nexus-rpc/sdk-go/nexus"
)

const (
	nexusServiceName   = "encryption"
	nexusOperationName = "echo"
)

// echoOperation is a synchronous Nexus operation. Its input and output are
// serialized with the caller's and handler's data converters, so both are
// encrypted; the operation itself does nothing interesting on purpose.
var echoOperation = nexus.NewSyncOperation(nexusOperationName,
	func(_ context.Context, input NexusInput, _ nexus.StartOperationOptions) (NexusOutput, error) {
		return NexusOutput{Echoed: input.Message}, nil
	})
