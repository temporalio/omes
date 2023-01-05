package client

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/spf13/pflag"
	"github.com/temporalio/omes/components/metrics"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type Options struct {
	Address        string
	Namespace      string
	ClientCertPath string
	ClientKeyPath  string
}

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

// MustConnect to server, with logging and metrics
func MustConnect(options *Options, metrics *metrics.Metrics, logger *zap.SugaredLogger) client.Client {
	tlsCfg, err := loadTLSConfig(options.ClientCertPath, options.ClientKeyPath)
	if err != nil {
		logger.Fatalf("Failed to load TLS config %s: %v", options.Address, err)
	}
	var clientOptions client.Options
	clientOptions.HostPort = options.Address
	clientOptions.Namespace = options.Namespace
	clientOptions.ConnectionOptions.TLS = tlsCfg
	clientOptions.Logger = NewZapAdapter(logger.Desugar())
	clientOptions.MetricsHandler = metrics.Handler()

	c, err := client.Dial(clientOptions)
	if err != nil {
		logger.Fatalf("Failed to dial %s: %v", options.Address, err)
	}
	logger.Infof("Client connected to %s, namespace: %s", options.Address, options.Namespace)
	return c
}

func AddCLIFlags(fs *pflag.FlagSet, options *Options) {
	fs.StringVarP(&options.Address, "server-address", "a", "localhost:7233", "address of Temporal server")
	fs.StringVarP(&options.Namespace, "namespace", "n", "default", "namespace to connect to")
	fs.StringVar(&options.ClientCertPath, "tls-cert-path", "", "Path to client TLS certificate")
	fs.StringVar(&options.ClientKeyPath, "tls-key-path", "", "Path to client private key")
}
