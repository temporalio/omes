package io.temporal.omes.harness;

import io.temporal.client.WorkflowClient;
import io.temporal.serviceclient.ServiceStubs;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;
import io.temporal.worker.WorkerFactoryOptions;
import io.temporal.worker.WorkerOptions;
import io.temporal.worker.tuning.PollerBehaviorAutoscaling;
import java.util.ArrayList;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;
import org.slf4j.Logger;
import picocli.CommandLine;

public final class WorkerHarness {

  @FunctionalInterface
  public interface WorkerRegistrar {
    void register(WorkflowClient client, Worker worker, WorkerContext context) throws Exception;
  }

  public static final class WorkerContext {
    public final Logger logger;
    public final String taskQueue;
    public final boolean errOnUnimplemented;

    public WorkerContext(Logger logger, String taskQueue, boolean errOnUnimplemented) {
      this.logger = logger;
      this.taskQueue = taskQueue;
      this.errOnUnimplemented = errOnUnimplemented;
    }
  }

  private WorkerHarness() {}

  static void runWorkerCli(
      WorkerRegistrar workerRegistrar, HarnessClients.ClientFactory clientFactory, String... argv)
      throws Exception {
    Arguments args = parseArguments(argv);
    Logger logger = HarnessHelpers.configure(args.logLevel, args.logEncoding);
    WorkflowClient client =
        clientFactory.create(
            HarnessClients.buildClientConfig(
                args.serverAddress,
                args.namespace,
                args.authHeader,
                args.tls,
                args.tlsCertPath,
                args.tlsKeyPath,
                null,
                false,
                args.promListenAddress,
                args.promHandlerPath));
    WorkerFactory workerFactory =
        WorkerFactory.newInstance(
            client, WorkerFactoryOptions.newBuilder().setMaxWorkflowThreadCount(1000).build());
    AtomicBoolean shutdown = new AtomicBoolean(false);
    CountDownLatch stopSignal = new CountDownLatch(1);
    Thread shutdownHook =
        new Thread(
            () -> {
              stopSignal.countDown();
              shutdownWorkersAndClient(workerFactory, client, shutdown);
            },
            "omes-java-harness-worker-shutdown");
    boolean shutdownHookAdded = false;

    try {
      registerWorkers(
          client,
          workerFactory,
          workerRegistrar,
          logger,
          buildTaskQueues(
              logger,
              args.taskQueue,
              args.taskQueueSuffixIndexStart,
              args.taskQueueSuffixIndexEnd),
          args.errOnUnimplemented,
          buildWorkerOptions(args));

      Runtime.getRuntime().addShutdownHook(shutdownHook);
      shutdownHookAdded = true;
      runWorkerFactory(workerFactory, client, stopSignal, shutdown);
    } finally {
      if (shutdownHookAdded) {
        try {
          Runtime.getRuntime().removeShutdownHook(shutdownHook);
        } catch (IllegalStateException ignored) {
          // JVM shutdown is already in progress.
        }
      }
    }
  }

  static Arguments parseArguments(String... argv) {
    Arguments args = new Arguments();
    new CommandLine(args).parseArgs(argv);
    if (args.taskQueueSuffixIndexStart > args.taskQueueSuffixIndexEnd) {
      throw new IllegalArgumentException("Task queue suffix start after end");
    }
    return args;
  }

  static void registerWorkers(
      WorkflowClient client,
      WorkerFactory workerFactory,
      WorkerRegistrar workerRegistrar,
      Logger logger,
      List<String> taskQueues,
      boolean errOnUnimplemented,
      WorkerOptions workerOptions)
      throws Exception {
    for (String taskQueue : taskQueues) {
      Worker worker = workerFactory.newWorker(taskQueue, workerOptions);
      workerRegistrar.register(
          client, worker, new WorkerContext(logger, taskQueue, errOnUnimplemented));
    }
  }

  static List<String> buildTaskQueues(
      Logger logger, String taskQueue, int suffixStart, int suffixEnd) {
    if (suffixEnd == 0) {
      logger.info("Java worker will run on task queue {}", taskQueue);
      return List.of(taskQueue);
    }

    List<String> taskQueues = new ArrayList<>(suffixEnd - suffixStart + 1);
    for (int index = suffixStart; index <= suffixEnd; index++) {
      taskQueues.add(String.format("%s-%d", taskQueue, index));
    }
    logger.info("Java worker will run on {} task queue(s)", taskQueues.size());
    return taskQueues;
  }

  static WorkerOptions buildWorkerOptions(Arguments args) {
    WorkerOptions.Builder workerOptions = WorkerOptions.newBuilder();
    if (args.workflowPollerAutoscaleMax != null) {
      workerOptions.setWorkflowTaskPollersBehavior(
          new PollerBehaviorAutoscaling(null, args.workflowPollerAutoscaleMax, null));
    } else if (args.maxConcurrentWorkflowPollers != null) {
      workerOptions.setMaxConcurrentWorkflowTaskPollers(args.maxConcurrentWorkflowPollers);
    }

    if (args.activityPollerAutoscaleMax != null) {
      workerOptions.setActivityTaskPollersBehavior(
          new PollerBehaviorAutoscaling(null, args.activityPollerAutoscaleMax, null));
    } else if (args.maxConcurrentActivityPollers != null) {
      workerOptions.setMaxConcurrentActivityTaskPollers(args.maxConcurrentActivityPollers);
    }

    if (args.maxConcurrentActivities != null) {
      workerOptions.setMaxConcurrentActivityExecutionSize(args.maxConcurrentActivities);
    }
    if (args.maxConcurrentWorkflowTasks != null) {
      workerOptions.setMaxConcurrentWorkflowTaskExecutionSize(args.maxConcurrentWorkflowTasks);
    }
    if (args.activitiesPerSecond != null) {
      workerOptions.setMaxWorkerActivitiesPerSecond(args.activitiesPerSecond);
    }
    return workerOptions.build();
  }

