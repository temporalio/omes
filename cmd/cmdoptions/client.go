package cmdoptions

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
	Address string
	// Temporal namespace
	Namespace string
	// TLS client cert
	ClientCertPath string
	// TLS client private key
	ClientKeyPath string
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
	fs.StringVar(&c.Address, "server-address", "localhost:7233", "Address of Temporal server")
	fs.StringVar(&c.Namespace, "namespace", "default", "Namespace to connect to")
	fs.StringVar(&c.ClientCertPath, "tls-cert-path", "", "Path to client TLS certificate")
	fs.StringVar(&c.ClientKeyPath, "tls-key-path", "", "Path to client private key")
}

// ToFlags converts these options to string flags.
func (c *ClientOptions) ToFlags() (flags []string) {
	if c.Address != "" {
		flags = append(flags, "--server-address", c.Address)
	}
	if c.Namespace != "" {
		flags = append(flags, "--namespace", c.Namespace)
	}
	if c.ClientCertPath != "" {
		flags = append(flags, "--tls-cert-path", c.ClientCertPath)
	}
	if c.ClientKeyPath != "" {
		flags = append(flags, "--tls-key-path", c.ClientKeyPath)
	}
	return
}
