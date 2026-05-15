package harness

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/temporalio/omes/cmd/clioptions"
	"github.com/temporalio/omes/metrics"
	sdkclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type ClientConfig struct {
	TargetHost string
	Namespace  string
	APIKey     string
	TLS        *tls.Config
	Logger     *zap.SugaredLogger
	Metrics    *metrics.Metrics
}

type ClientFactory func(ClientConfig) (sdkclient.Client, error)

type clientConfigOptions struct {
	Logger                  *zap.SugaredLogger
	ServerAddress           string
	Namespace               string
	AuthHeader              string
	EnableTLS               bool
	TLSCertPath             string
	TLSKeyPath              string
	TLSServerName           string
	DisableHostVerification bool
	PromListenAddress       string
	PromHandlerPath         string
}

// BuildSDKClientOptions converts the harness client config into Go SDK client
// options.
func BuildSDKClientOptions(config ClientConfig) sdkclient.Options {
	logger := config.Logger
	if logger == nil {
		nop := zap.NewNop()
		logger = nop.Sugar()
	}
	clientMetrics := config.Metrics
	if clientMetrics == nil {
		clientMetrics = &metrics.Metrics{
			Registry: prometheus.NewRegistry(),
			Cache:    make(map[string]any),
		}
	}
	options := sdkclient.Options{
		HostPort:       config.TargetHost,
		Namespace:      config.Namespace,
		Logger:         clioptions.NewZapAdapter(logger.Desugar()),
		MetricsHandler: clientMetrics.NewHandler(),
	}
	options.ConnectionOptions.TLS = config.TLS
	if config.APIKey != "" {
		options.Credentials = sdkclient.NewAPIKeyStaticCredentials(config.APIKey)
	}
	return options
}

func DefaultClientFactory(config ClientConfig) (sdkclient.Client, error) {
	options := BuildSDKClientOptions(config)
	client, err := sdkclient.Dial(options)
	if err != nil {
		return nil, fmt.Errorf("failed to dial: %w", err)
	}
	return client, nil
}

func buildClientConfig(opts clientConfigOptions) (ClientConfig, error) {
	logger := opts.Logger
	if logger == nil {
		nop := zap.NewNop()
		logger = nop.Sugar()
	}
	tlsConfig, err := buildTLSConfig(logger, opts)
	if err != nil {
		return ClientConfig{}, err
	}
	clientMetrics, err := buildMetrics(logger, opts.PromListenAddress, opts.PromHandlerPath)
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfig{
		TargetHost: opts.ServerAddress,
		Namespace:  opts.Namespace,
		APIKey:     buildAPIKey(opts.AuthHeader),
		TLS:        tlsConfig,
		Logger:     logger,
		Metrics:    clientMetrics,
	}, nil
}

func buildTLSConfig(logger *zap.SugaredLogger, opts clientConfigOptions) (*tls.Config, error) {
	if opts.DisableHostVerification {
		logger.Warn("disable_host_verification is not supported by the Go harness; ignoring")
	}
	tlsConfig := &tls.Config{
		ServerName: opts.TLSServerName,
	}
	if opts.TLSCertPath != "" {
		if opts.TLSKeyPath == "" {
			return nil, fmt.Errorf("Client cert specified, but not client key!")
		}
		cert, err := tls.LoadX509KeyPair(opts.TLSCertPath, opts.TLSKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load certs: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		return tlsConfig, nil
	}
	if opts.TLSKeyPath != "" {
		return nil, fmt.Errorf("Client key specified, but not client cert!")
	}
	if opts.EnableTLS {
		return tlsConfig, nil
	}
	return nil, nil
}

func buildAPIKey(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

func buildMetrics(logger *zap.SugaredLogger, listenAddress, handlerPath string) (*metrics.Metrics, error) {
	registry := prometheus.NewRegistry()
	clientMetrics := &metrics.Metrics{
		Registry: registry,
		Cache:    make(map[string]any),
	}
	if listenAddress == "" {
		return clientMetrics, nil
	}
	if handlerPath == "" {
		handlerPath = "/metrics"
	}
	mux := http.NewServeMux()
	mux.Handle(handlerPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Prometheus HTTP listener on %s: %w", listenAddress, err)
	}
	server := &http.Server{
		Addr:    listener.Addr().String(),
		Handler: mux,
	}
	clientMetrics.Server = server
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.Errorf("Prometheus HTTP server error: %v", err)
		}
	}()
	return clientMetrics, nil
}
