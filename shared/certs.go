package shared

import (
	"crypto/tls"
	"errors"
	"fmt"

	"go.temporal.io/sdk/client"
)

// LoadTLSConfig inits a TLS config from the provided cert and key files.
func LoadTLSConfig(clientCertPath, clientKeyPath string) (*tls.Config, error) {
	if clientCertPath != "" {
		if clientKeyPath == "" {
			return nil, errors.New("got TLS cert with no key")
		}
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %s", err)
		}
		return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
	} else if clientKeyPath != "" {
		return nil, errors.New("got TLS key with no cert")
	}
	return nil, nil
}

// LoadCertsIntoOptions loads certs from disk and populates the client options
func LoadCertsIntoOptions(options *client.Options, clientCertPath, clientKeyPath string) error {
	tlsCfg, err := LoadTLSConfig(clientCertPath, clientKeyPath)
	if err != nil {
		return err
	}
	options.ConnectionOptions.TLS = tlsCfg
	return nil
}
