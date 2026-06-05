package io.temporal.omes.harness;

import io.temporal.worker.WorkerOptions;
import io.temporal.worker.tuning.ResourceBasedControllerOptions;
import io.temporal.worker.tuning.ResourceBasedTuner;
import java.util.HashMap;
import java.util.Map;

final class WorkerProfiles {
  static final String WORKER_PROFILE_ENV_VAR = "OMES_WORKER_PROFILE";
  static final String RESOURCE_BASED_DEFAULT_PROFILE = "resource-based-default";
  static final String THROUGHPUT_STRESS_BASELINE_PROFILE = "throughput-stress-baseline";

  static final class WorkerProfile {
    private final WorkerOptions workerOptions;
    private final Integer workflowCacheSize;

    private WorkerProfile(WorkerOptions workerOptions, Integer workflowCacheSize) {
      this.workerOptions = workerOptions;
      this.workflowCacheSize = workflowCacheSize;
    }

    WorkerOptions workerOptions() {
      return workerOptions;
    }

    Integer workflowCacheSize() {
      return workflowCacheSize;
    }
  }

  private static final Map<String, WorkerProfile> PROFILES = new HashMap<>();

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
    register(
        THROUGHPUT_STRESS_BASELINE_PROFILE,
        new WorkerProfile(
            WorkerOptions.newBuilder()
                .setMaxConcurrentWorkflowTaskExecutionSize(8)
                .setMaxConcurrentActivityExecutionSize(32)
                .setMaxConcurrentLocalActivityExecutionSize(32)
                .setMaxConcurrentWorkflowTaskPollers(2)
                .setMaxConcurrentActivityTaskPollers(4)
                .build(),
            50));
  }

  private WorkerProfiles() {}

  static WorkerProfile lookupWorkerProfile(String name) {
    WorkerProfile profile = PROFILES.get(name);
    if (profile == null) {
      throw new IllegalArgumentException(String.format("Unknown worker profile \"%s\"", name));
    }
    return profile;
  }

  private static void register(String name, WorkerOptions profile) {
    PROFILES.put(name, new WorkerProfile(profile, null));
  }

  private static void register(String name, WorkerProfile profile) {
    PROFILES.put(name, profile);
  }
}
