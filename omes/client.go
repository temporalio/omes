package omes

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// Options for creating a Temporal client.
type ClientOptions struct {
	// Address of Temporal server to connect to
	Address string `flag:"server-address"`
	// Temporal namespace
	Namespace string `flag:"namespace"`
	// TLS client cert
	ClientCertPath string `flag:"tls-cert-path"`
	// TLS client private key
	ClientKeyPath string `flag:"tls-key-path"`
}

// loadTLSConfig inits a TLS config from the provided cert and key files.
func (c *ClientOptions) loadTLSConfig() (*tls.Config, error) {
	if c.ClientCertPath != "" {
		if c.ClientKeyPath == "" {
			return nil, errors.New("got TLS cert with no key")
		}
		cert, err := tls.LoadX509KeyPair(c.ClientCertPath, c.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %s", err)
		}
		return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
	} else if c.ClientKeyPath != "" {
		return nil, errors.New("got TLS key with no cert")
	}
	return nil, nil
}

// MustDial connects to a Temporal server, with logging, metrics and loaded TLS certs.
func (c *ClientOptions) MustDial(metrics *Metrics, logger *zap.SugaredLogger) client.Client {
	tlsCfg, err := c.loadTLSConfig()
	if err != nil {
		logger.Fatalf("Failed to load TLS config %s: %v", c.Address, err)
	}
	var clientOptions client.Options
	clientOptions.HostPort = c.Address
	clientOptions.Namespace = c.Namespace
	clientOptions.ConnectionOptions.TLS = tlsCfg
	clientOptions.Logger = NewZapAdapter(logger.Desugar())
	clientOptions.MetricsHandler = metrics.NewHandler()

	client, err := client.Dial(clientOptions)
	if err != nil {
		logger.Fatalf("Failed to dial %s: %v", c.Address, err)
	}
	logger.Infof("Client connected to %s, namespace: %s", c.Address, c.Namespace)
	return client
}

// AddCLIFlags adds the relevant flags to populate the options struct.
func (c *ClientOptions) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&c.Address, OptionToFlagName(c, "Address"), "a", "localhost:7233", "Address of Temporal server")
	fs.StringVarP(&c.Namespace, OptionToFlagName(c, "Namespace"), "n", "default", "Namespace to connect to")
	fs.StringVar(&c.ClientCertPath, OptionToFlagName(c, "ClientCertPath"), "", "Path to client TLS certificate")
	fs.StringVar(&c.ClientKeyPath, OptionToFlagName(c, "ClientKeyPath"), "", "Path to client private key")
}
