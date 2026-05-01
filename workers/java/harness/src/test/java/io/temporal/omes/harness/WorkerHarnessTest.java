package io.temporal.omes.harness;

import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import io.temporal.client.WorkflowClient;
import io.temporal.common.SimplePlugin;
import io.temporal.testing.TestWorkflowEnvironment;
import io.temporal.worker.Worker;
import io.temporal.worker.WorkerFactory;
import io.temporal.worker.WorkerFactoryOptions;
import java.util.ArrayList;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CompletableFuture;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.function.BiConsumer;
import org.junit.jupiter.api.Test;

class WorkerHarnessTest {
  @Test
  void runWorkerFactoryShutsDownAllWorkersWhenStartFails() {
    LifecyclePlugin lifecycle = new LifecyclePlugin("omes-1", 2);

    try (TestWorkflowEnvironment environment = TestWorkflowEnvironment.newInstance()) {
      environment.start();
      WorkflowClient client = environment.getWorkflowClient();
      WorkerFactory workerFactory = newWorkerFactory(client, lifecycle);
      addWorker(workerFactory, "omes-1");
      addWorker(workerFactory, "omes-2");

      IllegalStateException error =
          assertThrows(
              IllegalStateException.class,
              () -> WorkerHarness.runWorkerFactory(workerFactory, client, new CountDownLatch(1)));

      assertEquals("boom", error.getMessage());
      assertEquals(Set.of("omes-1", "omes-2"), lifecycle.shutdownTaskQueues());
    }
  }

  @Test
  void runWorkerFactoryShutsDownAllWorkersWhenStopped() throws Exception {
    LifecyclePlugin lifecycle = new LifecyclePlugin(null, 2);
    CountDownLatch stopSignal = new CountDownLatch(1);

    try (TestWorkflowEnvironment environment = TestWorkflowEnvironment.newInstance()) {
      environment.start();
      WorkflowClient client = environment.getWorkflowClient();
      WorkerFactory workerFactory = newWorkerFactory(client, lifecycle);
      addWorker(workerFactory, "omes-1");
      addWorker(workerFactory, "omes-2");

      CompletableFuture<Void> runTask =
          CompletableFuture.runAsync(
              () -> runWorkerFactory(workerFactory, client, stopSignal));

      assertTrue(lifecycle.awaitStarted(5, TimeUnit.SECONDS));
      stopSignal.countDown();
      runTask.join();

      assertEquals(Set.of("omes-1", "omes-2"), lifecycle.startedTaskQueues());
      assertEquals(Set.of("omes-1", "omes-2"), lifecycle.shutdownTaskQueues());
    }
  }

  private static WorkerFactory newWorkerFactory(
      WorkflowClient client, LifecyclePlugin lifecycle) {
    return WorkerFactory.newInstance(
        client,
        WorkerFactoryOptions.newBuilder()
            .setMaxWorkflowThreadCount(4)
            .setPlugins(lifecycle)
            .build());
  }

  private static void addWorker(WorkerFactory workerFactory, String taskQueue) {
    Worker worker = workerFactory.newWorker(taskQueue);
    worker.registerWorkflowImplementationTypes(ProjectHarnessEchoWorkflowImpl.class);
  }

  private static void runWorkerFactory(
      WorkerFactory workerFactory, WorkflowClient client, CountDownLatch stopSignal) {
    try {
      WorkerHarness.runWorkerFactory(workerFactory, client, stopSignal);
    } catch (InterruptedException e) {
      Thread.currentThread().interrupt();
      throw new AssertionError(e);
    }
  }

  private static final class LifecyclePlugin extends SimplePlugin {
    private final String failOnStartTaskQueue;
    private final CountDownLatch startedWorkers;
    private final List<String> startedTaskQueues = new ArrayList<>();
    private final List<String> shutdownTaskQueues = new ArrayList<>();

    private LifecyclePlugin(String failOnStartTaskQueue, int expectedWorkerCount) {
      super("io.temporal.omes.harness-test.lifecycle");
      this.failOnStartTaskQueue = failOnStartTaskQueue;
      this.startedWorkers = new CountDownLatch(expectedWorkerCount);
    }

    @Override
    public synchronized void startWorker(
        String taskQueue, Worker worker, BiConsumer<String, Worker> next) {
      next.accept(taskQueue, worker);
      startedTaskQueues.add(taskQueue);
      startedWorkers.countDown();
      if (taskQueue.equals(failOnStartTaskQueue)) {
        throw new IllegalStateException("boom");
      }
    }

    @Override
    public synchronized void shutdownWorker(
        String taskQueue, Worker worker, BiConsumer<String, Worker> next) {
      shutdownTaskQueues.add(taskQueue);
      next.accept(taskQueue, worker);
    }

    private boolean awaitStarted(long timeout, TimeUnit unit) throws InterruptedException {
      return startedWorkers.await(timeout, unit);
    }

    private synchronized Set<String> startedTaskQueues() {
      return new HashSet<>(startedTaskQueues);
    }

    private synchronized Set<String> shutdownTaskQueues() {
      return new HashSet<>(shutdownTaskQueues);
    }
  }
}
