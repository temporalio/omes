package clioptions

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/pflag"
	"github.com/temporalio/omes/metrics"
	"go.temporal.io/api/common/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/converter"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/resolver/manual"
)

const AUTH_HEADER_ENV_VAR = "TEMPORAL_OMES_AUTH_HEADER"

var manualResolverID atomic.Int64

type headersProvider map[string]string

func (hp headersProvider) GetHeaders(ctx context.Context) (map[string]string, error) {
	return hp, nil
}

func newHeadersProvider(headers map[string]string) headersProvider {
	return headers
}

// Options for creating a Temporal client.
type ClientOptions struct {
	// Address of Temporal server to connect to
	Address string
	// Addresses of Temporal servers to connect to
	Addresses []string
	// Temporal namespace
	Namespace string
	// Enable TLS
	EnableTLS bool
	// TLS client cert
	ClientCertPath string
	// TLS client private key
	ClientKeyPath string
	// TLS server name
	TLSServerName string
	// Authorization header value
	AuthHeader string
	// Disable Host Verification
	DisableHostVerification bool

	fs *pflag.FlagSet
}

func (c *ClientOptions) ServerAddresses() []string {
	addresses := c.Addresses
	if len(addresses) == 0 {
		address := c.Address
		if address == "" {
			address = client.DefaultHostPort
		}
		addresses = []string{address}
	}

	out := make([]string, 0, len(addresses))
	for _, address := range addresses {
		address = strings.TrimSpace(address)
		if address != "" {
			out = append(out, address)
		}
	}
	return out
}

func (c *ClientOptions) ServerAddress() string {
	return strings.Join(c.ServerAddresses(), ",")
}

func (c *ClientOptions) UsesDefaultServerAddress() bool {
	addresses := c.ServerAddresses()
	return len(addresses) == 1 && addresses[0] == client.DefaultHostPort
}

func (c *ClientOptions) HostPort() (string, error) {
	addresses := c.ServerAddresses()
	if len(addresses) == 0 {
		return "", errors.New("at least one server address is required")
	}
	if len(addresses) == 1 {
		return addresses[0], nil
	}

	scheme := fmt.Sprintf("omes-manual-%d", manualResolverID.Add(1))
	builder := manual.NewBuilderWithScheme(scheme)
	resolverAddresses := make([]resolver.Address, len(addresses))
	for i, address := range addresses {
		resolverAddresses[i] = resolver.Address{Addr: address}
	}
	builder.InitialState(resolver.State{Addresses: resolverAddresses})
	resolver.Register(builder)
	return scheme + ":///temporal", nil
}

// loadTLSConfig inits a TLS config from the provided cert and key files.
func (c *ClientOptions) loadTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.DisableHostVerification,
		ServerName:         c.TLSServerName,
		MinVersion:         tls.VersionTLS13,
	}
	if c.ClientCertPath != "" {
		if c.ClientKeyPath == "" {
			return nil, errors.New("got TLS cert with no key")
		}
		cert, err := tls.LoadX509KeyPair(c.ClientCertPath, c.ClientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %s", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		return tlsConfig, nil
	} else if c.ClientKeyPath != "" {
		return nil, errors.New("got TLS key with no cert")
	}
	if c.EnableTLS {
		return tlsConfig, nil
	}
	return nil, nil
}

// MustDial connects to a Temporal server, with logging, metrics and loaded TLS certs.
func (c *ClientOptions) MustDial(metrics *metrics.Metrics, logger *zap.SugaredLogger) client.Client {
	client, err := c.Dial(metrics, logger)
	if err != nil {
		logger.Fatal(err)
	}
	return client
}

