package io.temporal.omes;

import ch.qos.logback.classic.Level;
import ch.qos.logback.classic.Logger;
import ch.qos.logback.classic.spi.ILoggingEvent;
import ch.qos.logback.core.ConsoleAppender;
import com.sun.net.httpserver.HttpServer;
import com.uber.m3.tally.RootScopeBuilder;
import com.uber.m3.tally.Scope;
import com.uber.m3.tally.StatsReporter;
import io.grpc.netty.shaded.io.netty.handler.ssl.SslContext;
import io.micrometer.core.instrument.util.StringUtils;
import io.micrometer.prometheus.PrometheusConfig;
import io.micrometer.prometheus.PrometheusMeterRegistry;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowClientOptions;
import io.temporal.common.converter.*;
import io.temporal.common.reporter.MicrometerClientStatsReporter;
import io.temporal.serviceclient.SimpleSslContextBuilder;
import io.temporal.serviceclient.WorkflowServiceStubs;
import io.temporal.serviceclient.WorkflowServiceStubsOptions;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;
import io.temporal.worker.WorkerFactoryOptions;
import io.temporal.worker.WorkerOptions;
import io.temporal.worker.tuning.PollerBehaviorAutoscaling;
import java.io.FileInputStream;
import java.io.FileNotFoundException;
import java.io.InputStream;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import javax.net.ssl.SSLException;
import net.logstash.logback.encoder.LogstashEncoder;
import picocli.CommandLine;

@CommandLine.Command(name = "features", description = "Runs Java features")
public class Main implements Runnable {
  @CommandLine.Option(
      names = {"-q", "--task-queue"},
      description = "Task queue to use",
      defaultValue = "omes")
  private String taskQueue;

  @CommandLine.Option(
      names = "--task-queue-suffix-index-start",
      description = "Inclusive start for task queue suffix range",
      defaultValue = "0")
  private Integer taskQueueIndexStart;

  @CommandLine.Option(
      names = "--task-queue-suffix-index-end",
      description = "Inclusive end for task queue suffix range",
      defaultValue = "0")
  private Integer taskQueueIndexEnd;

  // Log arguments
  @CommandLine.Option(names = "--log-level", description = "Log level", defaultValue = "info")
  private String logLevel;

  @CommandLine.Option(
      names = "--log-encoding",
      description = "Log encoding",
      defaultValue = "console")
  private String logEncoding;

  // Client arguments
  @CommandLine.Option(
      names = {"-n", "--namespace"},
      description = "The namespace to use",
      defaultValue = "default")
  private String namespace;

  @CommandLine.Option(
      names = {"-a", "--server-address"},
      description = "The host:port of the server",
      defaultValue = "localhost:7233")
  private String serverAddress;

  @CommandLine.Option(names = "--tls", description = "Enable TLS")
  private boolean isTlsEnabled;

  @CommandLine.Option(names = "--tls-cert-path", description = "Path to a client cert for TLS")
  private String clientCertPath;

  @CommandLine.Option(names = "--tls-key-path", description = "Path to a client key for TLS")
  private String clientKeyPath;

  // Metric parameters
  @CommandLine.Option(
      names = "--prom-listen-address",
      description = "Prometheus listen address",
      defaultValue = "localhost")
  private String promListenAddress;

  @CommandLine.Option(
      names = "--prom-handler-path",
      description = "Prometheus handler path",
      defaultValue = "/metrics")
  private String promHandlerPath;

  // Worker parameters
  @CommandLine.Option(
      names = "--max-concurrent-activity-pollers",
      description = "Max concurrent activity pollers")
  private int maxConcurrentActivityPollers;

  @CommandLine.Option(
      names = "--max-concurrent-workflow-pollers",
      description = "Max concurrent workflow pollers")
  private int maxConcurrentWorkflowPollers;

  @CommandLine.Option(
      names = "--activity-poller-autoscale-max",
      description = "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)")
  private int activityPollerAutoscaleMax;

  @CommandLine.Option(
      names = "--workflow-poller-autoscale-max",
      description = "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)")
  private int workflowPollerAutoscaleMax;

  @CommandLine.Option(
      names = "--max-concurrent-activities",
      description = "Max concurrent activities")
  private int maxConcurrentActivities;

  @CommandLine.Option(
      names = "--max-concurrent-workflow-tasks",
      description = "Max concurrent workflow tasks")
  private int maxConcurrentWorkflowTasks;

