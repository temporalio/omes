import type { WorkerOptions } from '@temporalio/worker';

export const WORKER_PROFILE_ENV_VAR = 'OMES_WORKER_PROFILE';
export const RESOURCE_BASED_DEFAULT_PROFILE = 'resource-based-default';
export const THROUGHPUT_STRESS_BASELINE_PROFILE = 'throughput-stress-baseline';

export type WorkerProfile = Readonly<Partial<WorkerOptions>>;

const profiles = new Map<string, WorkerProfile>();

function registerWorkerProfile(name: string, profile: WorkerProfile): void {
  profiles.set(name, profile);
}

export function lookupWorkerProfile(name: string): Partial<WorkerOptions> {
  const profile = profiles.get(name);
  if (profile === undefined) {
    throw new Error(`Unknown worker profile "${name}"`);
  }
  return { ...profile };
}

registerWorkerProfile(RESOURCE_BASED_DEFAULT_PROFILE, {
  tuner: {
    tunerOptions: {
      targetMemoryUsage: 0.8,
      targetCpuUsage: 0.8,
    },
  },
});

function throughputStressProfile(scale: number): WorkerProfile {
  return {
    maxCachedWorkflows: 50 * scale,
    maxConcurrentWorkflowTaskExecutions: 8 * scale,
    maxConcurrentActivityTaskExecutions: 32 * scale,
    maxConcurrentLocalActivityExecutions: 32 * scale,
    maxConcurrentWorkflowTaskPolls: 2 * scale,
    maxConcurrentActivityTaskPolls: 4 * scale,
  };
}

registerWorkerProfile(THROUGHPUT_STRESS_BASELINE_PROFILE, throughputStressProfile(1));
// The baseline profile with every limit scaled, e.g. throughput-stress-baseline-x4.
for (const scale of [2, 4, 8, 16]) {
  registerWorkerProfile(`${THROUGHPUT_STRESS_BASELINE_PROFILE}-x${scale}`, throughputStressProfile(scale));
}
