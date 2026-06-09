package io.temporal.omes.apps;

import io.temporal.omes.apps.helloworld.HelloWorldApp;
import io.temporal.omes.apps.worker.WorkerApp;
import io.temporal.omes.harness.Harness;
import java.util.Arrays;
import java.util.Map;

public final class Registry {
  private static final String DEFAULT_APP_NAME = "worker";
  private static final Map<String, Harness.App> REGISTRY =
      Map.of("helloworld", HelloWorldApp.APP, "worker", WorkerApp.APP);

  private Registry() {}

  public static void main(String... args) throws Exception {
    String appName = DEFAULT_APP_NAME;
    String[] harnessArgs = args;
    if (args.length >= 2 && "--app".equals(args[0])) {
      appName = args[1];
      harnessArgs = Arrays.copyOfRange(args, 2, args.length);
    }

    Harness.App app = REGISTRY.get(appName);
    if (app == null || appName.isEmpty()) {
      throw new IllegalArgumentException(String.format("unknown Java worker app %s", appName));
    }

    Harness.run(app, harnessArgs);
  }
}
