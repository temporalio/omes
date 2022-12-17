package shared

import (
	"crypto/tls"
	"errors"
	"fmt"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// loadTLSConfig inits a TLS config from the provided cert and key files.
func loadTLSConfig(clientCertPath, clientKeyPath string) (*tls.Config, error) {
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
	tlsCfg, err := loadTLSConfig(clientCertPath, clientKeyPath)
	if err != nil {
		return err
	}
	options.ConnectionOptions.TLS = tlsCfg
	return nil
}

// Connect to server, with logging and metrics
func Connect(options client.Options, logger *zap.SugaredLogger) (client.Client, error) {
	options.Logger = NewZapAdapter(logger.Desugar())
	options.MetricsHandler = NewMetricsHandler()

	c, err := client.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %v", options.HostPort, err)
	}
	logger.Infof("Client connected to %s, namespace: %s", options.HostPort, options.Namespace)
	return c, nil
}
