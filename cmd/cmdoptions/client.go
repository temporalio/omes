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
	// Enable TLS
	EnableTLS bool
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
	if c.EnableTLS {
		return &tls.Config{}, nil
	}
	return nil, nil
}

// MustDial connects to a Temporal server, with logging, metrics and loaded TLS certs.
func (c *ClientOptions) MustDial(metrics *Metrics, logger *zap.SugaredLogger) client.Client {
	client, err := c.Dial(metrics, logger)
	if err != nil {
		logger.Fatal(err)
	}
	return client
}

// Dial connects to a Temporal server, with logging, metrics and loaded TLS certs.
func (c *ClientOptions) Dial(metrics *Metrics, logger *zap.SugaredLogger) (client.Client, error) {
	tlsCfg, err := c.loadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}
	var clientOptions client.Options
	clientOptions.HostPort = c.Address
	clientOptions.Namespace = c.Namespace
	clientOptions.ConnectionOptions.TLS = tlsCfg
	clientOptions.Logger = NewZapAdapter(logger.Desugar())
	clientOptions.MetricsHandler = metrics.NewHandler()

	client, err := client.Dial(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	logger.Infof("Client connected to %s, namespace: %s", c.Address, c.Namespace)
	return client, nil
}

// AddCLIFlags adds the relevant flags to populate the options struct.
func (c *ClientOptions) AddCLIFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.Address, "server-address", client.DefaultHostPort, "Address of Temporal server")
	fs.StringVar(&c.Namespace, "namespace", client.DefaultNamespace, "Namespace to connect to")
	fs.BoolVar(&c.EnableTLS, "tls", false, "Enable TLS")
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
	if c.EnableTLS {
		flags = append(flags, "--tls")
	}
	if c.ClientCertPath != "" {
		flags = append(flags, "--tls-cert-path", c.ClientCertPath)
	}
	if c.ClientKeyPath != "" {
		flags = append(flags, "--tls-key-path", c.ClientKeyPath)
	}
	return
}
