package harness

import (
	"context"

	"go.temporal.io/sdk/client"
)

type staticHeadersProvider map[string]string

func (s staticHeadersProvider) GetHeaders(context.Context) (map[string]string, error) {
	return s, nil
}

func buildClientOptions(
	serverAddress string,
	namespace string,
	authHeader string,
) client.Options {
	opts := client.Options{
		HostPort:  serverAddress,
		Namespace: namespace,
	}
	if authHeader != "" {
		opts.HeadersProvider = staticHeadersProvider{
			"Authorization": authHeader,
		}
	}
	return opts
}