  @Override
  public void run() {
    // Configure TLS
    SslContext sslContext = null;
    if (StringUtils.isNotEmpty(clientCertPath)) {
      if (StringUtils.isEmpty(clientKeyPath)) {
        throw new RuntimeException("Client key path must be specified since cert path is");
      }

      try {
        InputStream clientCert = new FileInputStream(clientCertPath);
        InputStream clientKey = new FileInputStream(clientKeyPath);
        sslContext = SimpleSslContextBuilder.forPKCS8(clientCert, clientKey).build();
      } catch (FileNotFoundException | SSLException e) {
        throw new RuntimeException("Error loading certs", e);
      }

    } else if (StringUtils.isNotEmpty(clientKeyPath) && StringUtils.isEmpty(clientCertPath)) {
      throw new RuntimeException("Client cert path must be specified since key path is");
    } else if (isTlsEnabled) {
      try {
        sslContext = SimpleSslContextBuilder.noKeyOrCertChain().build();
      } catch (SSLException e) {
        throw new RuntimeException(e);
      }
    }

    // Configure logging
    Logger logger =
        (Logger) org.slf4j.LoggerFactory.getLogger(ch.qos.logback.classic.Logger.ROOT_LOGGER_NAME);
    logger.setLevel(Level.valueOf(logLevel));
    if (logEncoding == "json") {
      ConsoleAppender appender = new ConsoleAppender<ILoggingEvent>();
      LogstashEncoder encoder = new LogstashEncoder();
      appender.setEncoder(encoder);
      logger.addAppender(appender);
    }
    // Configure metrics
    PrometheusMeterRegistry registry = new PrometheusMeterRegistry(PrometheusConfig.DEFAULT);
    StatsReporter reporter = new MicrometerClientStatsReporter(registry);
    // set up a new scope, report every 10 seconds
    Scope scope =
        new RootScopeBuilder()
            .reporter(reporter)
            .reportEvery(com.uber.m3.util.Duration.ofSeconds(1));
    // Start the prometheus scrape endpoint for starter metrics
    HttpServer scrapeEndpoint =
        MetricsUtils.startPrometheusScrapeEndpoint(registry, promHandlerPath, promListenAddress);
    // Stopping the starter will stop the http server that exposes the
    // scrape endpoint.
    Runtime.getRuntime().addShutdownHook(new Thread(() -> scrapeEndpoint.stop(1)));
    // Configure client
    WorkflowServiceStubs service =
        WorkflowServiceStubs.newServiceStubs(
            WorkflowServiceStubsOptions.newBuilder()
                .setTarget(serverAddress)
                .setSslContext(sslContext)
                .setMetricsScope(scope)
                .build());

    PayloadConverter[] arr = {
      new NullPayloadConverter(),
      new ByteArrayPayloadConverter(),
      new PassthroughDataConverter(),
      new ProtobufJsonPayloadConverter(),
      new ProtobufPayloadConverter(),
      new JacksonJsonPayloadConverter()
    };

    WorkflowClient client =
        WorkflowClient.newInstance(
            service,
            WorkflowClientOptions.newBuilder()
                .setDataConverter(new DefaultDataConverter(arr))
                .setNamespace(namespace)
                .build());

    // Collect task queues to run workers for (if there is a suffix end, we run multiple)
    List<String> taskQueues;
    if (taskQueueIndexStart == 0) {
      taskQueues = Collections.singletonList(taskQueue);
    } else {
      taskQueues = new ArrayList<>(taskQueueIndexEnd - taskQueueIndexStart);
      for (int i = taskQueueIndexStart; i <= taskQueueIndexEnd; i++) {
        taskQueues.add(String.format("%s-%d", taskQueue, i));
      }
    }
    // Create worker factory
    WorkerFactory workerFactory =
        WorkerFactory.newInstance(
            client, WorkerFactoryOptions.newBuilder().setMaxWorkflowThreadCount(1000).build());
    // Create the base worker options
    WorkerOptions.Builder workerOptions = WorkerOptions.newBuilder();
    // Workflow options
    if (workflowPollerAutoscaleMax > 0) {
      workerOptions.setWorkflowTaskPollerBehavior(
          PollerBehaviorAutoscaling.newBuilder()
              .setMaximumNumberOfPollers(workflowPollerAutoscaleMax)
              .build());
    } else if (maxConcurrentWorkflowPollers > 0) {
      workerOptions.setMaxConcurrentWorkflowTaskPollers(maxConcurrentWorkflowPollers);
    }
    workerOptions.setMaxConcurrentWorkflowTaskExecutionSize(maxConcurrentWorkflowTasks);
    // Activity options
    if (activityPollerAutoscaleMax > 0) {
      workerOptions.setActivityTaskPollerBehavior(
          PollerBehaviorAutoscaling.newBuilder()
              .setMaximumNumberOfPollers(activityPollerAutoscaleMax)
              .build());
    } else if (maxConcurrentActivityPollers > 0) {
      workerOptions.setMaxConcurrentActivityTaskPollers(maxConcurrentActivityPollers);
    }
    workerOptions.setMaxConcurrentActivityExecutionSize(maxConcurrentActivities);
    // Start all workers, throwing on first exception
    for (String taskQueue : taskQueues) {
      Worker worker = workerFactory.newWorker(taskQueue, workerOptions.build());
      worker.registerWorkflowImplementationTypes(KitchenSinkWorkflowImpl.class);
      worker.registerActivitiesImplementations(new ActivitiesImpl(client));
    }
    workerFactory.start();
    CountDownLatch latch = new CountDownLatch(1);

    Runtime.getRuntime()
        .addShutdownHook(
            new Thread(
                () -> {
                  scrapeEndpoint.stop(1);
                  // Shut all workers down
                  workerFactory.shutdownNow();
                  latch.countDown();
                }));
    try {
      latch.await();
      System.exit(0);
    } catch (InterruptedException e) {
      throw new RuntimeException(e);
    }
  }

  public static void main(String... args) {
    System.exit(new CommandLine(new Main()).execute(args));
  }
}
