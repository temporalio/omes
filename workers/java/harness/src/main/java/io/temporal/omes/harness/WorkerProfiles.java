package io.temporal.omes.harness;

import io.temporal.worker.WorkerOptions;
import io.temporal.worker.tuning.ResourceBasedControllerOptions;
import io.temporal.worker.tuning.ResourceBasedTuner;
import java.util.HashMap;
import java.util.Map;

final class WorkerProfiles {
  static final String WORKER_PROFILE_ENV_VAR = "OMES_WORKER_PROFILE";
  static final String RESOURCE_BASED_DEFAULT_PROFILE = "resource-based-default";

  private static final Map<String, WorkerOptions> PROFILES = new HashMap<>();

  static {
    register(
        RESOURCE_BASED_DEFAULT_PROFILE,
        WorkerOptions.newBuilder()
            .setWorkerTuner(
                ResourceBasedTuner.newBuilder()
                    .setControllerOptions(
                        ResourceBasedControllerOptions.newBuilder(0.8, 0.8).build())
                    .build())
            .build());
  }

  private WorkerProfiles() {}

  static WorkerOptions lookupWorkerProfile(String name) {
    WorkerOptions profile = PROFILES.get(name);
    if (profile == null) {
      throw new IllegalArgumentException(String.format("Unknown worker profile \"%s\"", name));
    }
    return profile;
  }

  private static void register(String name, WorkerOptions profile) {
    PROFILES.put(name, profile);
  }
}
