package harness

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"

	"go.temporal.io/sdk/client"
)

type staticHeadersProvider map[string]string

func (s staticHeadersProvider) GetHeaders(context.Context) (map[string]string, error) {
	return s, nil
}

// TLSOptions holds the TLS configuration for connecting to Temporal.
type TLSOptions struct {
	EnableTLS               bool
	CertPath                string
	KeyPath                 string
	ServerName              string
	DisableHostVerification bool
}

func buildClientOptions(
	serverAddress string,
	namespace string,
	authHeader string,
	tlsOpts TLSOptions,
) (client.Options, error) {
	opts := client.Options{
		HostPort:  serverAddress,
		Namespace: namespace,
	}
	if authHeader != "" {
		opts.HeadersProvider = staticHeadersProvider{
			"Authorization": authHeader,
		}
	}
	tlsCfg, err := loadTLSConfig(tlsOpts)
	if err != nil {
		return client.Options{}, fmt.Errorf("failed to load TLS config: %w", err)
	}
	if tlsCfg != nil {
		opts.ConnectionOptions.TLS = tlsCfg
	}
	return opts, nil
}

func loadTLSConfig(opts TLSOptions) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: opts.DisableHostVerification,
		ServerName:         opts.ServerName,
		MinVersion:         tls.VersionTLS13,
	}
	if opts.CertPath != "" {
		if opts.KeyPath == "" {
			return nil, errors.New("got TLS cert with no key")
		}
		cert, err := tls.LoadX509KeyPair(opts.CertPath, opts.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		return tlsConfig, nil
	} else if opts.KeyPath != "" {
		return nil, errors.New("got TLS key with no cert")
	}
	if opts.EnableTLS {
		return tlsConfig, nil
	}
	return nil, nil
}
