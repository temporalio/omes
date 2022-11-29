package shared

import (
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func Connect(options client.Options, logger *zap.SugaredLogger) (client.Client, error) {
	options.Logger = NewZapAdapter(logger.Desugar())

	c, err := client.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", options.HostPort, err)
	}
	logger.Infof("Client connected to %s, namespace: %s", options.HostPort, options.Namespace)
	return c, nil
}