  static void runWorkerFactory(
      WorkerFactory workerFactory, WorkflowClient client, CountDownLatch stopSignal)
      throws InterruptedException {
    runWorkerFactory(workerFactory, client, stopSignal, new AtomicBoolean(false));
  }

  private static void runWorkerFactory(
      WorkerFactory workerFactory,
      WorkflowClient client,
      CountDownLatch stopSignal,
      AtomicBoolean shutdown)
      throws InterruptedException {
    try {
      workerFactory.start();
      stopSignal.await();
    } finally {
      shutdownWorkersAndClient(workerFactory, client, shutdown);
    }
  }

  private static void shutdownWorkersAndClient(
      WorkerFactory workerFactory, WorkflowClient client, AtomicBoolean shutdown) {
    if (!shutdown.compareAndSet(false, true)) {
      return;
    }

    workerFactory.shutdownNow();
    workerFactory.awaitTermination(5, TimeUnit.SECONDS);
    ServiceStubs<?, ?> serviceStubs = client.getWorkflowServiceStubs();
    serviceStubs.shutdownNow();
    serviceStubs.awaitTermination(5, TimeUnit.SECONDS);
  }

  static final class Arguments {
    @CommandLine.Option(
        names = {"-q", "--task-queue"},
        description = "Task queue to use",
        defaultValue = "omes")
    String taskQueue;

    @CommandLine.Option(
        names = "--task-queue-suffix-index-start",
        description = "Inclusive start for task queue suffix range",
        defaultValue = "0")
    int taskQueueSuffixIndexStart;

    @CommandLine.Option(
        names = "--task-queue-suffix-index-end",
        description = "Inclusive end for task queue suffix range",
        defaultValue = "0")
    int taskQueueSuffixIndexEnd;

    @CommandLine.Option(
        names = "--max-concurrent-activity-pollers",
        description = "Max concurrent activity pollers")
    Integer maxConcurrentActivityPollers;

    @CommandLine.Option(
        names = "--max-concurrent-workflow-pollers",
        description = "Max concurrent workflow pollers")
    Integer maxConcurrentWorkflowPollers;

    @CommandLine.Option(
        names = "--activity-poller-autoscale-max",
        description =
            "Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)")
    Integer activityPollerAutoscaleMax;

    @CommandLine.Option(
        names = "--workflow-poller-autoscale-max",
        description =
            "Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)")
    Integer workflowPollerAutoscaleMax;

    @CommandLine.Option(
        names = "--max-concurrent-activities",
        description = "Max concurrent activities")
    Integer maxConcurrentActivities;

    @CommandLine.Option(
        names = "--max-concurrent-workflow-tasks",
        description = "Max concurrent workflow tasks")
    Integer maxConcurrentWorkflowTasks;

    @CommandLine.Option(
        names = "--activities-per-second",
        description = "Per-worker activity rate limit")
    Double activitiesPerSecond;

    @CommandLine.Option(
        names = "--err-on-unimplemented",
        description =
            "Error when receiving unimplemented actions (currently only affects concurrent client actions)",
        arity = "0..1",
        fallbackValue = "true",
        defaultValue = "false",
        converter = BooleanFlagConverter.class)
    boolean errOnUnimplemented;

    @CommandLine.Option(
        names = "--log-level",
        description = "(debug info warn error panic fatal)",
        defaultValue = "info")
    String logLevel;

    @CommandLine.Option(
        names = "--log-encoding",
        description = "(console json)",
        defaultValue = "console")
    String logEncoding;

    @CommandLine.Option(
        names = {"-n", "--namespace"},
        description = "The namespace to use",
        defaultValue = "default")
    String namespace;

    @CommandLine.Option(
        names = {"-a", "--server-address"},
        description = "The host:port of the server",
        defaultValue = "localhost:7233")
    String serverAddress;

    @CommandLine.Option(
        names = "--tls",
        description = "Enable TLS",
        arity = "0..1",
        fallbackValue = "true",
        defaultValue = "false",
        converter = BooleanFlagConverter.class)
    boolean tls;

    @CommandLine.Option(
        names = "--tls-cert-path",
        description = "Path to a client cert for TLS",
        defaultValue = "")
    String tlsCertPath;

    @CommandLine.Option(
        names = "--tls-key-path",
        description = "Path to a client key for TLS",
        defaultValue = "")
    String tlsKeyPath;

    @CommandLine.Option(names = "--prom-listen-address", description = "Prometheus listen address")
    String promListenAddress;

    @CommandLine.Option(
        names = "--prom-handler-path",
        description = "Prometheus handler path",
        defaultValue = "/metrics")
    String promHandlerPath;

    @CommandLine.Option(
        names = "--auth-header",
        description = "Authorization header value",
        defaultValue = "")
    String authHeader;

    @CommandLine.Option(names = "--build-id", description = "Build ID", defaultValue = "")
    String buildId;
  }

  private static final class BooleanFlagConverter implements CommandLine.ITypeConverter<Boolean> {
    @Override
    public Boolean convert(String value) {
      if ("true".equalsIgnoreCase(value)
          || "1".equals(value)
          || "yes".equalsIgnoreCase(value)) {
        return true;
      }
      if ("false".equalsIgnoreCase(value)
          || "0".equals(value)
          || "no".equalsIgnoreCase(value)) {
        return false;
      }
      throw new IllegalArgumentException("Invalid boolean value: " + value);
    }
  }
}
