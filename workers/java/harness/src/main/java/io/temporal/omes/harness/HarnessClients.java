package io.temporal.omes.harness;

import com.sun.net.httpserver.HttpServer;
import com.uber.m3.tally.RootScopeBuilder;
import com.uber.m3.tally.Scope;
import com.uber.m3.tally.StatsReporter;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;
import io.micrometer.core.instrument.Meter;
import io.micrometer.core.instrument.config.NamingConvention;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;
import io.micrometer.prometheus.PrometheusNamingConvention;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowClientOptions;
import io.temporal.common.converter.DataConverter;
import io.temporal.common.reporter.MicrometerClientStatsReporter;
import io.temporal.serviceclient.SimpleSslContextBuilder;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.serviceclient.WorkflowServiceStubsOptions;
import java.io.FileInputStream;
import java.io.IOException;
import java.io.InputStream;
import javax.net.ssl.SSLException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

public final class HarnessClients {
  private static final String DEFAULT_PROM_HANDLER_PATH = "/metrics";
  private static final Logger logger = LoggerFactory.getLogger(HarnessClients.class);

  private HarnessClients() {}

  static ClientConfig buildClientConfig(
      String serverAddress,
      String namespace,
      String authHeader,
      boolean tls,
      String tlsCertPath,
      String tlsKeyPath,
      String tlsServerName,
      boolean disableHostVerification,
      String promListenAddress,
      String promHandlerPath) {
    if (disableHostVerification) {
      logger.warn("disable_host_verification is not supported by the Java SDK harness; ignoring");
    }

    return new ClientConfig(
        serverAddress,
        namespace,
        buildApiKey(authHeader),
        buildTlsContext(tls, tlsCertPath, tlsKeyPath),
        normalizeEmpty(promListenAddress),
        normalizePromHandlerPath(promHandlerPath),
        normalizeEmpty(tlsServerName));
  }

  public static WorkflowClient defaultClientFactory(ClientConfig config) throws Exception {
    return newWorkflowClient(config, DataConverter.getDefaultInstance());
  }

  public static WorkflowClient newWorkflowClient(ClientConfig config, DataConverter dataConverter)
      throws Exception {
    WorkflowServiceStubsOptions.Builder serviceOptionsBuilder =
        WorkflowServiceStubsOptions.newBuilder().setTarget(config.targetHost);

    if (config.tls != null) {
      serviceOptionsBuilder.setSslContext(config.tls);
      if (config.tlsServerName != null) {
        serviceOptionsBuilder.setChannelInitializer(
            channelBuilder -> channelBuilder.overrideAuthority(config.tlsServerName));
      }
    }

    if (config.apiKey != null) {
      serviceOptionsBuilder.addApiKey(() -> config.apiKey);
    }

    Scope metricsScope = maybeCreateMetricsScope(config.promListenAddress, config.promHandlerPath);
    if (metricsScope != null) {
      serviceOptionsBuilder.setMetricsScope(metricsScope);
    }

    WorkflowServiceStubs service =
        WorkflowServiceStubs.newServiceStubs(serviceOptionsBuilder.build());

    return WorkflowClient.newInstance(
        service,
        WorkflowClientOptions.newBuilder()
            .setDataConverter(dataConverter)
            .setNamespace(config.namespace)
            .build());
  }

  public interface ClientFactory {
    WorkflowClient create(ClientConfig config) throws Exception;
  }

  public static final class ClientConfig {
    public final String targetHost;
    public final String namespace;
    public final String apiKey;
    public final SslContext tls;
    final String promListenAddress;
    final String promHandlerPath;
    final String tlsServerName;

    public ClientConfig(
        String targetHost,
        String namespace,
        String apiKey,
        SslContext tls,
        String promListenAddress,
        String promHandlerPath,
        String tlsServerName) {
      this.targetHost = targetHost;
      this.namespace = namespace;
      this.apiKey = apiKey;
      this.tls = tls;
      this.promListenAddress = promListenAddress;
      this.promHandlerPath = promHandlerPath;
      this.tlsServerName = tlsServerName;
    }
  }

  private static String buildApiKey(String authHeader) {
    if (authHeader == null || authHeader.isEmpty()) {
      return null;
    }
    if (authHeader.startsWith("Bearer ")) {
      return authHeader.substring("Bearer ".length());
    }
    return authHeader;
  }

  private static SslContext buildTlsContext(boolean tls, String tlsCertPath, String tlsKeyPath) {
    String certPath = normalizeEmpty(tlsCertPath);
    String keyPath = normalizeEmpty(tlsKeyPath);

    if (certPath != null) {
      if (keyPath == null) {
        throw new IllegalArgumentException("Client cert specified, but not client key!");
      }
      try (InputStream clientCert = new FileInputStream(certPath);
          InputStream clientKey = new FileInputStream(keyPath)) {
        return SimpleSslContextBuilder.forPKCS8(clientCert, clientKey).build();
      } catch (IOException e) {
        throw new IllegalArgumentException("Unable to load TLS credentials", e);
      }
    }
    if (keyPath != null) {
      throw new IllegalArgumentException("Client key specified, but not client cert!");
    }
    if (tls) {
      try {
        return SimpleSslContextBuilder.noKeyOrCertChain().build();
      } catch (SSLException e) {
        throw new IllegalArgumentException("Unable to build TLS context", e);
      }
    }
    return null;
  }

  private static Scope maybeCreateMetricsScope(String promListenAddress, String promHandlerPath) {
    if (promListenAddress == null || promListenAddress.isEmpty()) {
      return null;
    }

    PrometheusMeterRegistry registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);
    registry
        .config()
        .namingConvention(
            new PrometheusNamingConvention() {
              @Override
              public String name(String name, Meter.Type type, String baseUnit) {
                return NamingConvention.snakeCase.name(name, type, null);
              }
            });

    StatsReporter reporter = new MicrometerClientStatsReporter(registry);
    Scope scope =
        new RootScopeBuilder()
            .reporter(reporter)
            .reportEvery(com.uber.m3.util.Duration.ofSeconds(1));
    HttpServer scrapeEndpoint =
        HarnessMetricsUtils.startPrometheusScrapeEndpoint(registry, promHandlerPath, promListenAddress);
    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> scrapeEndpoint.stop(1), "omes-java-harness-metrics-shutdown"));
    return scope;
  }

  private static String normalizePromHandlerPath(String promHandlerPath) {
    if (promHandlerPath == null || promHandlerPath.isEmpty()) {
      return DEFAULT_PROM_HANDLER_PATH;
    }
    return promHandlerPath;
  }

  private static String normalizeEmpty(String value) {
    if (value == null || value.isEmpty()) {
      return null;
    }
    return value;
  }
}
