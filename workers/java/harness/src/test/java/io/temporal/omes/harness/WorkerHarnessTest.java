package io.temporal.omes.harness;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertSame;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import io.temporal.client.WorkflowClient;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;
import io.temporal.worker.WorkerFactoryOptions;
import io.temporal.worker.WorkerOptions;
import java.util.ArrayList;
import java.util.List;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

class WorkerHarnessTest {
  @Test
  void harnessRunDefaultsToWorkerModeWhenNoCommandIsProvided() {
    IllegalStateException error =
        assertThrows(
            IllegalStateException.class,
            () ->
                Harness.run(
                    new Harness.App(
                        (client, worker, context) -> {},
                        config -> {
                          throw new IllegalStateException("worker mode");
                        })));

    assertEquals("worker mode", error.getMessage());
  }

  @Test
  void runPassesSharedClientAndContextToEachWorkerFactory() throws Exception {
    WorkflowClient client = HarnessTestSupport.fakeWorkflowClient();
    WorkerFactory workerFactory =
        WorkerFactory.newInstance(
            client, WorkerFactoryOptions.newBuilder().setMaxWorkflowThreadCount(4).build());
    WorkerOptions workerOptions = WorkerOptions.newBuilder().build();
    Logger logger = LoggerFactory.getLogger("worker-harness-test");
    List<ConfiguredWorker> configuredWorkers = new ArrayList<>();

    WorkerHarness.WorkerConfigurer configurer =
        (configuredClient, worker, context) ->
            configuredWorkers.add(new ConfiguredWorker(configuredClient, context));

    List<Worker> workers =
        WorkerHarness.configureWorkers(
            client,
            workerFactory,
            configurer,
            logger,
            List.of("omes-1", "omes-2"),
            true,
            workerOptions);

    assertEquals(2, workers.size());
    assertEquals(2, configuredWorkers.size());
    assertSame(client, configuredWorkers.get(0).client);
    assertSame(client, configuredWorkers.get(1).client);
    assertEquals("omes-1", configuredWorkers.get(0).context.taskQueue);
    assertEquals("omes-2", configuredWorkers.get(1).context.taskQueue);
  }

  @Test
  void parseArgumentsRejectsInvalidTaskQueueRange() {
    IllegalArgumentException error =
        assertThrows(
            IllegalArgumentException.class,
            () ->
                WorkerHarness.parseArguments(
                    "--task-queue-suffix-index-start",
                    "2",
                    "--task-queue-suffix-index-end",
                    "1"));

    assertEquals("Task queue suffix start after end", error.getMessage());
  }

  @Test
  void parseArgumentsAndBuildWorkerOptionsMapCliValues() {
    WorkerHarness.Arguments args =
        WorkerHarness.parseArguments(
            "--task-queue",
            "custom",
            "--task-queue-suffix-index-start",
            "1",
            "--task-queue-suffix-index-end",
            "3",
            "--max-concurrent-activity-pollers",
            "4",
            "--max-concurrent-workflow-pollers",
            "5",
            "--max-concurrent-activities",
            "6",
            "--max-concurrent-workflow-tasks",
            "7",
            "--activities-per-second",
            "8.5",
            "--err-on-unimplemented",
            "--tls=yes");
    WorkerOptions options = WorkerHarness.buildWorkerOptions(args);

    assertEquals("custom", args.taskQueue);
    assertTrue(args.errOnUnimplemented);
    assertTrue(args.tls);
    assertEquals(4, options.getMaxConcurrentActivityTaskPollers());
    assertEquals(5, options.getMaxConcurrentWorkflowTaskPollers());
    assertEquals(6, options.getMaxConcurrentActivityExecutionSize());
    assertEquals(7, options.getMaxConcurrentWorkflowTaskExecutionSize());
    assertEquals(8.5, options.getMaxWorkerActivitiesPerSecond());
  }

  private static final class ConfiguredWorker {
    private final WorkflowClient client;
    private final WorkerHarness.WorkerContext context;

    private ConfiguredWorker(WorkflowClient client, WorkerHarness.WorkerContext context) {
      this.client = client;
      this.context = context;
    }
  }
}
