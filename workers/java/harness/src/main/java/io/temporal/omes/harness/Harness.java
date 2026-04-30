package io.temporal.omes.harness;

import java.util.Arrays;
import java.util.Objects;

public final class Harness {
  private Harness() {}

  public static void run(App app, String... argv) throws Exception {
    if (argv.length == 0 || "worker".equals(argv[0])) {
      String[] workerArgv =
          argv.length == 0 ? new String[0] : Arrays.copyOfRange(argv, 1, argv.length);
      WorkerHarness.runWorkerCli(app.worker, app.clientFactory, workerArgv);
      return;
    }

    if ("project-server".equals(argv[0])) {
      if (app.project == null) {
        throw new IllegalStateException(
            "Wanted project-server but no project handlers registered for this app");
      }
      ProjectHarness.runProjectServerCli(
          app.project, app.clientFactory, Arrays.copyOfRange(argv, 1, argv.length));
      return;
    }

    throw new IllegalArgumentException(
        String.format(
            "Unknown command: [%s]. Expected 'worker' or 'project-server'", argv[0]));
  }

  public static final class App {
    public final WorkerHarness.WorkerConfigurer worker;
    public final HarnessClients.ClientFactory clientFactory;
    public final ProjectHarness.ProjectHandlers project;

    public App(
        WorkerHarness.WorkerConfigurer worker, HarnessClients.ClientFactory clientFactory) {
      this(worker, clientFactory, null);
    }

    public App(
        WorkerHarness.WorkerConfigurer worker,
        HarnessClients.ClientFactory clientFactory,
        ProjectHarness.ProjectHandlers project) {
      this.worker = Objects.requireNonNull(worker);
      this.clientFactory = Objects.requireNonNull(clientFactory);
      this.project = project;
    }
  }
}
