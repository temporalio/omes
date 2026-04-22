using Microsoft.Extensions.Logging;
using Temporalio.Runtime;

namespace Temporalio.Omes.Projects.Harness;

internal static class HarnessHelpers
{
    internal static ILoggerFactory ConfigureLogger(string logLevel, string logEncoding)
    {
        return Microsoft.Extensions.Logging.LoggerFactory.Create(
            builder =>
            {
                builder.ClearProviders();
                builder.SetMinimumLevel(MapLoggerLevel(logLevel));
                if (string.Equals(logEncoding, "json", StringComparison.OrdinalIgnoreCase))
                {
                    builder.AddJsonConsole();
                }
                else
                {
                    builder.AddSimpleConsole(
                        options =>
                        {
                            options.IncludeScopes = true;
                            options.SingleLine = true;
                            options.TimestampFormat = "HH:mm:ss ";
                        });
                }
            });
    }

    private static LogLevel MapLoggerLevel(string logLevel) =>
        logLevel.ToUpperInvariant() switch
        {
            "DEBUG" => LogLevel.Debug,
            "INFO" => LogLevel.Information,
            "WARN" => LogLevel.Warning,
            "ERROR" => LogLevel.Error,
            "FATAL" => LogLevel.Critical,
            "PANIC" => LogLevel.Critical,
            _ => throw new ArgumentException($"Invalid log level: {logLevel}"),
        };

    internal static TelemetryFilterOptions.Level MapTelemetryLevel(string logLevel) =>
        logLevel.ToUpperInvariant() switch
        {
            "DEBUG" => TelemetryFilterOptions.Level.Debug,
            "INFO" => TelemetryFilterOptions.Level.Info,
            "WARN" => TelemetryFilterOptions.Level.Warn,
            "ERROR" => TelemetryFilterOptions.Level.Error,
            "FATAL" => TelemetryFilterOptions.Level.Error,
            "PANIC" => TelemetryFilterOptions.Level.Error,
            _ => TelemetryFilterOptions.Level.Info,
        };
}
