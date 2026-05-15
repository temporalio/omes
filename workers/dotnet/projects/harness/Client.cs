using Microsoft.Extensions.Logging;
using Microsoft.Extensions.Logging.Abstractions;
using Temporalio.Client;
using Temporalio.Runtime;

namespace Temporalio.Omes.Projects.Harness;

public sealed record ClientConfig(
    string ServerAddress,
    string Namespace,
    string? ApiKey,
    TlsOptions? Tls,
    TemporalRuntime Runtime,
    ILoggerFactory LoggerFactory);

public delegate Task<ITemporalClient> ClientFactory(ClientConfig config);

public static class ClientHelpers
{
    public static async Task<ITemporalClient> DefaultClientFactory(ClientConfig config) =>
        await TemporalClient.ConnectAsync(
            new(config.ServerAddress)
            {
                Namespace = config.Namespace,
                ApiKey = config.ApiKey,
                Tls = config.Tls,
                Runtime = config.Runtime,
                LoggerFactory = config.LoggerFactory,
            });

    public static ClientConfig BuildClientConfig(
        string serverAddress,
        string @namespace,
        string authHeader,
        bool tls,
        string tlsCertPath,
        string tlsKeyPath,
        string? tlsServerName = null,
        bool disableHostVerification = false,
        string? promListenAddress = null,
        ILoggerFactory? loggerFactory = null)
    {
        loggerFactory ??= NullLoggerFactory.Instance;
        var logger = loggerFactory.CreateLogger("Temporalio.Omes.Projects.Harness.Client");

        return new(
            ServerAddress: serverAddress,
            Namespace: @namespace,
            ApiKey: BuildApiKey(authHeader),
            Tls: BuildTlsOptions(
                tls,
                tlsCertPath,
                tlsKeyPath,
                tlsServerName,
                disableHostVerification,
                logger),
            Runtime: BuildRuntime(promListenAddress),
            LoggerFactory: loggerFactory);
    }

    private static string? BuildApiKey(string authHeader)
    {
        if (string.IsNullOrEmpty(authHeader))
        {
            return null;
        }

        return authHeader.StartsWith("Bearer ", StringComparison.Ordinal)
            ? authHeader["Bearer ".Length..]
            : authHeader;
    }

    private static TlsOptions? BuildTlsOptions(
        bool tls,
        string tlsCertPath,
        string tlsKeyPath,
        string? tlsServerName,
        bool disableHostVerification,
        ILogger logger)
    {
        if (disableHostVerification)
        {
            logger.LogWarning("disable_host_verification is not supported by the .NET SDK; ignoring");
        }

        if (!string.IsNullOrEmpty(tlsCertPath))
        {
            if (string.IsNullOrEmpty(tlsKeyPath))
            {
                throw new ArgumentException("Client cert specified, but not client key!");
            }

            return new TlsOptions
            {
                ClientCert = File.ReadAllBytes(tlsCertPath),
                ClientPrivateKey = File.ReadAllBytes(tlsKeyPath),
                Domain = string.IsNullOrWhiteSpace(tlsServerName) ? null : tlsServerName,
            };
        }

        if (!string.IsNullOrEmpty(tlsKeyPath))
        {
            throw new ArgumentException("Client key specified, but not client cert!");
        }

        if (!tls)
        {
            return null;
        }

        return new TlsOptions { Domain = string.IsNullOrWhiteSpace(tlsServerName) ? null : tlsServerName };
    }

    private static TemporalRuntime BuildRuntime(string? promListenAddress) =>
        new(
            new TemporalRuntimeOptions
            {
                Telemetry = new TelemetryOptions
                {
                    Logging = new LoggingOptions
                    {
                        Filter = new(
                            core: HarnessHelpers.MapTelemetryLevel(Environment.GetEnvironmentVariable("TEMPORAL_CORE_LOG_LEVEL") ?? "info"),
                            other: TelemetryFilterOptions.Level.Warn),
                    },
                    Metrics = string.IsNullOrEmpty(promListenAddress)
                        ? null
                        : new MetricsOptions
                        {
                            Prometheus = new PrometheusOptions(promListenAddress)
                            {
                                UseSecondsForDuration = true,
                            },
                        },
                },
            });
}