// Dial connects to a Temporal server, with logging, metrics, loaded TLS certs and set auth header.
func (c *ClientOptions) Dial(metrics *metrics.Metrics, logger *zap.SugaredLogger) (client.Client, error) {
	tlsCfg, err := c.loadTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS config: %w", err)
	}
	hostPort, err := c.HostPort()
	if err != nil {
		return nil, err
	}
	var clientOptions client.Options
	clientOptions.HostPort = hostPort
	clientOptions.Namespace = c.Namespace
	clientOptions.ConnectionOptions.TLS = tlsCfg
	clientOptions.Logger = NewZapAdapter(logger.Desugar())
	clientOptions.MetricsHandler = metrics.NewHandler()

	var authHeader string
	if c.AuthHeader == "" {
		authHeader = os.Getenv(AUTH_HEADER_ENV_VAR)
	} else {
		authHeader = c.AuthHeader
	}
	if authHeader != "" {
		clientOptions.HeadersProvider = newHeadersProvider(map[string]string{
			"Authorization": authHeader,
		})
	}

	clientOptions.DataConverter = OmesDataConverter()

	client, err := client.Dial(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	logger.Infof("Client connected to %s, namespace: %s", c.ServerAddress(), c.Namespace)
	return client, nil
}

func OmesDataConverter() converter.DataConverter {
	return converter.NewCompositeDataConverter(
		converter.NewNilPayloadConverter(),
		converter.NewByteSlicePayloadConverter(),
		&PassThroughPayloadConverter{},
		converter.NewProtoJSONPayloadConverter(),
		converter.NewProtoPayloadConverter(),
		converter.NewJSONPayloadConverter(),
	)
}

// FlagSet adds the relevant flags to populate the options struct and returns a pflag.FlagSet.
func (c *ClientOptions) FlagSet() *pflag.FlagSet {
	if c.fs != nil {
		return c.fs
	}

	c.fs = pflag.NewFlagSet("client_options", pflag.ExitOnError)
	defaultAddresses := c.Addresses
	if len(defaultAddresses) == 0 {
		address := c.Address
		if address == "" {
			address = client.DefaultHostPort
		}
		defaultAddresses = []string{address}
	}
	c.fs.StringArrayVar(&c.Addresses, "server-address", defaultAddresses, "Address of Temporal server; may be supplied multiple times")
	c.fs.StringVar(&c.Namespace, "namespace", client.DefaultNamespace, "Namespace to connect to")
	c.fs.BoolVar(&c.EnableTLS, "tls", false, "Enable TLS")
	c.fs.StringVar(&c.ClientCertPath, "tls-cert-path", "", "Path to client TLS certificate")
	c.fs.StringVar(&c.ClientKeyPath, "tls-key-path", "", "Path to client private key")
	c.fs.BoolVar(&c.DisableHostVerification, "disable-tls-host-verification", false, "Disable TLS host verification")
	c.fs.StringVar(&c.TLSServerName, "tls-server-name", "", "TLS target server name")
	c.fs.StringVar(&c.AuthHeader, "auth-header", "",
		fmt.Sprintf("Authorization header value (can also be set via %s env var)", AUTH_HEADER_ENV_VAR))
	return c.fs
}

type PassThroughPayloadConverter struct{}

func (p *PassThroughPayloadConverter) ToPayload(value interface{}) (*common.Payload, error) {
	if valuePayload, ok := value.(*common.Payload); ok {
		asBytes, err := proto.Marshal(valuePayload)
		if err != nil {
			return nil, fmt.Errorf("unable to encode raw payload: %w", err)
		}

		return &common.Payload{
			Metadata: map[string][]byte{
				"encoding": []byte(p.Encoding()),
			},
			Data: asBytes,
		}, nil
	}
	return nil, nil
}

func (p *PassThroughPayloadConverter) FromPayload(payload *common.Payload, valuePtr interface{}) error {
	innerPayload := &common.Payload{}
	err := proto.Unmarshal(payload.GetData(), innerPayload)
	if err != nil {
		return fmt.Errorf("unable to decode raw payload: %w", err)
	}
	return converter.GetDefaultDataConverter().FromPayload(innerPayload, valuePtr)
}

func (p *PassThroughPayloadConverter) ToString(payload *common.Payload) string {
	return base64.RawStdEncoding.EncodeToString(payload.GetData())
}

func (p *PassThroughPayloadConverter) Encoding() string {
	return "_passthrough"
}
